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
	"io/ioutil"
	"math/rand"
	"os"
	"testing"
	"time"

	gitprovider "github.com/fluxcd/go-git-providers"
	githubapi "github.com/google/go-github/v32/github"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const (
	ghTokenFile  = "/tmp/github-token"
	githubDomain = "github.com"

	defaultDescription = "Foo description"
	// TODO: This will change
	defaultBranch = "master"
)

func init() {
	// Call testing.Init() prior to tests.NewParams(), as otherwise -test.* will not be recognised. See also: https://golang.org/doc/go1.13#testing
	testing.Init()
	rand.Seed(time.Now().UnixNano())
}

func TestProvider(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Github Provider Suite")
}

var _ = Describe("Github Provider", func() {
	var (
		ctx context.Context
		c   gitprovider.Client

		testRepoName string
		testOrgName  string = "fluxcd-testing"
	)

	BeforeSuite(func() {
		githubToken := os.Getenv("GITHUB_TOKEN")
		if len(githubToken) == 0 {
			githubToken = os.Getenv("GITPROVIDER_BOT_TOKEN")
		}
		if len(githubToken) == 0 {
			b, err := ioutil.ReadFile(ghTokenFile)
			if token := string(b); err == nil && len(token) != 0 {
				githubToken = token
			} else {
				Fail("couldn't aquire GITHUB_TOKEN env variable")
			}
		}

		if orgName := os.Getenv("GIT_PROVIDER_ORGANIZATION"); len(orgName) != 0 {
			testOrgName = orgName
		}

		ctx = context.Background()
		var err error
		c, err = NewClient(ctx, WithPersonalAccessToken(githubToken))
		Expect(err).ToNot(HaveOccurred())
	})

	It("should list the available organizations the user has access to", func() {
		// Get a list of all organizations the user is part of
		orgs, err := c.Organizations().List(ctx)
		Expect(err).ToNot(HaveOccurred())

		// Make sure we find the expected one given as testOrgName
		var listedOrg *gitprovider.Organization
		for _, org := range orgs {
			if org.GetOrganization() == testOrgName {
				listedOrg = &org
				break
			}
		}
		Expect(listedOrg).ToNot(BeNil())

		// Do a GET call for that organization
		_, getOrg, err := c.Organization(ctx, listedOrg)
		Expect(err).ToNot(HaveOccurred())
		// Expect that the organization's info is the same regardless of method
		Expect(gitprovider.Equals(getOrg.OrganizationInfo, listedOrg.OrganizationInfo)).To(BeTrue())
		// We don't expect the name from LIST calls, but we do expect
		// the description, see: https://docs.github.com/en/rest/reference/orgs#list-organizations
		Expect(listedOrg.Name).To(BeNil())
		Expect(listedOrg.Description).ToNot(BeNil())
		// We expect the name and description to be populated
		// in the GET call. Note: This requires the user to set up
		// the given organization with a name and description in the UI :)
		// https://docs.github.com/en/rest/reference/orgs#get-an-organization
		Expect(getOrg.Name).ToNot(BeNil())
		Expect(getOrg.Description).ToNot(BeNil())
		// Expect Name and Description to match their underlying data
		internal := getOrg.Internal.(*githubapi.Organization)
		Expect(getOrg.Name).To(Equal(internal.Name))
		Expect(getOrg.Description).To(Equal(internal.Description))
	})

	It("should fail when .Children is called", func() {
		// Expect .Children to return gitprovider.ErrProviderNoSupport
		_, err := c.Organizations().Children(ctx, nil)
		Expect(errors.Is(err, gitprovider.ErrProviderNoSupport)).To(BeTrue())
	})

	It("should be possible to create a repository", func() {
		// First, check what repositories are available
		repos, err := c.Repositories().List(ctx, newOrgInfo(testOrgName))
		Expect(err).ToNot(HaveOccurred())

		// Generate a repository name which doesn't exist already
		testRepoName = fmt.Sprintf("test-repo-%03d", rand.Intn(1000))
		for findRepo(repos, testRepoName) != nil {
			testRepoName = fmt.Sprintf("test-repo-%03d", rand.Intn(1000))
		}

		// We know that a repo with this name doesn't exist in the organization, let's verify we get an
		// ErrNotFound
		repoInfo := newRepoInfo(testOrgName, testRepoName)
		_, err = c.Repositories().Get(ctx, repoInfo)
		Expect(errors.Is(err, gitprovider.ErrNotFound)).To(BeTrue())

		// Create a new repo
		repo, err := c.Repositories().Create(ctx, &gitprovider.Repository{
			RepositoryInfo: repoInfo,
			Description:    gitprovider.StringVar(defaultDescription),
			// Default visibility is private, no need to set this at least now
			//Visibility:     gitprovider.RepositoryVisibilityVar(gitprovider.RepositoryVisibilityPrivate),
		}, &gitprovider.RepositoryCreateOptions{
			AutoInit:        gitprovider.BoolVar(true),
			LicenseTemplate: gitprovider.LicenseTemplateVar(gitprovider.LicenseTemplateApache2),
		})
		Expect(err).ToNot(HaveOccurred())
		validateRepo(repo, repoInfo)

		getRepo, err := c.Repositories().Get(ctx, repoInfo)
		Expect(err).ToNot(HaveOccurred())
		// Expect the two responses (one from POST and one from GET to be equal)
		Expect(gitprovider.Equals(getRepo, repo))
	})

	It("should error at creation time if the repo already does exist", func() {
		_, err := c.Repositories().Create(ctx, &gitprovider.Repository{
			RepositoryInfo: newRepoInfo(testOrgName, testRepoName),
		})
		Expect(errors.Is(err, gitprovider.ErrAlreadyExists)).To(BeTrue())
	})

	It("should update if the repository already exists when reconciling", func() {
		// No-op reconcile
		_, actionTaken, err := c.Repositories().Reconcile(ctx, &gitprovider.Repository{
			RepositoryInfo: newRepoInfo(testOrgName, testRepoName),
			Description:    gitprovider.StringVar(defaultDescription),
			DefaultBranch:  gitprovider.StringVar(defaultBranch),
			Visibility:     gitprovider.RepositoryVisibilityVar(gitprovider.RepositoryVisibilityPrivate),
		})
		Expect(err).ToNot(HaveOccurred())
		Expect(actionTaken).To(BeFalse())

		// Update reconcile
		newDesc := "New description"
		repo, actionTaken, err := c.Repositories().Reconcile(ctx, &gitprovider.Repository{
			RepositoryInfo: newRepoInfo(testOrgName, testRepoName),
			Description:    gitprovider.StringVar(newDesc),
		})
		// Expect the update to succeed, and modify the state
		Expect(err).ToNot(HaveOccurred())
		Expect(actionTaken).To(BeTrue())
		Expect(*repo.Description).To(Equal(newDesc))

		// Delete the repository and later re-create
		Expect(deleteRepo(c, repo.RepositoryInfo)).ToNot(HaveOccurred())

		// Reconcile and create
		newRepo, actionTaken, err := c.Repositories().Reconcile(ctx, &gitprovider.Repository{
			RepositoryInfo: repo.RepositoryInfo,
			Description:    gitprovider.StringVar(defaultDescription),
		}, &gitprovider.RepositoryCreateOptions{
			AutoInit:        gitprovider.BoolVar(true),
			LicenseTemplate: gitprovider.LicenseTemplateVar(gitprovider.LicenseTemplateMIT),
		})
		// Expect the create to succeed, and have modified the state. Also validate the newRepo data
		Expect(err).ToNot(HaveOccurred())
		Expect(actionTaken).To(BeTrue())
		validateRepo(newRepo, repo.RepositoryInfo)
	})

	AfterSuite(func() {
		// Don't do anything more if c wasn't created
		if c == nil {
			return
		}
		Expect(deleteRepo(c, newRepoInfo(testOrgName, testRepoName))).ToNot(HaveOccurred())
	})
})

