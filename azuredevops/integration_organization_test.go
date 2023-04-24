/*
Copyright 2023 The Flux authors

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

package azuredevops

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"reflect"

	"github.com/fluxcd/go-git-providers/gitprovider"
	"github.com/fluxcd/go-git-providers/gitprovider/testutils"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/git"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Azure Provider", func() {
	var (
		ctx context.Context = context.Background()
	)

	validateOrgRepo := func(repo gitprovider.OrgRepository, expectedRepo gitprovider.RepositoryRef) {
		info := repo.Get()
		// Expect certain fields to be set
		Expect(repo.Repository()).To(Equal(expectedRepo))
		Expect(*info.Visibility).To(Equal(gitprovider.RepositoryVisibilityPrivate))

		// Expect high-level fields to match their underlying data
		internal := repo.APIObject().(*git.GitRepository)
		Expect(repo.Repository().GetRepository()).To(Equal(*internal.Name))
		Expect(repo.Repository().GetIdentity()).To(Equal(testOrgName))
		Expect(*info.Visibility).To(Equal(gitprovider.RepositoryVisibilityPrivate))
	}

	It("should be possible to create an organization repo", func() {
		// Get the test organization
		orgRef := newOrgRef(testOrgName)
		testOrg, err := client.Organizations().Get(ctx, orgRef)
		Expect(err).ToNot(HaveOccurred())

		// First, check what repositories are available
		repos, err := client.OrgRepositories().List(ctx, testOrg.Organization())
		Expect(err).ToNot(HaveOccurred())

		// Generate a repository name which doesn't exist already
		testOrgRepoName = fmt.Sprintf("test-org-repo-%03d", rand.Intn(1000))
		for findRepo(repos, testOrgRepoName) != nil {
			testOrgRepoName = fmt.Sprintf("test-org-repo-%03d", rand.Intn(1000))
		}

		fmt.Print("Creating repository - create org test ", testOrgRepoName, "...")
		// We know that a repo with this name doesn't exist in the organization, let's verify we get an
		// ErrNotFound
		// Verify that we can get clone url for the repo
		repoRef := newRepoRef(testOrg.Organization(), testOrgRepoName)
		sshURL := repoRef.GetCloneURL(gitprovider.TransportTypeSSH)
		Expect(sshURL).NotTo(Equal(""))
		_, err = client.OrgRepositories().Get(ctx, repoRef)
		Expect(err).To(MatchError(gitprovider.ErrNotFound))

		// Create a new repo
		repo, err := client.OrgRepositories().Create(ctx, repoRef, gitprovider.RepositoryInfo{

			// Default visibility is private, no need to set this at least now
			Visibility: gitprovider.RepositoryVisibilityVar(gitprovider.RepositoryVisibilityPrivate),
			Name:       gitprovider.StringVar(testOrgName),
		}, &gitprovider.RepositoryCreateOptions{
			AutoInit:        gitprovider.BoolVar(true),
			LicenseTemplate: gitprovider.LicenseTemplateVar(gitprovider.LicenseTemplateApache2),
		})
		Expect(err).ToNot(HaveOccurred())

		getRepoRef, err := client.OrgRepositories().Get(ctx, repoRef)
		Expect(err).ToNot(HaveOccurred())

		validateOrgRepo(repo, getRepoRef.Repository())

		// Verify that we can get clone url for the repo
		url := getRepoRef.Repository().GetCloneURL(gitprovider.TransportTypeHTTPS)
		Expect(url).ToNot(BeEmpty())
		fmt.Println("Clone URL: ", url)

		sshURL = getRepoRef.Repository().GetCloneURL(gitprovider.TransportTypeSSH)
		Expect(url).ToNot(BeEmpty())
		fmt.Println("Clone ssh URL: ", sshURL)

		// Expect the two responses (one from POST and one from GET to have equal "spec")
		getSpec := repositoryFromAPI(getRepoRef.APIObject().(*git.GitRepository))
		postSpec := repositoryFromAPI(repo.APIObject().(*git.GitRepository))
		Expect(getSpec.Equals(postSpec)).To(BeTrue())
	})

	It("should error at creation time if the org repo already does exist", func() {
		// Get the test organization
		orgRef := newOrgRef(testOrgName)
		testOrg, err := client.Organizations().Get(ctx, orgRef)
		Expect(err).ToNot(HaveOccurred())

		repoRef := newRepoRef(testOrg.Organization(), testOrgRepoName)
		_, err = client.OrgRepositories().Get(ctx, repoRef)
		Expect(err).ToNot(HaveOccurred())

		_, err = client.OrgRepositories().Create(ctx, repoRef, gitprovider.RepositoryInfo{})
		Expect(err).To(MatchError(gitprovider.ErrAlreadyExists))
	})

	It("should not update if the org repo already exists when reconciling", func() {
		// Get the test organization
		orgRef := newOrgRef(testOrgName)
		testOrg, err := client.Organizations().Get(ctx, orgRef)
		Expect(err).ToNot(HaveOccurred())

		repoRef := newRepoRef(testOrg.Organization(), testOrgRepoName)
		repo, err := client.OrgRepositories().Get(ctx, repoRef)
		Expect(err).ToNot(HaveOccurred())

		// No-op reconcile
		resp, actionTaken, err := client.OrgRepositories().Reconcile(ctx, repo.Repository().(gitprovider.OrgRepositoryRef), gitprovider.RepositoryInfo{
			Name:       gitprovider.StringVar(testOrgRepoName),
			Visibility: gitprovider.RepositoryVisibilityVar(gitprovider.RepositoryVisibilityPrivate),
		})

		reflect.DeepEqual(repositoryFromAPI(resp.APIObject().(*git.GitRepository)), gitprovider.RepositoryInfo{
			Name:       gitprovider.StringVar(testOrgRepoName),
			Visibility: gitprovider.RepositoryVisibilityVar(gitprovider.RepositoryVisibilityPrivate),
		})

		Expect(err).ToNot(HaveOccurred())
		Expect(actionTaken).To(BeFalse())
		// no-op set & reconcile
		Expect(resp.Set(resp.Get())).ToNot(HaveOccurred())
		actionTaken, err = resp.Reconcile(ctx)
		Expect(err).ToNot(HaveOccurred())
		Expect(actionTaken).To(BeFalse())

		// Update reconcile
		newName := "new-repo"
		req := resp.Get()
		req.Name = gitprovider.StringVar(newName)
		Expect(resp.Set(req)).ToNot(HaveOccurred())
		actionTaken, err = resp.Reconcile(ctx)
		// Expect the update to succeed, and modify the state
		Expect(err).ToNot(HaveOccurred())
		Expect(actionTaken).To(BeTrue())
		Expect(*resp.Get().Name).To(Equal(newName))

		// Delete the repository and later re-create
		Expect(resp.Delete(ctx)).ToNot(HaveOccurred())

		var newRepo gitprovider.OrgRepository
		retryOp := testutils.NewRetry()
		Eventually(func() bool {
			var err error
			// Reconcile and create
			newRepo, actionTaken, err = client.OrgRepositories().Reconcile(ctx, repoRef, gitprovider.RepositoryInfo{
				Description: gitprovider.StringVar(defaultDescription),
			})
			return retryOp.IsRetryable(err, fmt.Sprintf("reconcile org repository: %s", repoRef.RepositoryName))
		}, retryOp.Timeout(), retryOp.Interval()).Should(BeTrue())

		Expect(actionTaken).To(BeTrue())
		validateOrgRepo(newRepo, repo.Repository().(gitprovider.OrgRepositoryRef))
	})

	It("should be possible to create a pr for an org repository", func() {
		// Get the test organization
		orgRef := newOrgRef(testOrgName)
		testOrg, err := client.Organizations().Get(ctx, orgRef)
		Expect(err).ToNot(HaveOccurred())

		testRepoName = fmt.Sprintf("test-org-repo-%03d", rand.Intn(1000))
		fmt.Print("Creating repository - create pr ", testOrgRepoName, "...")
		repoRef := newRepoRef(testOrg.Organization(), testRepoName)
		// Create a new repo

		repo, err := client.OrgRepositories().Create(ctx, repoRef,
			gitprovider.RepositoryInfo{
				Visibility: gitprovider.RepositoryVisibilityVar(gitprovider.RepositoryVisibilityPrivate),
			})
		Expect(err).ToNot(HaveOccurred())

		var commits []gitprovider.Commit = []gitprovider.Commit{}
		retryOp := testutils.NewRetry()
		Eventually(func() bool {
			var err error
			//for Azure devops you'll need to create a commit first
			path := "setup/config.txt"
			content := "yaml content"
			files := []gitprovider.CommitFile{
				{
					Path:    &path,
					Content: &content,
				},
			}

			createCommits, _ := repo.Commits().Create(ctx, defaultBranch, "initial commit", files)
			//get the list of commits
			commits, err = repo.Commits().ListPage(ctx, defaultBranch, 1, 0)
			if err != nil && len(commits) == 0 {

				err = errors.New("empty commits list")

				Expect(err).ToNot(HaveOccurred())
				Expect(createCommits).ToNot(BeNil())

			}

			return retryOp.IsRetryable(err, fmt.Sprintf("get commits, repository: %s", repo.Repository().GetRepository()))
		}, retryOp.Timeout(), retryOp.Interval()).Should(BeTrue())

		latestCommit := commits[0]

		branchName := fmt.Sprintf("test-branch-%03d", rand.Intn(1000))
		branchName2 := fmt.Sprintf("test-branch-%03d", rand.Intn(1000))

		err = repo.Branches().Create(ctx, branchName, latestCommit.Get().Sha)
		Expect(err).ToNot(HaveOccurred())

		err = repo.Branches().Create(ctx, branchName2, "wrong-sha")
		Expect(err).To(HaveOccurred())

		path := "setup/config2.txt"
		content := "yaml content"
		files := []gitprovider.CommitFile{
			{
				Path:    &path,
				Content: &content,
			},
		}

		_, err = repo.Commits().Create(ctx, branchName, "added config file", files)
		Expect(err).ToNot(HaveOccurred())

		prTitle := "Added config file"
		prDesc := "added config file"
		pr, err := repo.PullRequests().Create(ctx, prTitle, branchName, defaultBranch, prDesc)
		Expect(err).ToNot(HaveOccurred())
		Expect(pr.Get().WebURL).ToNot(BeEmpty())
		Expect(pr.Get().SourceBranch).To(Equal("refs/heads/" + branchName))
		Expect(pr.Get().Title).To(Equal(prTitle))
		Expect(pr.Get().Description).To(Equal(prDesc))
	})
})

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

func newRepoRef(orgRef gitprovider.OrganizationRef, repoName string) gitprovider.OrgRepositoryRef {
	return gitprovider.OrgRepositoryRef{
		OrganizationRef: orgRef,
		RepositoryName:  repoName,
	}
}
