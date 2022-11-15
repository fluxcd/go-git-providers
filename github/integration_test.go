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

package github

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/go-github/v47/github"
	"github.com/gregjones/httpcache"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/fluxcd/go-git-providers/gitprovider"
	"github.com/fluxcd/go-git-providers/gitprovider/testutils"
)

const (
	ghTokenFile  = "/tmp/github-token"
	githubDomain = "github.com"

	defaultDescription = "Foo description"
	// TODO: This will change
	defaultBranch = "main"
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
	RunSpecs(t, "GitHub Provider Suite")
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

var _ = Describe("GitHub Provider", func() {
	var (
		ctx context.Context = context.Background()
		c   gitprovider.Client

		testOrgRepoName  string
		testUserRepoName string
		testOrgName      string = "fluxcd-testing"
		testUser         string = "fluxcd-gitprovider-bot"
	)

	BeforeSuite(func() {
		githubToken := os.Getenv("GITHUB_TOKEN")
		if len(githubToken) == 0 {
			b, err := os.ReadFile(ghTokenFile)
			if token := string(b); err == nil && len(token) != 0 {
				githubToken = token
			} else {
				Fail("couldn't acquire GITHUB_TOKEN env variable")
			}
		}

		if orgName := os.Getenv("GIT_PROVIDER_ORGANIZATION"); len(orgName) != 0 {
			testOrgName = orgName
		}

		if gitProviderUser := os.Getenv("GIT_PROVIDER_USER"); len(gitProviderUser) != 0 {
			testUser = gitProviderUser
		}

		var err error
		c, err = NewClient(
			gitprovider.WithOAuth2Token(githubToken),
			gitprovider.WithDestructiveAPICalls(true),
			gitprovider.WithConditionalRequests(true),
			gitprovider.WithPreChainTransportHook(customTransportFactory),
		)
		Expect(err).ToNot(HaveOccurred())
	})

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
		fmt.Fprintf(os.Stderr, "Deleting repos starting with %s for user: %s\n", prefix, testUser)
		repos, err := c.UserRepositories().List(ctx, newUserRef(testUser))
		Expect(err).ToNot(HaveOccurred())
		fmt.Fprintf(os.Stderr, "repos, len: %d\n", len(repos))
		for _, repo := range repos {
			fmt.Fprintf(os.Stderr, "repo: %s\n", repo.Repository().GetRepository())
			name := repo.Repository().GetRepository()
			if !strings.HasPrefix(name, prefix) {
				fmt.Fprintf(os.Stderr, "Skipping the org repo: %s\n", name)
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
		// We don't expect the name from LIST calls, but we do expect
		// the description, see: https://docs.github.com/en/rest/reference/orgs#list-organizations
		Expect(listedOrg.Get().Name).To(BeNil())
		Expect(listedOrg.Get().Description).ToNot(BeNil())
		// We expect the name and description to be populated
		// in the GET call. Note: This requires the user to set up
		// the given organization with a name and description in the UI :)
		// https://docs.github.com/en/rest/reference/orgs#get-an-organization
		Expect(getOrg.Get().Name).ToNot(BeNil())
		Expect(getOrg.Get().Description).ToNot(BeNil())
		// Expect Name and Description to match their underlying data
		internal := getOrg.APIObject().(*github.Organization)
		Expect(getOrg.Get().Name).To(Equal(internal.Name))
		Expect(getOrg.Get().Description).To(Equal(internal.Description))

		// Expect that when we do the same request a second time, it will hit the cache
		hits = customTransportImpl.countCacheHitsForFunc(func() {
			getOrg2, err := c.Organizations().Get(ctx, listedOrg.Organization())
			Expect(err).ToNot(HaveOccurred())
			Expect(getOrg2).ToNot(BeNil())
		})
		Expect(hits).To(Equal(1))
	})

	It("should fail when .Children is called", func() {
		// Expect .Children to return gitprovider.ErrProviderNoSupport
		_, err := c.Organizations().Children(ctx, gitprovider.OrganizationRef{})
		Expect(errors.Is(err, gitprovider.ErrNoProviderSupport)).To(BeTrue())
	})

	It("should be possible to create an org repository", func() {
		// First, check what repositories are available
		repos, err := c.OrgRepositories().List(ctx, newOrgRef(testOrgName))
		Expect(err).ToNot(HaveOccurred())

		// Generate a repository name which doesn't exist already
		testOrgRepoName = fmt.Sprintf("test-repo-%03d", rand.Intn(1000))
		for findOrgRepo(repos, testOrgRepoName) != nil {
			testOrgRepoName = fmt.Sprintf("test-repo-%03d", rand.Intn(1000))
		}

		// We know that a repo with this name doesn't exist in the organization, let's verify we get an
		// ErrNotFound
		repoRef := newOrgRepoRef(testOrgName, testOrgRepoName)
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
		validateRepo(repo, repoRef)

		var getRepo gitprovider.OrgRepository
		Eventually(func() error {
			getRepo, err = c.OrgRepositories().Get(ctx, repoRef)
			return err
		}, 3*time.Second, 1*time.Second).ShouldNot(HaveOccurred())

		// Expect the two responses (one from POST and one from GET to have equal "spec")
		getSpec := newGithubRepositorySpec(getRepo.APIObject().(*github.Repository))
		postSpec := newGithubRepositorySpec(repo.APIObject().(*github.Repository))
		Expect(getSpec.Equals(postSpec)).To(BeTrue())
	})

	It("should be possible to create a user repository", func() {
		// First, check what repositories are available
		repos, err := c.UserRepositories().List(ctx, newUserRef(testUser))
		Expect(err).ToNot(HaveOccurred())

		// Generate a repository name which doesn't exist already
		testUserRepoName = fmt.Sprintf("test-repo-%03d", rand.Intn(1000))
		for findUserRepo(repos, testUserRepoName) != nil {
			testUserRepoName = fmt.Sprintf("test-repo-%03d", rand.Intn(1000))
		}

		desc := "GGP integration test user repo"
		userRepoRef := newUserRepoRef(testUser, testUserRepoName)
		userRepoInfo := gitprovider.RepositoryInfo{
			Description: &desc,
			Visibility:  gitprovider.RepositoryVisibilityVar(gitprovider.RepositoryVisibilityPrivate),
		}

		// Check that the repository doesn't exist
		_, err = c.UserRepositories().Get(ctx, userRepoRef)
		Expect(errors.Is(err, gitprovider.ErrNotFound)).To(BeTrue())

		// Create it
		userRepo, err := c.UserRepositories().Create(ctx, userRepoRef, userRepoInfo, &gitprovider.RepositoryCreateOptions{
			AutoInit:        gitprovider.BoolVar(true),
			LicenseTemplate: gitprovider.LicenseTemplateVar(gitprovider.LicenseTemplateApache2),
		})
		Expect(err).ToNot(HaveOccurred())

		// Should not be able to see the repo publicly
		anonClient, err := NewClient()
		Expect(err).ToNot(HaveOccurred())
		_, err = anonClient.UserRepositories().Get(ctx, userRepoRef)
		Expect(errors.Is(err, gitprovider.ErrNotFound)).To(BeTrue())

		var getUserRepo gitprovider.UserRepository
		Eventually(func() error {
			getUserRepo, err = c.UserRepositories().Get(ctx, userRepoRef)
			return err
		}, 3*time.Second, 1*time.Second).ShouldNot(HaveOccurred())

		// Expect the two responses (one from POST and one from GET to have equal "spec")
		getSpec := newGithubRepositorySpec(getUserRepo.APIObject().(*github.Repository))
		postSpec := newGithubRepositorySpec(userRepo.APIObject().(*github.Repository))
		Expect(getSpec.Equals(postSpec)).To(BeTrue())
	})

	It("should error at creation time if the repo already does exist", func() {
		repoRef := newOrgRepoRef(testOrgName, testOrgRepoName)
		_, err := c.OrgRepositories().Create(ctx, repoRef, gitprovider.RepositoryInfo{})
		Expect(errors.Is(err, gitprovider.ErrAlreadyExists)).To(BeTrue())
	})

	It("should update if the repository already exists when reconciling", func() {
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
		resp, actionTaken, err = c.OrgRepositories().Reconcile(ctx, repoRef, gitprovider.RepositoryInfo{
			Description:   gitprovider.StringVar(newDesc),
			DefaultBranch: gitprovider.StringVar(defaultBranch),
			Visibility:    gitprovider.RepositoryVisibilityVar(gitprovider.RepositoryVisibilityPrivate),
		})

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
			newRepo, actionTaken, err = c.OrgRepositories().Reconcile(ctx, repoRef, gitprovider.RepositoryInfo{
				Description: gitprovider.StringVar(defaultDescription),
			}, &gitprovider.RepositoryCreateOptions{
				AutoInit:        gitprovider.BoolVar(true),
				LicenseTemplate: gitprovider.LicenseTemplateVar(gitprovider.LicenseTemplateMIT),
			})
			if err == nil && !actionTaken {
				err = errors.New("expecting action taken to be true")
			}
			return retryOp.IsRetryable(err, fmt.Sprintf("reconcile user repository: %s", repoRef.RepositoryName))
		}, time.Second*90, retryOp.Interval()).Should(BeTrue())

		Expect(actionTaken).To(BeTrue())
		validateRepo(newRepo, repoRef)

		// Reconcile by setting an "internal" field and updating it
		r := newRepo.APIObject().(*github.Repository)
		r.DeleteBranchOnMerge = gitprovider.BoolVar(true)

		retryOp = testutils.NewRetry()
		retryOp.SetTimeout(time.Second * 90)
		Eventually(func() bool {
			var err error
			actionTaken, err = newRepo.Reconcile(ctx)
			if err == nil && !actionTaken {
				err = errors.New("expecting action taken to be true")
			}
			return retryOp.IsRetryable(err, fmt.Sprintf("reconcile repository: %s", newRepo.Repository().GetRepository()))
		}, retryOp.Timeout(), retryOp.Interval()).Should(BeTrue())

		Expect(actionTaken).To(BeTrue())
	})

	It("should validate that the token has the correct permissions", func() {
		hasPermission, err := c.HasTokenPermission(ctx, 0)
		Expect(err).To(Equal(gitprovider.ErrNoProviderSupport))
		Expect(hasPermission).To(Equal(false))

		hasPermission, err = c.HasTokenPermission(ctx, gitprovider.TokenPermissionRWRepository)
		Expect(err).ToNot(HaveOccurred())
		Expect(hasPermission).To(Equal(true))
	})

	It("should be possible to create and edit a pr for a user repository", func() {

		userRepoRef := newUserRepoRef(testUser, testUserRepoName)

		var userRepo gitprovider.UserRepository
		retryOp := testutils.NewRetry()
		Eventually(func() bool {
			var err error
			userRepo, err = c.UserRepositories().Get(ctx, userRepoRef)
			return retryOp.IsRetryable(err, fmt.Sprintf("get user repository: %s", userRepoRef.RepositoryName))
		}, retryOp.Timeout(), retryOp.Interval()).Should(BeTrue())

		defaultBranch := userRepo.Get().DefaultBranch

		var commits []gitprovider.Commit = []gitprovider.Commit{}
		retryOp = testutils.NewRetry()
		Eventually(func() bool {
			var err error
			commits, err = userRepo.Commits().ListPage(ctx, *defaultBranch, 1, 0)
			if err == nil && len(commits) == 0 {
				err = errors.New("empty commits list")
			}
			return retryOp.IsRetryable(err, fmt.Sprintf("get commits, repository: %s", userRepo.Repository().GetRepository()))
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
		Expect(pr.Get().Merged).To(BeFalse())
		Expect(pr.Get().SourceBranch).To(Equal(branchName))

		prs, err := userRepo.PullRequests().List(ctx)
		Expect(err).NotTo(HaveOccurred())
		Expect(len(prs)).To(Equal(1))
		Expect(prs[0].Get().WebURL).To(Equal(pr.Get().WebURL))

		err = userRepo.PullRequests().Merge(ctx, pr.Get().Number, gitprovider.MergeMethodSquash, "squash merged")
		Expect(err).ToNot(HaveOccurred())

		getPR, err := userRepo.PullRequests().Get(ctx, pr.Get().Number)
		Expect(err).ToNot(HaveOccurred())

		Expect(getPR.Get().Merged).To(BeTrue())

		path = "setup/config2.txt"
		content = "yaml content"
		files = []gitprovider.CommitFile{
			{
				Path:    &path,
				Content: &content,
			},
		}

		_, err = userRepo.Commits().Create(ctx, branchName, "added second config file", files)
		Expect(err).ToNot(HaveOccurred())

		pr, err = userRepo.PullRequests().Create(ctx, "Added second config file", branchName, *defaultBranch, "added second config file")
		Expect(err).ToNot(HaveOccurred())
		Expect(pr.Get().WebURL).ToNot(BeEmpty())
		Expect(pr.Get().Merged).To(BeFalse())

		editedPR, err := userRepo.PullRequests().Edit(ctx, pr.Get().Number, gitprovider.EditOptions{
			Title: gitprovider.StringVar("a new title"),
		})
		Expect(err).ToNot(HaveOccurred())

		err = userRepo.PullRequests().Merge(ctx, editedPR.Get().Number, gitprovider.MergeMethodMerge, "merged")
		Expect(err).ToNot(HaveOccurred())

		getPR, err = userRepo.PullRequests().Get(ctx, editedPR.Get().Number)
		Expect(err).ToNot(HaveOccurred())

		Expect(getPR.Get().Merged).To(BeTrue())
		apiObject := getPR.APIObject()
		githubPR, ok := apiObject.(*github.PullRequest)
		Expect(ok).To(BeTrue(), "API object of PullRequest has unexpected type %q", reflect.TypeOf(apiObject))
		Expect(githubPR.Title).ToNot(BeNil())
		Expect(*githubPR.Title).To(Equal("a new title"))
	})

	It("should be possible to download files from path and branch specified", func() {

		userRepoRef := newUserRepoRef(testUser, testUserRepoName)

		userRepo, err := c.UserRepositories().Get(ctx, userRepoRef)
		Expect(err).ToNot(HaveOccurred())

		defaultBranch := userRepo.Get().DefaultBranch

		path0 := "cluster/machine1.yaml"
		content0 := "machine1 yaml content"
		path1 := "cluster/machine2.yaml"
		content1 := "machine2 yaml content"

		files := []gitprovider.CommitFile{
			{
				Path:    &path0,
				Content: &content0,
			},
			{
				Path:    &path1,
				Content: &content1,
			},
		}

		commitFiles := make([]gitprovider.CommitFile, 0)
		for _, file := range files {
			path := file.Path
			content := file.Content
			commitFiles = append(commitFiles, gitprovider.CommitFile{
				Path:    path,
				Content: content,
			})
		}

		_, err = userRepo.Commits().Create(ctx, *defaultBranch, "added config files", commitFiles)
		Expect(err).ToNot(HaveOccurred())

		downloadedFiles, err := userRepo.Files().Get(ctx, "cluster", *defaultBranch)
		Expect(err).ToNot(HaveOccurred())

		for ind, downloadedFile := range downloadedFiles {
			Expect(*downloadedFile).To(Equal(files[ind]))
		}

	})

	It("should be possible to get and list repo tree", func() {

		userRepoRef := newUserRepoRef(testUser, testUserRepoName)

		userRepo, err := c.UserRepositories().Get(ctx, userRepoRef)
		Expect(err).ToNot(HaveOccurred())

		defaultBranch := userRepo.Get().DefaultBranch

		path0 := "clustersDir/cluster/machine.yaml"
		content0 := "machine yaml content"
		path1 := "clustersDir/cluster/machine1.yaml"
		content1 := "machine1 yaml content"
		path2 := "clustersDir/cluster2/clusterSubDir/machine2.yaml"
		content2 := "machine2 yaml content"

		files := []gitprovider.CommitFile{
			{
				Path:    &path0,
				Content: &content0,
			},
			{
				Path:    &path1,
				Content: &content1,
			},
			{
				Path:    &path2,
				Content: &content2,
			},
		}

		commitFiles := make([]gitprovider.CommitFile, 0)
		for _, file := range files {
			path := file.Path
			content := file.Content
			commitFiles = append(commitFiles, gitprovider.CommitFile{
				Path:    path,
				Content: content,
			})
		}

		commit, err := userRepo.Commits().Create(ctx, *defaultBranch, "added files", commitFiles)
		Expect(err).ToNot(HaveOccurred())
		commitSha := commit.Get().Sha

		// get tree
		tree, err := userRepo.Trees().Get(ctx, commitSha, true)
		Expect(err).ToNot(HaveOccurred())

		// Tree should have length 9 for : LICENSE, README.md, 3 blob (files), 4 tree (directories)
		Expect(tree.Tree).To(HaveLen(9))

		// itemsToBeIgnored initially with 2 for LICENSE and README.md, and will also include tree types
		itemsToBeIgnored := 2
		for ind, treeEntry := range tree.Tree {
			if treeEntry.Type == "blob" {
				if treeEntry.Path == "LICENSE" || treeEntry.Path == "README.md" {
					continue
				}
				Expect(treeEntry.Path).To(Equal(*files[ind-itemsToBeIgnored].Path))
				continue

			}
			itemsToBeIgnored += 1
		}

		// List tree items with no path provided
		treeEntries, err := userRepo.Trees().List(ctx, commitSha, "", true)
		Expect(err).ToNot(HaveOccurred())

		// Tree Entries should have length 5 for : LICENSE, README.md, 3 blob (files)
		Expect(treeEntries).To(HaveLen(5))
		for ind, treeEntry := range treeEntries {
			if treeEntry.Path == "LICENSE" || treeEntry.Path == "README.md" {
				continue
			}
			Expect(treeEntry.Path).To(Equal(*files[ind-2].Path))
		}

		//List tree items with path provided to filter on
		treeEntries, err = userRepo.Trees().List(ctx, commitSha, "clustersDir/", true)
		Expect(err).ToNot(HaveOccurred())

		// Tree Entries should have length 3 for :3 blob (files)
		Expect(treeEntries).To(HaveLen(3))
		for ind, treeEntry := range treeEntries {
			Expect(treeEntry.Path).To(Equal(*files[ind].Path))
		}

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
			defer cleanupOrgRepos("test-repo")
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
		userRepo, err := c.UserRepositories().Get(ctx, newUserRepoRef(testUser, testUserRepoName))
		Expect(err).ToNot(HaveOccurred())
		Expect(userRepo.Delete(ctx)).ToNot(HaveOccurred())
	})
})

func newOrgRef(organizationName string) gitprovider.OrganizationRef {
	return gitprovider.OrganizationRef{
		Domain:       githubDomain,
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
		Domain:    githubDomain,
		UserLogin: userLogin,
	}
}

func newUserRepoRef(userLogin, repoName string) gitprovider.UserRepositoryRef {
	return gitprovider.UserRepositoryRef{
		UserRef:        newUserRef(userLogin),
		RepositoryName: repoName,
	}
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

func validateRepo(repo gitprovider.OrgRepository, expectedRepoRef gitprovider.RepositoryRef) {
	info := repo.Get()
	// Expect certain fields to be set
	Expect(repo.Repository()).To(Equal(expectedRepoRef))
	Expect(*info.Description).To(Equal(defaultDescription))
	Expect(*info.Visibility).To(Equal(gitprovider.RepositoryVisibilityPrivate))
	Expect(*info.DefaultBranch).To(Equal(defaultBranch))
	// Expect high-level fields to match their underlying data
	internal := repo.APIObject().(*github.Repository)
	Expect(repo.Repository().GetRepository()).To(Equal(*internal.Name))
	Expect(repo.Repository().GetIdentity()).To(Equal(internal.Owner.GetLogin()))
	Expect(*info.Description).To(Equal(*internal.Description))
	Expect(string(*info.Visibility)).To(Equal(*internal.Visibility))
	Expect(*info.DefaultBranch).To(Equal(*internal.DefaultBranch))
}
