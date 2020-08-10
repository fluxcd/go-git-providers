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
		var listedOrg gitprovider.Organization
		for _, org := range orgs {
			if org.Organization().Organization == testOrgName {
				listedOrg = org
				break
			}
		}
		Expect(listedOrg).ToNot(BeNil())

		// Do a GET call for that organization
		getOrg, err := c.Organizations().Get(ctx, listedOrg.Organization())
		Expect(err).ToNot(HaveOccurred())
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
		internal := getOrg.APIObject().(*githubapi.Organization)
		Expect(getOrg.Get().Name).To(Equal(internal.Name))
		Expect(getOrg.Get().Description).To(Equal(internal.Description))
	})

	It("should fail when .Children is called", func() {
		// Expect .Children to return gitprovider.ErrProviderNoSupport
		_, err := c.Organizations().Children(ctx, gitprovider.OrganizationRef{})
		Expect(errors.Is(err, gitprovider.ErrNoProviderSupport)).To(BeTrue())
	})

	It("should be possible to create a repository", func() {
		// First, check what repositories are available
		repos, err := c.OrgRepositories().List(ctx, newOrgRef(testOrgName))
		Expect(err).ToNot(HaveOccurred())

		// Generate a repository name which doesn't exist already
		testRepoName = fmt.Sprintf("test-repo-%03d", rand.Intn(1000))
		for findRepo(repos, testRepoName) != nil {
			testRepoName = fmt.Sprintf("test-repo-%03d", rand.Intn(1000))
		}

		// We know that a repo with this name doesn't exist in the organization, let's verify we get an
		// ErrNotFound
		repoRef := newOrgRepoRef(testOrgName, testRepoName)
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

		getRepo, err := c.OrgRepositories().Get(ctx, repoRef)
		Expect(err).ToNot(HaveOccurred())
		// Expect the two responses (one from POST and one from GET to be equal)
		Expect(getRepo).To(Equal(repo))
	})

	It("should error at creation time if the repo already does exist", func() {
		repoRef := newOrgRepoRef(testOrgName, testRepoName)
		_, err := c.OrgRepositories().Create(ctx, repoRef, gitprovider.RepositoryInfo{})
		Expect(errors.Is(err, gitprovider.ErrAlreadyExists)).To(BeTrue())
	})

	It("should update if the repository already exists when reconciling", func() {
		repoRef := newOrgRepoRef(testOrgName, testRepoName)
		// No-op reconcile
		_, actionTaken, err := c.OrgRepositories().Reconcile(ctx, repoRef, gitprovider.RepositoryInfo{
			Description:   gitprovider.StringVar(defaultDescription),
			DefaultBranch: gitprovider.StringVar(defaultBranch),
			Visibility:    gitprovider.RepositoryVisibilityVar(gitprovider.RepositoryVisibilityPrivate),
		})
		Expect(err).ToNot(HaveOccurred())
		Expect(actionTaken).To(BeFalse())

		// Update reconcile
		newDesc := "New description"
		repo, actionTaken, err := c.OrgRepositories().Reconcile(ctx, repoRef, gitprovider.RepositoryInfo{
			Description: gitprovider.StringVar(newDesc),
		})
		// Expect the update to succeed, and modify the state
		Expect(err).ToNot(HaveOccurred())
		Expect(actionTaken).To(BeTrue())
		Expect(*repo.Get().Description).To(Equal(newDesc))

		// Delete the repository and later re-create
		Expect(repo.Delete(ctx)).ToNot(HaveOccurred())

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
		validateRepo(newRepo, repoRef)
	})

	AfterSuite(func() {
		// Don't do anything more if c wasn't created
		if c == nil {
			return
		}
		// Delete the test repo used
		repoRef := newOrgRepoRef(testOrgName, testRepoName)
		repo, err := c.OrgRepositories().Get(ctx, repoRef)
		Expect(err).ToNot(HaveOccurred())
		Expect(repo.Delete(ctx)).ToNot(HaveOccurred())
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

func findRepo(repos []gitprovider.OrgRepository, name string) gitprovider.OrgRepository {
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
	internal := repo.APIObject().(*githubapi.Repository)
	Expect(repo.Repository().GetRepository()).To(Equal(*internal.Name))
	Expect(repo.Repository().GetIdentity()).To(Equal(internal.Owner.GetLogin()))
	Expect(*info.Description).To(Equal(*internal.Description))
	Expect(string(*info.Visibility)).To(Equal(*internal.Visibility))
	Expect(*info.DefaultBranch).To(Equal(*internal.DefaultBranch))
}
