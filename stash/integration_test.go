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

package stash

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"encoding/json"
	"github.com/gregjones/httpcache"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"go.uber.org/zap"

	"github.com/fluxcd/go-git-providers/gitprovider"
	"github.com/fluxcd/go-git-providers/gitprovider/testutils"
)

const (
	ghTokenFile        = "/tmp/gostash.token"
	defaultDescription = "Foo description"
)

var (
	// customTransportImpl is a shared instance of a customTransport, allowing counting of cache hits.
	customTransportImpl *customTransport
	stashDomain         = "stash.example.com"
	defaultBranch       = "master"
)

func init() {
	// Call testing.Init() prior to tests.NewParams(), as otherwise -test.* will not be recognised. See also: https://golang.org/doc/go1.13#testing
	testing.Init()
	rand.Seed(time.Now().UnixNano())
}

func setupLogr() logr.Logger {
	zapLog, err := setupLog()
	if err != nil {
		panic(fmt.Sprintf("who watches the watchmen (%v)?", err))
	}
	return zapr.NewLogger(zapLog)
}

func setupLog() (*zap.Logger, error) {
	rawJSON := []byte(`{
	  "level": "info",
	  "encoding": "json",
	  "outputPaths": ["stdout"],
	  "errorOutputPaths": ["stderr"],
	  "initialFields": {"name": "stash integration test"},
	  "encoderConfig": {
	    "messageKey": "message",
	    "levelKey": "level",
	    "levelEncoder": "lowercase"
	  }
	}`)

	var cfg zap.Config
	if err := json.Unmarshal(rawJSON, &cfg); err != nil {
		return nil, err
	}
	logger, err := cfg.Build()
	if err != nil {
		return nil, err
	}
	defer logger.Sync()
	return logger, nil
}