func newOrgInfo(organizationName string) gitprovider.OrganizationInfo {
	return gitprovider.OrganizationInfo{
		Domain:       githubDomain,
		Organization: organizationName,
	}
}

func newRepoInfo(organizationName, repoName string) gitprovider.RepositoryInfo {
	return gitprovider.RepositoryInfo{
		IdentityRef:    newOrgInfo(organizationName),
		RepositoryName: repoName,
	}
}

func findRepo(repos []gitprovider.Repository, name string) *gitprovider.Repository {
	if name == "" {
		return nil
	}
	for _, repo := range repos {
		if repo.RepositoryName == name {
			return &repo
		}
	}
	return nil
}

func validateRepo(repo *gitprovider.Repository, repoInfo gitprovider.RepositoryInfo) {
	// Expect certain fields to be set
	Expect(repo.RepositoryInfo).To(Equal(repoInfo))
	Expect(*repo.Description).To(Equal(defaultDescription))
	Expect(*repo.Visibility).To(Equal(gitprovider.RepositoryVisibilityPrivate))
	Expect(*repo.DefaultBranch).To(Equal(defaultBranch))
	// Expect high-level fields to match their underlying data
	internal := repo.Internal.(*githubapi.Repository)
	Expect(repo.GetRepository()).To(Equal(*internal.Name))
	Expect(repo.GetIdentity()).To(Equal(internal.Owner.GetLogin()))
	Expect(*repo.Description).To(Equal(*internal.Description))
	Expect(string(*repo.Visibility)).To(Equal(*internal.Visibility))
	Expect(*repo.DefaultBranch).To(Equal(*internal.DefaultBranch))
}

func deleteRepo(c gitprovider.Client, ref gitprovider.RepositoryRef) error {
	return (c.Repositories().(*RepositoriesClient)).Delete(context.Background(), ref)
}
