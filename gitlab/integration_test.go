/*
Copyright 2020 The Flux CD contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package gitlab

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gregjones/httpcache"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/xanzy/go-gitlab"

	"github.com/fluxcd/go-git-providers/gitprovider"
	testutils "github.com/fluxcd/go-git-providers/gitprovider/testutils"
)

const (
	ghTokenFile = "/tmp/gitlab-token"

	// Include scheme if custom, e.g.:
	// gitlabDomain = "https://gitlab.acme.org/"
	// gitlabDomain = "https://gitlab.dev.wkp.weave.works"
	gitlabDomain = "gitlab.com"

	defaultDescription = "Foo description"
	defaultBranch      = "main"
)

var (
	// customTransportImpl is a shared instance of a customTransport, allowing counting of cache hits.
	customTransportImpl *customTransport
)

func init() {
	// Call testing.Init() prior to tests.NewParams(), as otherwise -test.* will not be recognised. See also: https://golang.org/doc/go1.13#testing
	testing.Init()
	rand.Seed(time.Now().UnixNano())
}

func TestProvider(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "GitLab Provider Suite")
}

func customTransportFactory(transport http.RoundTripper) http.RoundTripper {
	if customTransportImpl != nil {
		panic("didn't expect this function to be called twice")
	}
	customTransportImpl = &customTransport{
		transport:      transport,
		countCacheHits: false,
		cacheHits:      0,
		mux:            &sync.Mutex{},
	}
	return customTransportImpl
}

type customTransport struct {
	transport      http.RoundTripper
	countCacheHits bool
	cacheHits      int
	mux            *sync.Mutex
}

func getBodyFromReaderWithoutConsuming(r *io.ReadCloser) string {
	body, _ := io.ReadAll(*r)
	(*r).Close()
	*r = io.NopCloser(bytes.NewBuffer(body))
	return string(body)
}

const (
	ConnectionResetByPeer    = "connection reset by peer"
	ProjectStillBeingDeleted = "The project is still being deleted"
)

func (t *customTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	t.mux.Lock()
	defer t.mux.Unlock()

	var resp *http.Response
	var err error
	var responseBody string
	var requestBody string
	retryCount := 3
	for retryCount != 0 {
		responseBody = ""
		requestBody = ""
		if req != nil && req.Body != nil {
			requestBody = getBodyFromReaderWithoutConsuming(&req.Body)
		}
		resp, err = t.transport.RoundTrip(req)
		if resp != nil && resp.Body != nil {
			responseBody = getBodyFromReaderWithoutConsuming(&resp.Body)
		}
		if (err != nil && (strings.Contains(err.Error(), ConnectionResetByPeer))) ||
			strings.Contains(string(responseBody), ProjectStillBeingDeleted) {
			time.Sleep(2 * time.Second)
			if req != nil && req.Body != nil {
				req.Body = io.NopCloser(strings.NewReader(requestBody))
			}
			retryCount--
			continue
		}
		break
	}
	// If we should count, count all cache hits whenever found
	if t.countCacheHits {
		if _, ok := resp.Header[httpcache.XFromCache]; ok {
			t.cacheHits++
		}
	}
	return resp, err
}

func (t *customTransport) resetCounter() {
	t.mux.Lock()
	defer t.mux.Unlock()

	t.cacheHits = 0
}

func (t *customTransport) setCounter(state bool) {
	t.mux.Lock()
	defer t.mux.Unlock()

	t.countCacheHits = state
}

func (t *customTransport) getCacheHits() int {
	t.mux.Lock()
	defer t.mux.Unlock()

	return t.cacheHits
}

func (t *customTransport) countCacheHitsForFunc(fn func()) int {
	t.setCounter(true)
	t.resetCounter()
	fn()
	t.setCounter(false)
	return t.getCacheHits()
}

var _ = Describe("GitLab Provider", func() {
	var (
		ctx context.Context = context.Background()
		c   gitprovider.Client

		// Should exist in environment
		testOrgName      string = "fluxcd-testing"
		testSubgroupName string = "fluxcd-testing-sub-group"
		testTeamName     string = "fluxcd-testing-2"
		testUserName     string = "fluxcd-gitprovider-bot"

		// placeholders, will be randomized and created.
		testSharedOrgRepoName string = "testsharedorgrepo"
		testOrgRepoName       string = "testorgrepo"
		testRepoName          string = "testrepo"
	)

	BeforeSuite(func() {
		gitlabToken := os.Getenv("GITLAB_TOKEN")
		if len(gitlabToken) == 0 {
			b, err := os.ReadFile(ghTokenFile)
			if token := string(b); err == nil && len(token) != 0 {
				gitlabToken = token
			} else {
				Fail("couldn't acquire GITLAB_TOKEN env variable")
			}
		}

		if orgName := os.Getenv("GIT_PROVIDER_ORGANIZATION"); len(orgName) != 0 {
			testOrgName = orgName
		}

		if subGroupName := os.Getenv("GITLAB_TEST_SUBGROUP"); len(subGroupName) != 0 {
			testSubgroupName = subGroupName
		}

		if teamName := os.Getenv("GITLAB_TEST_TEAM_NAME"); len(teamName) != 0 {
			testTeamName = teamName
		}

		if gitProviderUser := os.Getenv("GIT_PROVIDER_USER"); len(gitProviderUser) != 0 {
			testUserName = gitProviderUser
		}

		var err error
		c, err = NewClient(
			gitlabToken, "",
			gitprovider.WithDomain(gitlabDomain),
			gitprovider.WithDestructiveAPICalls(true),
			gitprovider.WithConditionalRequests(true),
			gitprovider.WithPreChainTransportHook(customTransportFactory),
		)
		Expect(err).ToNot(HaveOccurred())
	})

	validateOrgRepo := func(repo gitprovider.OrgRepository, expectedRepoRef gitprovider.RepositoryRef) {
		info := repo.Get()
		fmt.Fprintf(os.Stderr, "validating repo: %s\n", repo.Repository().GetRepository())
		// Expect certain fields to be set
		Expect(repo.Repository()).To(Equal(expectedRepoRef))
		Expect(*info.Description).To(Equal(defaultDescription))
		Expect(*info.Visibility).To(Equal(gitprovider.RepositoryVisibilityPrivate))
		Expect(*info.DefaultBranch).To(Equal(defaultBranchName))
		// Expect high-level fields to match their underlying data
		internal := repo.APIObject().(*gitlab.Project)
		Expect(repo.Repository().GetRepository()).To(Equal(internal.Name))
		Expect(repo.Repository().GetIdentity()).To(Equal(testOrgName))
		Expect(*info.Description).To(Equal(internal.Description))
		Expect(string(*info.Visibility)).To(Equal(string(internal.Visibility)))
		Expect(*info.DefaultBranch).To(Equal(internal.DefaultBranch))
	}

	validateUserRepo := func(repo gitprovider.UserRepository, expectedRepoRef gitprovider.RepositoryRef) {
		info := repo.Get()
		// Expect certain fields to be set
		Expect(repo.Repository()).To(Equal(expectedRepoRef))
		Expect(*info.Description).To(Equal(defaultDescription))
		Expect(*info.Visibility).To(Equal(gitprovider.RepositoryVisibilityPrivate))
		Expect(*info.DefaultBranch).To(Equal(defaultBranchName))
		// Expect high-level fields to match their underlying data
		internal := repo.APIObject().(*gitlab.Project)
		Expect(repo.Repository().GetRepository()).To(Equal(internal.Name))
		Expect(repo.Repository().GetIdentity()).To(Equal(testUserName))
		Expect(*info.Description).To(Equal(internal.Description))
		Expect(string(*info.Visibility)).To(Equal(string(internal.Visibility)))
		Expect(*info.DefaultBranch).To(Equal(internal.DefaultBranch))
	}

	cleanupOrgRepos := func(prefix string) {
		fmt.Fprintf(os.Stderr, "Deleting repos starting with %s in org: %s\n", prefix, testOrgName)
		repos, err := c.OrgRepositories().List(ctx, newOrgRef(testOrgName))
		Expect(err).ToNot(HaveOccurred())
		for _, repo := range repos {
			// Delete the test org repo used
			name := repo.Repository().GetRepository()
			if !strings.HasPrefix(name, prefix) {
				continue
			}
			fmt.Fprintf(os.Stderr, "Deleting the org repo: %s\n", name)
			repo.Delete(ctx)
			Expect(err).ToNot(HaveOccurred())
		}
	}

	cleanupUserRepos := func(prefix string) {
		fmt.Fprintf(os.Stderr, "Deleting repos starting with %s for user: %s\n", prefix, testUserName)
		repos, err := c.UserRepositories().List(ctx, newUserRef(testUserName))
		Expect(err).ToNot(HaveOccurred())
		for _, repo := range repos {
			// Delete the test org repo used
			name := repo.Repository().GetRepository()
			if !strings.HasPrefix(name, prefix) {
				continue
			}
			fmt.Fprintf(os.Stderr, "Deleting the org repo: %s\n", name)
			repo.Delete(ctx)
			Expect(err).ToNot(HaveOccurred())
		}
	}

	It("should list the available organizations the user has access to", func() {
		// Get a list of all organizations the user is part of
		orgs, err := c.Organizations().List(ctx)
		Expect(err).ToNot(HaveOccurred())

		// Make sure we find the expected one given as testOrgName
		var listedOrg, getOrg gitprovider.Organization
		for _, org := range orgs {
			if org.Organization().Organization == testOrgName {
				listedOrg = org
				break
			}
		}
		Expect(listedOrg).ToNot(BeNil())

		hits := customTransportImpl.countCacheHitsForFunc(func() {
			// Do a GET call for that organization
			getOrg, err = c.Organizations().Get(ctx, listedOrg.Organization())
			Expect(err).ToNot(HaveOccurred())
		})
		// don't expect any cache hit, as we didn't request this before
		Expect(hits).To(Equal(0))

		// Expect that the organization's info is the same regardless of method
		Expect(getOrg.Organization()).To(Equal(listedOrg.Organization()))

		Expect(listedOrg.Get().Name).ToNot(BeNil())
		Expect(listedOrg.Get().Description).ToNot(BeNil())
		// We expect the name and description to be populated
		// in the GET call. Note: This requires the user to set up
		// the given organization with a name and description in the UI :)
		Expect(getOrg.Get().Name).ToNot(BeNil())
		Expect(getOrg.Get().Description).ToNot(BeNil())
		// Expect Name and Description to match their underlying data
		internal := getOrg.APIObject().(*gitlab.Group)
		derefOrgName := *getOrg.Get().Name
		Expect(derefOrgName).To(Equal(internal.Name))
		derefOrgDescription := *getOrg.Get().Description
		Expect(derefOrgDescription).To(Equal(internal.Description))

		// Expect that when we do the same request a second time, it will hit the cache
		hits = customTransportImpl.countCacheHitsForFunc(func() {
			getOrg2, err := c.Organizations().Get(ctx, listedOrg.Organization())
			Expect(err).ToNot(HaveOccurred())
			Expect(getOrg2).ToNot(BeNil())
		})
		Expect(hits).To(Equal(1))
	})

	It("should not fail when .Children is called", func() {
		_, err := c.Organizations().Children(ctx, gitprovider.OrganizationRef{
			Domain:       gitlabDomain,
			Organization: testOrgName,
		})
		Expect(err).ToNot(HaveOccurred())
	})

	It("should be possible to create a group project", func() {
		// First, check what repositories are available
		repos, err := c.OrgRepositories().List(ctx, newOrgRef(testOrgName))
		Expect(err).ToNot(HaveOccurred())

		// Generate a repository name which doesn't exist already
		testOrgRepoName = fmt.Sprintf("test-org-repo-%03d", rand.Intn(1000))
		for findOrgRepo(repos, testOrgRepoName) != nil {
			testOrgRepoName = fmt.Sprintf("test-org-repo-%03d", rand.Intn(1000))
		}

		// We know that a repo with this name doesn't exist in the organization, let's verify we get an
		// ErrNotFound
		repoRef := newOrgRepoRef(testOrgName, testOrgRepoName)
		sshURL := repoRef.GetCloneURL(gitprovider.TransportTypeSSH)
		Expect(sshURL).NotTo(Equal(""))
		_, err = c.OrgRepositories().Get(ctx, repoRef)
		Expect(errors.Is(err, gitprovider.ErrNotFound)).To(BeTrue())

		// Create a new repo
		repo, err := c.OrgRepositories().Create(ctx, repoRef, gitprovider.RepositoryInfo{
			Description: gitprovider.StringVar(defaultDescription),
			// Default visibility is private, no need to set this at least now
			//Visibility:     gitprovider.RepositoryVisibilityVar(gitprovider.RepositoryVisibilityPrivate),
		}, &gitprovider.RepositoryCreateOptions{
			AutoInit:        gitprovider.BoolVar(true),
			LicenseTemplate: gitprovider.LicenseTemplateVar(gitprovider.LicenseTemplateApache2),
		})
		Expect(err).ToNot(HaveOccurred())

		validateOrgRepo(repo, repoRef)

		getRepo, err := c.OrgRepositories().Get(ctx, repoRef)
		Expect(err).ToNot(HaveOccurred())
		// Expect the two responses (one from POST and one from GET to have equal "spec")
		getSpec := newGitlabProjectSpec(getRepo.APIObject().(*gitlab.Project))
		postSpec := newGitlabProjectSpec(repo.APIObject().(*gitlab.Project))
		Expect(getSpec.Equals(postSpec)).To(BeTrue())
	})

	It("should error at creation time if the org repo already does exist", func() {
		repoRef := newOrgRepoRef(testOrgName, testOrgRepoName)
		_, err := c.OrgRepositories().Create(ctx, repoRef, gitprovider.RepositoryInfo{})
		Expect(errors.Is(err, gitprovider.ErrAlreadyExists)).To(BeTrue())
	})

	It("should update if the org repo already exists when reconciling", func() {
		repoRef := newOrgRepoRef(testOrgName, testOrgRepoName)
		// No-op reconcile
		resp, actionTaken, err := c.OrgRepositories().Reconcile(ctx, repoRef, gitprovider.RepositoryInfo{
			Description:   gitprovider.StringVar(defaultDescription),
			DefaultBranch: gitprovider.StringVar(defaultBranch),
			Visibility:    gitprovider.RepositoryVisibilityVar(gitprovider.RepositoryVisibilityPrivate),
		})
		Expect(err).ToNot(HaveOccurred())
		Expect(actionTaken).To(BeFalse())
		// no-op set & reconcile
		Expect(resp.Set(resp.Get())).ToNot(HaveOccurred())
		actionTaken, err = resp.Reconcile(ctx)
		Expect(err).ToNot(HaveOccurred())
		Expect(actionTaken).To(BeFalse())

		// Update reconcile
		newDesc := "New description"
		req := resp.Get()
		req.Description = gitprovider.StringVar(newDesc)
		Expect(resp.Set(req)).ToNot(HaveOccurred())
		actionTaken, err = resp.Reconcile(ctx)
		// Expect the update to succeed, and modify the state
		Expect(err).ToNot(HaveOccurred())
		Expect(actionTaken).To(BeTrue())
		Expect(*resp.Get().Description).To(Equal(newDesc))

		// Delete the repository and later re-create
		Expect(resp.Delete(ctx)).ToNot(HaveOccurred())

		var newRepo gitprovider.OrgRepository
		retryOp := testutils.NewRetry()
		Eventually(func() bool {
			var err error
			// Reconcile and create
			newRepo, actionTaken, err = c.OrgRepositories().Reconcile(ctx, repoRef, gitprovider.RepositoryInfo{
				Description: gitprovider.StringVar(defaultDescription),
			}, &gitprovider.RepositoryCreateOptions{
				AutoInit:        gitprovider.BoolVar(true),
				LicenseTemplate: gitprovider.LicenseTemplateVar(gitprovider.LicenseTemplateMIT),
			})
			return retryOp.IsRetryable(err, fmt.Sprintf("reconcile org repository: %s", repoRef.RepositoryName))
		}, retryOp.Timeout(), retryOp.Interval()).Should(BeTrue())

		Expect(actionTaken).To(BeTrue())
		validateOrgRepo(newRepo, repoRef)
	})

	It("should update teams with access and permissions when reconciling", func() {

		// Get the test organization
		orgRef := newOrgRef(testOrgName)
		testOrg, err := c.Organizations().Get(ctx, orgRef)
		Expect(err).ToNot(HaveOccurred())

		// List all the teams with access to the org
		// There should be 1 existing subgroup already
		teams, err := testOrg.Teams().List(ctx)
		Expect(err).ToNot(HaveOccurred())
		Expect(len(teams)).To(Equal(1), "The 1 team wasn't there...")

		// First, check what repositories are available
		repos, err := c.OrgRepositories().List(ctx, newOrgRef(testOrgName))
		Expect(err).ToNot(HaveOccurred())

		// Generate an org repo name which doesn't exist already
		testSharedOrgRepoName = fmt.Sprintf("test-shared-org-repo-%03d", rand.Intn(1000))
		for findOrgRepo(repos, testSharedOrgRepoName) != nil {
			testSharedOrgRepoName = fmt.Sprintf("test-shared-org-repo-%03d", rand.Intn(1000))
		}

		// We know that a repo with this name doesn't exist in the organization, let's verify we get an
		// ErrNotFound
		repoRef := newOrgRepoRef(testOrgName, testSharedOrgRepoName)
		_, err = c.OrgRepositories().Get(ctx, repoRef)
		Expect(errors.Is(err, gitprovider.ErrNotFound)).To(BeTrue())

		// Create a new repo
		repo, err := c.OrgRepositories().Create(ctx, repoRef, gitprovider.RepositoryInfo{
			Description: gitprovider.StringVar(defaultDescription),
			// Default visibility is private, no need to set this at least now
			//Visibility:     gitprovider.RepositoryVisibilityVar(gitprovider.RepositoryVisibilityPrivate),
		}, &gitprovider.RepositoryCreateOptions{
			AutoInit:        gitprovider.BoolVar(true),
			LicenseTemplate: gitprovider.LicenseTemplateVar(gitprovider.LicenseTemplateApache2),
		})
		Expect(err).ToNot(HaveOccurred())

		validateOrgRepo(repo, repoRef)

		// No groups should have access to the repo at this point
		projectTeams, err := repo.TeamAccess().List(ctx)
		Expect(err).ToNot(HaveOccurred())
		Expect(len(projectTeams)).To(Equal(0))

		// Add a team to the project
		permission := gitprovider.RepositoryPermissionMaintain
		_, err = repo.TeamAccess().Create(ctx, gitprovider.TeamAccessInfo{
			Name:       testTeamName,
			Permission: &permission,
		})
		Expect(err).ToNot(HaveOccurred())

		// List all the teams with access to the project
		// Only
		projectTeams, err = repo.TeamAccess().List(ctx)
		Expect(err).ToNot(HaveOccurred())
		Expect(len(projectTeams)).To(Equal(1), "Project teams didn't equal 1")
		firstTeam := projectTeams[0]
		Expect(firstTeam.Get().Name).To(Equal(testTeamName))

		// Update the permission level and update
		ta, err := repo.TeamAccess().Get(ctx, testTeamName)
		Expect(err).ToNot(HaveOccurred())

		// Set permission level to Push and call Reconcile
		pushPermission := gitprovider.RepositoryPermissionPush
		pushTeamAccess := ta
		pushTeamAccessInfo := pushTeamAccess.Get()
		pushTeamAccessInfo.Permission = &pushPermission
		ta.Set(pushTeamAccessInfo)
		actionTaken, err := ta.Reconcile(ctx)
		Expect(err).ToNot(HaveOccurred())
		Expect(actionTaken).To(Equal(true))

		// Get the team access info again and verify it has been updated
		updatedTA, err := repo.TeamAccess().Get(ctx, testTeamName)
		Expect(err).ToNot(HaveOccurred())
		Expect(*updatedTA.Get().Permission).To(Equal(gitprovider.RepositoryPermissionPush))

		// What happens if a group project is shared with a subgroup
		_, err = repo.TeamAccess().Create(ctx, gitprovider.TeamAccessInfo{
			Name:       fmt.Sprintf("%s/%s", testOrgName, testSubgroupName),
			Permission: &pushPermission,
		})
		Expect(err).ToNot(HaveOccurred())

		// See that the subgroup is listed
		projectTeams, err = repo.TeamAccess().List(ctx)
		Expect(err).ToNot(HaveOccurred())
		Expect(len(projectTeams)).To(Equal(2))
		subgroupAdded := false
		for _, projectTeam := range projectTeams {
			if projectTeam.Get().Name == fmt.Sprintf("%s/%s", testOrgName, testSubgroupName) {
				subgroupAdded = true
				break
			}
		}
		Expect(subgroupAdded).To(Equal(true))

		// Assert that reconciling on subgroups works
		teamInfo := gitprovider.TeamAccessInfo{
			Name:       testOrgName + "/" + testSubgroupName,
			Permission: &pushPermission,
		}

		ta, actionTaken, err = repo.TeamAccess().Reconcile(ctx, teamInfo)
		Expect(err).ToNot(HaveOccurred())
		Expect(actionTaken).To(Equal(false))
	})

	It("should create, delete and reconcile deploy keys", func() {
		testDeployKeyName := "test-deploy-key"
		repoRef := newOrgRepoRef(testOrgName, testSharedOrgRepoName)

		orgRepo, err := c.OrgRepositories().Get(ctx, repoRef)
		Expect(err).ToNot(HaveOccurred())

		// List keys should return 0
		keys, err := orgRepo.DeployKeys().List(ctx)
		Expect(err).ToNot(HaveOccurred())
		Expect(len(keys)).To(Equal(0))

		rsaGen := testutils.NewRSAGenerator(256)
		keyPair1, err := rsaGen.Generate()
		Expect(err).ToNot(HaveOccurred())
		pubKey := keyPair1.PublicKey

		readOnly := false
		testDeployKeyInfo := gitprovider.DeployKeyInfo{
			Name:     testDeployKeyName,
			Key:      pubKey,
			ReadOnly: &readOnly,
		}
		_, err = orgRepo.DeployKeys().Create(ctx, testDeployKeyInfo)
		Expect(err).ToNot(HaveOccurred())

		// List keys should now return 1
		keys, err = orgRepo.DeployKeys().List(ctx)
		Expect(err).ToNot(HaveOccurred())
		Expect(len(keys)).To(Equal(1))

		// Getting the key directly should return the same object
		getKey, err := orgRepo.DeployKeys().Get(ctx, testDeployKeyName)
		Expect(err).ToNot(HaveOccurred())

		deployKeyStr := string(testDeployKeyInfo.Key)
		Expect(string(getKey.Get().Key)).To(Equal(strings.TrimSuffix(deployKeyStr, "\n")))
		Expect(getKey.Get().Name).To(Equal(testDeployKeyInfo.Name))

		Expect(getKey.Set(getKey.Get())).ToNot(HaveOccurred())
		actionTaken, err := getKey.Reconcile(ctx)
		Expect(err).ToNot(HaveOccurred())
		Expect(actionTaken).To(BeFalse())

		// Reconcile creates a new key if the title and key is different
		title := "new-title"
		req := getKey.Get()
		req.Name = title

		keyPair2, err := rsaGen.Generate()
		Expect(err).ToNot(HaveOccurred())
		anotherPubKey := keyPair2.PublicKey
		req.Key = anotherPubKey

		Expect(getKey.Set(req)).ToNot(HaveOccurred())
		actionTaken, err = getKey.Reconcile(ctx)
		// Expect the update to succeed, and modify the state
		Expect(err).ToNot(HaveOccurred())
		Expect(actionTaken).To(BeTrue())

		getKey, err = orgRepo.DeployKeys().Get(ctx, title)
		Expect(err).ToNot(HaveOccurred())
		Expect(getKey.Get().Name).To(Equal(title))

		// Delete the keys
		keys, err = orgRepo.DeployKeys().List(ctx)
		Expect(err).ToNot(HaveOccurred())
		for _, key := range keys {
			err = key.Delete(ctx)
			Expect(err).ToNot(HaveOccurred())
		}
	})

	It("should be possible to create a user project", func() {
		// First, check what repositories are available
		repos, err := c.UserRepositories().List(ctx, newUserRef(testUserName))
		Expect(err).ToNot(HaveOccurred())

		// Generate a repository name which doesn't exist already
		testRepoName = fmt.Sprintf("test-repo-%03d", rand.Intn(1000))
		for findUserRepo(repos, testRepoName) != nil {
			testRepoName = fmt.Sprintf("test-repo-%03d", rand.Intn(1000))
		}

		// We know that a repo with this name doesn't exist in the organization, let's verify we get an
		// ErrNotFound
		repoRef := newUserRepoRef(testUserName, testRepoName)
		_, err = c.UserRepositories().Get(ctx, repoRef)
		Expect(errors.Is(err, gitprovider.ErrNotFound)).To(BeTrue())
		db := defaultBranchName

		// Create a new repo
		repo, err := c.UserRepositories().Create(ctx, repoRef, gitprovider.RepositoryInfo{
			DefaultBranch: &db,
			Description:   gitprovider.StringVar(defaultDescription),
			// Default visibility is private, no need to set this at least now
			//Visibility:     gitprovider.RepositoryVisibilityVar(gitprovider.RepositoryVisibilityPrivate),
		}, &gitprovider.RepositoryCreateOptions{
			AutoInit:        gitprovider.BoolVar(true),
			LicenseTemplate: gitprovider.LicenseTemplateVar(gitprovider.LicenseTemplateApache2),
		})
		Expect(err).ToNot(HaveOccurred())

		validateUserRepo(repo, repoRef)

		getRepo, err := c.UserRepositories().Get(ctx, repoRef)
		Expect(err).ToNot(HaveOccurred())
		// Expect the two responses (one from POST and one from GET to have equal "spec")
		getSpec := newGitlabProjectSpec(getRepo.APIObject().(*gitlab.Project))
		postSpec := newGitlabProjectSpec(repo.APIObject().(*gitlab.Project))
		Expect(getSpec.Equals(postSpec)).To(BeTrue())

		gitlabClient := c.Raw().(*gitlab.Client)
		f, _, err := gitlabClient.RepositoryFiles.GetFile(testUserName+"/"+testRepoName, "README.md", &gitlab.GetFileOptions{
			Ref: gitlab.String(defaultBranchName),
		})
		Expect(err).ToNot(HaveOccurred())
		fileContents, err := base64.StdEncoding.DecodeString(f.Content)
		Expect(err).ToNot(HaveOccurred())
		Expect(string(fileContents)).To(ContainSubstring(defaultDescription))
	})

	It("should error at creation time if the user repo already does exist", func() {
		repoRef := newUserRepoRef(testUserName, testRepoName)
		_, err := c.UserRepositories().Create(ctx, repoRef, gitprovider.RepositoryInfo{})
		Expect(errors.Is(err, gitprovider.ErrAlreadyExists)).To(BeTrue())
	})

	It("should update if the user repo already exists when reconciling", func() {
		repoRef := newUserRepoRef(testUserName, testRepoName)
		// No-op reconcile
		resp, actionTaken, err := c.UserRepositories().Reconcile(ctx, repoRef, gitprovider.RepositoryInfo{
			Description:   gitprovider.StringVar(defaultDescription),
			DefaultBranch: gitprovider.StringVar(defaultBranchName),
			Visibility:    gitprovider.RepositoryVisibilityVar(gitprovider.RepositoryVisibilityPrivate),
		})
		Expect(err).ToNot(HaveOccurred())
		Expect(actionTaken).To(BeFalse())
		// no-op set & reconcile
		Expect(resp.Set(resp.Get())).ToNot(HaveOccurred())
		actionTaken, err = resp.Reconcile(ctx)
		Expect(err).ToNot(HaveOccurred())
		Expect(actionTaken).To(BeFalse())

		// Update reconcile
		newDesc := "New description"
		req := resp.Get()
		req.Description = gitprovider.StringVar(newDesc)
		Expect(resp.Set(req)).ToNot(HaveOccurred())
		actionTaken, err = resp.Reconcile(ctx)
		// Expect the update to succeed, and modify the state
		Expect(err).ToNot(HaveOccurred())
		Expect(actionTaken).To(BeTrue())
		Expect(*resp.Get().Description).To(Equal(newDesc))

		// Delete the repository and later re-create
		Expect(resp.Delete(ctx)).ToNot(HaveOccurred())

		var newRepo gitprovider.UserRepository
		retryOp := testutils.NewRetry()
		Eventually(func() bool {
			var err error
			// Reconcile and create
			newRepo, actionTaken, err = c.UserRepositories().Reconcile(ctx, repoRef, gitprovider.RepositoryInfo{
				Description: gitprovider.StringVar(defaultDescription),
			}, &gitprovider.RepositoryCreateOptions{
				AutoInit:        gitprovider.BoolVar(true),
				LicenseTemplate: gitprovider.LicenseTemplateVar(gitprovider.LicenseTemplateMIT),
			})
			return retryOp.IsRetryable(err, fmt.Sprintf("new user repository: %s", repoRef.RepositoryName))
		}, retryOp.Timeout(), retryOp.Interval()).Should(BeTrue())

		// Expect the create to succeed, and have modified the state. Also validate the newRepo data
		Expect(actionTaken).To(BeTrue())
		validateUserRepo(newRepo, repoRef)
	})

	FIt("should be possible to create a pr for a user repository", func() {

		testRepoName = fmt.Sprintf("test-repo2-%03d", rand.Intn(1000))
		repoRef := newUserRepoRef(testUserName, testRepoName)

		defaultBranch := defaultBranchName
		description := "test description"
		// Create a new repo
		userRepo, err := c.UserRepositories().Create(ctx, repoRef,
			gitprovider.RepositoryInfo{
				DefaultBranch: &defaultBranch,
				Description:   &description,
				Visibility:    gitprovider.RepositoryVisibilityVar(gitprovider.RepositoryVisibilityPrivate),
			},
			&gitprovider.RepositoryCreateOptions{
				AutoInit: gitprovider.BoolVar(true),
			})
		Expect(err).ToNot(HaveOccurred())

		var commits []gitprovider.Commit = []gitprovider.Commit{}
		retryOp := testutils.NewRetry()
		Eventually(func() bool {
			var err error
			commits, err = userRepo.Commits().ListPage(ctx, defaultBranch, 1, 0)
			if err == nil && len(commits) == 0 {
				err = errors.New("empty commits list")
			}
			return retryOp.IsRetryable(err, fmt.Sprintf("get commits, repository: %s", userRepo.Repository().GetRepository()))
		}, retryOp.Timeout(), retryOp.Interval()).Should(BeTrue())

		latestCommit := commits[0]

		branchName := fmt.Sprintf("test-branch-%03d", rand.Intn(1000))
		branchName2 := fmt.Sprintf("test-branch-%03d", rand.Intn(1000))

		err = userRepo.Branches().Create(ctx, branchName2, "wrong-sha")
		Expect(err).To(HaveOccurred())

		err = userRepo.Branches().Create(ctx, branchName, latestCommit.Get().Sha)
		Expect(err).ToNot(HaveOccurred())

		path := "setup/config.txt"
		content := "yaml content 1"
		files := []gitprovider.CommitFile{
			{
				Path:    &path,
				Content: &content,
			},
		}

		_, err = userRepo.Commits().Create(ctx, branchName, "added config file", files)
		Expect(err).ToNot(HaveOccurred())

		pr, err := userRepo.PullRequests().Create(ctx, "Added config file", branchName, defaultBranch, "added config file")
		Expect(err).ToNot(HaveOccurred())

		prs, err := userRepo.PullRequests().List(ctx)
		Expect(err).ToNot(HaveOccurred())
		Expect(len(prs)).To(Equal(1))
		Expect(prs[0].Get().WebURL).To(Equal(pr.Get().WebURL))

		Expect(pr.Get().WebURL).ToNot(BeEmpty())
		Expect(pr.Get().Merged).To(BeFalse())
		err = userRepo.PullRequests().Merge(ctx, pr.Get().Number, gitprovider.MergeMethodSquash, "squash merged")
		Expect(err).ToNot(HaveOccurred())

		expectPRToBeMerged(ctx, userRepo, pr.Get().Number)

		// another pr

		commits, err = userRepo.Commits().ListPage(ctx, defaultBranch, 1, 0)
		Expect(err).ToNot(HaveOccurred())
		latestCommit = commits[0]
		err = userRepo.Branches().Create(ctx, branchName2, latestCommit.Get().Sha)
		Expect(err).ToNot(HaveOccurred())

		path2 := "setup/config2.txt"
		content2 := "yaml content 2"
		files = []gitprovider.CommitFile{
			{
				Path:    &path,
				Content: nil,
			},
			{
				Path:    &path2,
				Content: &content2,
			},
		}

		_, err = userRepo.Commits().Create(ctx, branchName2, "added second config file and removed first", files)
		Expect(err).ToNot(HaveOccurred())

		pr, err = userRepo.PullRequests().Create(ctx, "Added second config file", branchName2, defaultBranch, "added second config file")
		Expect(err).ToNot(HaveOccurred())
		Expect(pr.Get().WebURL).ToNot(BeEmpty())
		Expect(pr.Get().Merged).To(BeFalse())

		err = userRepo.PullRequests().Merge(ctx, pr.Get().Number, gitprovider.MergeMethodMerge, "merged")
		Expect(err).ToNot(HaveOccurred())

		expectPRToBeMerged(ctx, userRepo, pr.Get().Number)

		gitlabClient := c.Raw().(*gitlab.Client)
		_, res, err := gitlabClient.RepositoryFiles.GetFile(testUserName+"/"+testRepoName, path, &gitlab.GetFileOptions{
			Ref: gitlab.String(defaultBranchName),
		})
		Expect(err).To(HaveOccurred())
		Expect(res.StatusCode).To(Equal(404))
	})

	AfterSuite(func() {
		if os.Getenv("SKIP_CLEANUP") == "1" {
			return
		}

		// Don't do anything more if c wasn't created
		if c == nil {
			return
		}

		if len(os.Getenv("CLEANUP_ALL")) > 0 {
			defer cleanupOrgRepos("test-org-repo")
			defer cleanupOrgRepos("test-shared-org-repo")
			defer cleanupUserRepos("test-repo")
		}
		// Delete the test repo used
		fmt.Println("Deleting the user repo: ", testRepoName)
		repoRef := newUserRepoRef(testUserName, testRepoName)
		repo, err := c.UserRepositories().Get(ctx, repoRef)
		if errors.Is(err, gitprovider.ErrNotFound) {
			return
		}
		Expect(err).ToNot(HaveOccurred())
		Expect(repo.Delete(ctx)).ToNot(HaveOccurred())

		// Delete the test org repo used
		fmt.Println("Deleting the org repo: ", testOrgRepoName)
		orgRepoRef := newOrgRepoRef(testOrgName, testOrgRepoName)
		repo, err = c.OrgRepositories().Get(ctx, orgRepoRef)
		if errors.Is(err, gitprovider.ErrNotFound) {
			return
		}
		Expect(err).ToNot(HaveOccurred())
		Expect(repo.Delete(ctx)).ToNot(HaveOccurred())

		// Delete the test shared org repo used
		fmt.Println("Deleting the shared org repo: ", testSharedOrgRepoName)
		sharedOrgRepoRef := newOrgRepoRef(testOrgName, testSharedOrgRepoName)
		repo, err = c.OrgRepositories().Get(ctx, sharedOrgRepoRef)
		if errors.Is(err, gitprovider.ErrNotFound) {
			return
		}
		Expect(err).ToNot(HaveOccurred())
		Expect(repo.Delete(ctx)).ToNot(HaveOccurred())
	})
})

func expectPRToBeMerged(ctx context.Context, userRepo gitprovider.UserRepository, prNumber int) {
	Eventually(func() bool {
		getPR, err := userRepo.PullRequests().Get(ctx, prNumber)
		Expect(err).ToNot(HaveOccurred())
		return getPR.Get().Merged
	}, time.Second*5).Should(BeTrue(), `PR status didn't change to "merged"`)
}

func newOrgRef(organizationName string) gitprovider.OrganizationRef {
	return gitprovider.OrganizationRef{
		Domain:       gitlabDomain,
		Organization: organizationName,
	}
}

func newOrgRepoRef(organizationName, repoName string) gitprovider.OrgRepositoryRef {
	return gitprovider.OrgRepositoryRef{
		OrganizationRef: newOrgRef(organizationName),
		RepositoryName:  repoName,
	}
}

func newUserRef(userLogin string) gitprovider.UserRef {
	return gitprovider.UserRef{
		Domain:    gitlabDomain,
		UserLogin: userLogin,
	}
}

func newUserRepoRef(userLogin, repoName string) gitprovider.UserRepositoryRef {
	return gitprovider.UserRepositoryRef{
		UserRef:        newUserRef(userLogin),
		RepositoryName: repoName,
	}
}

func findOrgRepo(repos []gitprovider.OrgRepository, name string) gitprovider.OrgRepository {
	if name == "" {
		return nil
	}
	for _, repo := range repos {
		if repo.Repository().GetRepository() == name {
			return repo
		}
	}
	return nil
}

func findUserRepo(repos []gitprovider.UserRepository, name string) gitprovider.UserRepository {
	if name == "" {
		return nil
	}
	for _, repo := range repos {
		if repo.Repository().GetRepository() == name {
			return repo
		}
	}
	return nil
}