func TestProvider(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Stash Provider Suite")
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

func (t *customTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	t.mux.Lock()
	defer t.mux.Unlock()

	resp, err := t.transport.RoundTrip(req)
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

var _ = Describe("Stash Provider", func() {
	var (
		ctx context.Context = context.Background()
		c   gitprovider.Client

		// Should exist in environment
		testOrgName string = "fluxcd-testing"
		//testSubgroupName string = "fluxcd-testing-sub-group" No subgroups in stash
		testTeamName string = "fluxcd-testing-2"
		testUserName string = "fluxcd-gitprovider-bot"

		// placeholders, will be randomized and created.
		testSharedOrgRepoName string = "testsharedorgrepo"
		testOrgRepoName       string = "testorgrepo"
		testRepoName          string = "testrepo"
	)

	BeforeSuite(func() {

		log := setupLogr()
		log.V(1).Info("logger construction succeeded")

		stashToken := os.Getenv("STASH_TOKEN")
		if len(stashToken) == 0 {
			b, err := ioutil.ReadFile(ghTokenFile)
			if token := string(b); err == nil && len(token) != 0 {
				stashToken = token
			} else {
				Fail("couldn't acquire STASH_TOKEN env variable")
			}
		}

		if stashDomainVar := os.Getenv("STASH_DOMAIN"); len(stashDomainVar) != 0 {
			stashDomain = stashDomainVar
		}

		if orgName := os.Getenv("GIT_PROVIDER_ORGANIZATION"); len(orgName) != 0 {
			testOrgName = orgName
		}

		if gitProviderUser := os.Getenv("GIT_PROVIDER_USER"); len(gitProviderUser) != 0 {
			testUserName = gitProviderUser
		}

		if teamName := os.Getenv("STASH_TEST_TEAM_NAME"); len(teamName) != 0 {
			testTeamName = teamName
		}

		var err error
		c, err = NewClient(
			stashToken, "",
			WithDomain(stashDomain),
			WithDestructiveAPICalls(true),
			WithConditionalRequests(true),
			WithPreChainTransportHook(customTransportFactory),
			WithLogger(&log),
		)
		Expect(err).ToNot(HaveOccurred())
	})

	validateOrgRepo := func(repo gitprovider.OrgRepository, expectedRepoRef gitprovider.RepositoryRef) {
		info := repo.Get()
		// Expect certain fields to be set
		Expect(repo.Repository()).To(Equal(expectedRepoRef))
		Expect(*info.Description).To(Equal(defaultDescription))
		Expect(*info.Visibility).To(Equal(gitprovider.RepositoryVisibilityPrivate))
		Expect(*info.DefaultBranch).To(Equal(masterBranchName))
		// Expect high-level fields to match their underlying data
		internal := repo.APIObject().(*Repository)
		Expect(repo.Repository().GetRepository()).To(Equal(internal.Name))
		Expect(repo.Repository().GetIdentity()).To(Equal(testOrgName))
		Expect(*info.Visibility).To(Equal(gitprovider.RepositoryVisibilityPrivate))
		Expect(*info.DefaultBranch).To(Equal(masterBranchName))
	}

	validateUserRepo := func(repo gitprovider.UserRepository, expectedRepoRef gitprovider.RepositoryRef) {
		info := repo.Get()
		// Expect certain fields to be set
		Expect(repo.Repository()).To(Equal(expectedRepoRef))
		Expect(*info.Description).To(Equal(defaultDescription))
		Expect(*info.Visibility).To(Equal(gitprovider.RepositoryVisibilityPrivate))
		Expect(*info.DefaultBranch).To(Equal(masterBranchName))
		// Expect high-level fields to match their underlying data
		internal := repo.APIObject().(*Repository)
		Expect(repo.Repository().GetRepository()).To(Equal(internal.Name))
		Expect(repo.Repository().GetIdentity()).To(Equal(testUserName))
		if !internal.Public {
			Expect(*info.Visibility).To(Equal(gitprovider.RepositoryVisibilityPrivate))
		}

		//Expect(*info.DefaultBranch).To(Equal(internal.Branch))
	}

	cleanupOrgRepos := func(prefix string) {
		fmt.Printf("Deleting repos starting with %s in org: %s\n", prefix, testOrgName)
		repos, err := c.OrgRepositories().List(ctx, newOrgRef(testOrgName))
		Expect(err).ToNot(HaveOccurred())
		for _, repo := range repos {
			// Delete the test org repo used
			name := repo.Repository().GetRepository()
			if !strings.HasPrefix(name, prefix) {
				continue
			}
			fmt.Printf("Deleting the org repo: %s\n", name)
			repo.Delete(ctx)
			Expect(err).ToNot(HaveOccurred())
		}
	}

	cleanupUserRepos := func(prefix string) {
		fmt.Printf("Deleting repos starting with %s for user: %s\n", prefix, testUserName)
		repos, err := c.UserRepositories().List(ctx, newUserRef(testUserName))
		Expect(err).ToNot(HaveOccurred())
		for _, repo := range repos {
			// Delete the test org repo used
			name := repo.Repository().GetRepository()
			if !strings.HasPrefix(name, prefix) {
				continue
			}
			fmt.Printf("Deleting the org repo: %s\n", name)
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
		// No Descrition available - Expect(listedOrg.Get().Description).ToNot(BeNil())

		// We expect the name and description to be populated
		// in the GET call. Note: This requires the user to set up
		// the given organization with a name and description in the UI :)
		Expect(getOrg.Get().Name).ToNot(BeNil())
		// No Descrition available - Expect(getOrg.Get().Description).ToNot(BeNil())
		// Expect Name and Description to match their underlying data
		internal := getOrg.APIObject().(*Project)
		derefOrgName := *getOrg.Get().Name
		Expect(derefOrgName).To(Equal(internal.Name))
		/* no cache
		// Expect that when we do the same request a second time, it will hit the cache
		hits = customTransportImpl.countCacheHitsForFunc(func() {
			getOrg2, err := c.Organizations().Get(ctx, listedOrg.Organization())
			Expect(err).ToNot(HaveOccurred())
			Expect(getOrg2).ToNot(BeNil())
		})
		Expect(hits).To(Equal(1))
		*/
	})

	It("should not fail when .Children is called", func() {
		_, err := c.Organizations().Children(ctx, gitprovider.OrganizationRef{
			Domain:       stashDomain,
			Organization: testOrgName,
		})
		Expect(err).To(Equal(gitprovider.ErrNoProviderSupport))
	})

	It("should be possible to create a project repo", func() {
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
		getSpec := newStashRepositorySpec(getRepo.APIObject().(*Repository))
		postSpec := newStashRepositorySpec(repo.APIObject().(*Repository))
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

		// Reconcile and create
		newRepo, actionTaken, err := c.OrgRepositories().Reconcile(ctx, repoRef, gitprovider.RepositoryInfo{
			Description: gitprovider.StringVar(defaultDescription),
		}, &gitprovider.RepositoryCreateOptions{
			AutoInit:        gitprovider.BoolVar(true),
			LicenseTemplate: gitprovider.LicenseTemplateVar(gitprovider.LicenseTemplateMIT),
		})
		// Expect the create to succeed, and have modified the state. Also validate the newRepo data
		Expect(err).ToNot(HaveOccurred())
		Expect(actionTaken).To(BeTrue())
		validateOrgRepo(newRepo, repoRef)
	})

	It("should update teams with access and permissions when reconciling", func() {

		// Get the test organization
		orgRef := newOrgRef(testOrgName)
		testOrg, err := c.Organizations().Get(ctx, orgRef)
		Expect(err).ToNot(HaveOccurred())

		// List all the teams with access to the org
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
		permission := gitprovider.RepositoryPermissionAdmin // Changed to admin because Push and Maintain both map to REPO_WRITE and we use push below in a reconcile test
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

		/* No subgroups in stash
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
		/* No subgroups in stash
		// Assert that reconciling on subgroups works
		teamInfo := gitprovider.TeamAccessInfo{
			Name:       testOrgName + "/" + testSubgroupName,
			Permission: &pushPermission,
		}

		ta, actionTaken, err = repo.TeamAccess().Reconcile(ctx, teamInfo)
		Expect(err).ToNot(HaveOccurred())
		Expect(actionTaken).To(Equal(false))
		*/
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

		readOnly := false
		testDeployKeyInfo := gitprovider.DeployKeyInfo{
			Name:     testDeployKeyName,
			Key:      []byte("ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDLbHBknQQaasdl2/O9DfgizMyUh/lhYwXk9GrBY9Ow9fFHy+lIiRBiS8H4rjvP2YkECrWWSbcevTKe+yk7PsU98RZiPL9S2+d2ENo3uQ2Rp6xnKY+XtvJnSvpLnABz/mGDPgvcLxXvPj2rAGu35u08DZ1WufU7hzgiWuLM3TH/albVcadw5ZflOAXalMmUhinB9m/O71DWyaP33pIqZBGCc8IBMcUHOL72NkNcpatXvCALltApJVUPZIvQUnrmUOglQMklaCeeWn6B269UI9kH9TjhGbbIvHpPZ7Ky9RTklZTeINLZW5Yql/leA/vJGcIipyXQkDPs7RSwtpmp5kat test-deploy-key"),
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

		Expect(getKey.Get().Key).To(Equal(testDeployKeyInfo.Key))
		Expect(getKey.Get().Name).To(Equal(testDeployKeyInfo.Name))

		Expect(getKey.Set(getKey.Get())).ToNot(HaveOccurred())
		actionTaken, err := getKey.Reconcile(ctx)
		Expect(err).ToNot(HaveOccurred())
		Expect(actionTaken).To(BeFalse())

		// Reconcile creates a new key if the title and key is different
		title := "new-title"
		req := getKey.Get()
		req.Name = title
		req.Key = []byte("ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQCasdHV91pRmqTaWPJnvZmvTPZPpHmIYocY1kmYFeOOL6/ofdvYb1sxNwsccOJEeLGJjp6FGe4BWNQSqDUCeO3EVU8A7ZTnd9eizB8nYDGoACbmG2GfMmtAdxKfsPE/lNRzAOFmHAHrzOnL6zk5SMPe0Y2poW1Z5w+If5r62WwfqG2/ujUA7BU3Vf/arFIYJvXvuEOJMP3QbezWL0b22Wmedu8esKrOYcak80I6Ti8qiof8ly1JZa58ezHJVvcEWZGSKU4G53jmDz7ky4GGb9DRo+LqOaU1qetdJX1GiCRNnhvz8DsxGcL77BJPE7HPBct44lN1TZCeIOG00Hai4bDp dinos@dinos-desktop")
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

	It("should be possible to create a user repo", func() {
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

		// Create a new repo
		repo, err := c.UserRepositories().Create(ctx, repoRef, gitprovider.RepositoryInfo{
			Description: gitprovider.StringVar(defaultDescription),
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
		getSpec := newStashRepositorySpec(getRepo.APIObject().(*Repository))
		postSpec := newStashRepositorySpec(repo.APIObject().(*Repository))
		Expect(getSpec.Equals(postSpec)).To(BeTrue())
		/*
			stashClient := c.Raw().(*Client)
			f, _, err := stashClient.RepositoryFiles.GetFile(testUserName+"/"+testRepoName, "README.md")
			Expect(err).ToNot(HaveOccurred())
			fileContents, err := base64.StdEncoding.DecodeString(f.Content)
			Expect(err).ToNot(HaveOccurred())
			Expect(string(fileContents)).To(ContainSubstring(defaultDescription))
		*/
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

		time.Sleep(10 * time.Second)
		// Reconcile and create
		newRepo, actionTaken, err := c.UserRepositories().Reconcile(ctx, repoRef, gitprovider.RepositoryInfo{
			Description: gitprovider.StringVar(defaultDescription),
		}, &gitprovider.RepositoryCreateOptions{
			AutoInit:        gitprovider.BoolVar(true),
			LicenseTemplate: gitprovider.LicenseTemplateVar(gitprovider.LicenseTemplateMIT),
		})
		// Expect the create to succeed, and have modified the state. Also validate the newRepo data
		Expect(err).ToNot(HaveOccurred())
		Expect(actionTaken).To(BeTrue())
		validateUserRepo(newRepo, repoRef)
	})

	It("should be possible to create a pr for a user repository", func() {

		userRepoRef := newUserRepoRef(testUserName, testRepoName)

		var userRepo gitprovider.UserRepository
		retryOp := testutils.NewRetry()
		retryOp.SetTimeout(time.Second * 10)
		Eventually(func() bool {
			var err error
			userRepo, err = c.UserRepositories().Get(ctx, userRepoRef)
			return retryOp.Retry(err, fmt.Sprintf("get user repository: %s", userRepoRef.RepositoryName))
		}, retryOp.Timeout(), retryOp.Interval()).Should(BeTrue())

		defaultBranch := userRepo.Get().DefaultBranch

		var commits []gitprovider.Commit = []gitprovider.Commit{}
		retryOp = testutils.NewRetry()
		retryOp.SetTimeout(time.Second * 10)
		Eventually(func() bool {
			var err error
			commits, err = userRepo.Commits().ListPage(ctx, *defaultBranch, 1, 0)
			if err == nil && len(commits) == 0 {
				err = errors.New("empty commits list")
			}
			return retryOp.Retry(err, fmt.Sprintf("get commits, repository: %s", userRepo.Repository().GetRepository()))
		}, retryOp.Timeout(), retryOp.Interval()).Should(BeTrue())

		latestCommit := commits[0]

		branchName := fmt.Sprintf("test-branch-%03d", rand.Intn(1000))

		err := userRepo.Branches().Create(ctx, branchName, latestCommit.Get().Sha)
		Expect(err).ToNot(HaveOccurred())

		err = userRepo.Branches().Create(ctx, branchName, "wrong-sha")
		Expect(err).To(HaveOccurred())

		path := "setup/config.txt"
		content := "yaml content"
		files := []gitprovider.CommitFile{
			{
				Path:    &path,
				Content: &content,
			},
		}

		_, err = userRepo.Commits().Create(ctx, branchName, "added config file", files)
		Expect(err).ToNot(HaveOccurred())

		pr, err := userRepo.PullRequests().Create(ctx, "Added config file", branchName, *defaultBranch, "added config file")
		Expect(err).ToNot(HaveOccurred())
		Expect(pr.Get().WebURL).ToNot(BeEmpty())
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

		// Delete the org test repo used
		orgRepo, err := c.OrgRepositories().Get(ctx, newOrgRepoRef(testOrgName, testOrgRepoName))
		if err != nil && len(os.Getenv("CLEANUP_ALL")) > 0 {
			fmt.Fprintf(os.Stderr, "failed to get repo: %s in org: %s, error: %s\n", testOrgRepoName, testOrgName, err)
			fmt.Fprintf(os.Stderr, "CLEANUP_ALL set so continuing\n")
		} else {
			Expect(err).ToNot(HaveOccurred())
			Expect(orgRepo.Delete(ctx)).ToNot(HaveOccurred())
		}
		// Delete the user test repo used
		userRepo, err := c.UserRepositories().Get(ctx, newUserRepoRef(testUserName, testRepoName))
		Expect(err).ToNot(HaveOccurred())
		Expect(userRepo.Delete(ctx)).ToNot(HaveOccurred())
	})
})

func newOrgRef(organizationName string) gitprovider.OrganizationRef {
	return gitprovider.OrganizationRef{
		Domain:       stashDomain,
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
		Domain:    stashDomain,
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
