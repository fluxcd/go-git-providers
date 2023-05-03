//go:build e2e

/*
Copyright 2021 The Flux authors

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
	"math/rand"
	"reflect"

	"github.com/fluxcd/go-git-providers/gitprovider"
	"github.com/fluxcd/go-git-providers/gitprovider/testutils"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Stash Provider", func() {
	var (
		ctx context.Context = context.Background()
	)

	validateOrgRepo := func(repo gitprovider.OrgRepository, expectedRepo gitprovider.RepositoryRef) {
		info := repo.Get()
		// Expect certain fields to be set
		Expect(repo.Repository()).To(Equal(expectedRepo))
		Expect(*info.Description).To(Equal(defaultDescription))
		Expect(*info.Visibility).To(Equal(gitprovider.RepositoryVisibilityPrivate))
		Expect(*info.DefaultBranch).To(Equal(defaultBranch))

		// Expect high-level fields to match their underlying data
		internal := repo.APIObject().(*Repository)
		Expect(repo.Repository().GetRepository()).To(Equal(internal.Name))
		Expect(repo.Repository().(gitprovider.Slugger).Slug()).To(Equal(internal.Slug))
		Expect(repo.Repository().GetIdentity()).To(Equal(testOrgName))
		Expect(*info.Visibility).To(Equal(gitprovider.RepositoryVisibilityPrivate))
		Expect(*info.DefaultBranch).To(Equal(defaultBranch))
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
		for findOrgRepo(repos, testOrgRepoName) != nil {
			testOrgRepoName = fmt.Sprintf("test-org-repo-%03d", rand.Intn(1000))
		}

		fmt.Print("Creating repository ", testOrgRepoName, "...")
		// We know that a repo with this name doesn't exist in the organization, let's verify we get an
		// ErrNotFound
		repoRef := newOrgRepoRef(testOrg.Organization(), testOrgRepoName)
		httpsURL := repoRef.GetCloneURL(gitprovider.TransportTypeHTTPS)
		Expect(httpsURL).NotTo(Equal(""))
		_, err = client.OrgRepositories().Get(ctx, repoRef)
		Expect(err).To(MatchError(gitprovider.ErrNotFound))

		// Create a new repo
		repo, err := client.OrgRepositories().Create(ctx, repoRef, gitprovider.RepositoryInfo{
			Description: gitprovider.StringVar(defaultDescription),
			// Default visibility is private, no need to set this at least now
			Visibility:    gitprovider.RepositoryVisibilityVar(gitprovider.RepositoryVisibilityPrivate),
			DefaultBranch: gitprovider.StringVar(defaultBranch),
		}, &gitprovider.RepositoryCreateOptions{
			AutoInit:        gitprovider.BoolVar(true),
			LicenseTemplate: gitprovider.LicenseTemplateVar(gitprovider.LicenseTemplateApache2),
		})
		Expect(err).ToNot(HaveOccurred())

		getRepoRef, err := client.OrgRepositories().Get(ctx, repoRef)
		Expect(err).ToNot(HaveOccurred())

		validateOrgRepo(repo, getRepoRef.Repository())

		// Verify that we can get clone url for the repo
		if cloner, ok := getRepoRef.(gitprovider.CloneableURL); ok {
			url := cloner.GetCloneURL("scm", gitprovider.TransportTypeHTTPS)
			Expect(url).ToNot(BeEmpty())
			fmt.Println("Clone URL: ", url)

			sshURL := cloner.GetCloneURL("scm", gitprovider.TransportTypeSSH)
			Expect(url).ToNot(BeEmpty())
			fmt.Println("Clone ssh URL: ", sshURL)
		}

		// Expect the two responses (one from POST and one from GET to have equal "spec")
		getSpec := repositoryFromAPI(getRepoRef.APIObject().(*Repository))
		postSpec := repositoryFromAPI(repo.APIObject().(*Repository))
		Expect(getSpec.Equals(postSpec)).To(BeTrue())
	})

	It("should error at creation time if the org repo already does exist", func() {
		// Get the test organization
		orgRef := newOrgRef(testOrgName)
		testOrg, err := client.Organizations().Get(ctx, orgRef)
		Expect(err).ToNot(HaveOccurred())

		// get the repo first to be sure to get the slug
		repoRef := newOrgRepoRef(testOrg.Organization(), testOrgRepoName)
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

		// get the repo first to be sure to get the slug
		repoRef := newOrgRepoRef(testOrg.Organization(), testOrgRepoName)
		repo, err := client.OrgRepositories().Get(ctx, repoRef)
		Expect(err).ToNot(HaveOccurred())

		// No-op reconcile
		resp, actionTaken, err := client.OrgRepositories().Reconcile(ctx, repo.Repository().(gitprovider.OrgRepositoryRef), gitprovider.RepositoryInfo{
			Description:   gitprovider.StringVar(defaultDescription),
			DefaultBranch: gitprovider.StringVar(defaultBranch),
			Visibility:    gitprovider.RepositoryVisibilityVar(gitprovider.RepositoryVisibilityPrivate),
		})

		reflect.DeepEqual(repositoryFromAPI(resp.APIObject().(*Repository)), gitprovider.RepositoryInfo{
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
			newRepo, actionTaken, err = client.OrgRepositories().Reconcile(ctx, repoRef, gitprovider.RepositoryInfo{
				Description: gitprovider.StringVar(defaultDescription),
			}, &gitprovider.RepositoryCreateOptions{
				AutoInit:        gitprovider.BoolVar(true),
				LicenseTemplate: gitprovider.LicenseTemplateVar(gitprovider.LicenseTemplateMIT),
			})
			return retryOp.IsRetryable(err, fmt.Sprintf("reconcile org repository: %s", repoRef.RepositoryName))
		}, retryOp.Timeout(), retryOp.Interval()).Should(BeTrue())

		Expect(actionTaken).To(BeTrue())
		validateOrgRepo(newRepo, repo.Repository().(gitprovider.OrgRepositoryRef))
	})

	It("should update teams with access and permissions when reconciling", func() {

		// Get the test organization
		orgRef := newOrgRef(testOrgName)
		testOrg, err := client.Organizations().Get(ctx, orgRef)
		Expect(err).ToNot(HaveOccurred())

		// List all the teams with access to the org
		// There should be 1 existing subgroup already
		teams, err := testOrg.Teams().List(ctx)
		Expect(err).ToNot(HaveOccurred())
		Expect(len(teams)).To(Equal(1), "The 1 team wasn't there...")

		// First, check what repositories are available
		repos, err := client.OrgRepositories().List(ctx, testOrg.Organization())
		Expect(err).ToNot(HaveOccurred())

		// Generate an org repo name which doesn't exist already
		testSharedOrgRepoName = fmt.Sprintf("test-shared-org-repo-%03d", rand.Intn(1000))
		for findOrgRepo(repos, testSharedOrgRepoName) != nil {
			testSharedOrgRepoName = fmt.Sprintf("test-shared-org-repo-%03d", rand.Intn(1000))
		}

		// We know that a repo with this name doesn't exist in the organization, let's verify we get an
		// ErrNotFound
		sharedRepoRef := newOrgRepoRef(testOrg.Organization(), testSharedOrgRepoName)
		_, err = client.OrgRepositories().Get(ctx, sharedRepoRef)
		Expect(err).To(MatchError(gitprovider.ErrNotFound))

		// Create a new repo
		repo, err := client.OrgRepositories().Create(ctx, sharedRepoRef, gitprovider.RepositoryInfo{
			Description: gitprovider.StringVar(defaultDescription),
			// Default visibility is private, no need to set this at least now
			Visibility:    gitprovider.RepositoryVisibilityVar(gitprovider.RepositoryVisibilityPrivate),
			DefaultBranch: gitprovider.StringVar(defaultBranch),
		}, &gitprovider.RepositoryCreateOptions{
			AutoInit:        gitprovider.BoolVar(true),
			LicenseTemplate: gitprovider.LicenseTemplateVar(gitprovider.LicenseTemplateApache2),
		})
		Expect(err).ToNot(HaveOccurred())

		getRepoRef, err := client.OrgRepositories().Get(ctx, sharedRepoRef)
		Expect(err).ToNot(HaveOccurred())

		validateOrgRepo(repo, getRepoRef.Repository())

		// 2 teams should have access to the repo
		projectTeams, err := repo.TeamAccess().List(ctx)
		Expect(err).ToNot(HaveOccurred())
		// go-git-provider-testing & fluxcd-test-team group already exists, so we should have 2 teams
		Expect(len(projectTeams)).To(Equal(1))

		// Add a team to the project
		permission := gitprovider.RepositoryPermissionPull
		_, err = repo.TeamAccess().Create(ctx, gitprovider.TeamAccessInfo{
			Name:       testTeamName,
			Permission: &permission,
		})
		Expect(err).ToNot(HaveOccurred())

		// List all the teams with access to the project
		// Only
		projectTeams, err = repo.TeamAccess().List(ctx)
		Expect(err).ToNot(HaveOccurred())
		Expect(len(projectTeams)).To(Equal(2), "Project teams didn't equal 2")
		var firstTeam gitprovider.TeamAccess
		for _, v := range projectTeams {
			if v.Get().Name == testTeamName {
				firstTeam = v
			}
		}
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

		// Assert that reconciling works
		teamInfo := gitprovider.TeamAccessInfo{
			Name:       testTeamName,
			Permission: &pushPermission,
		}

		ta, actionTaken, err = repo.TeamAccess().Reconcile(ctx, teamInfo)
		Expect(err).ToNot(HaveOccurred())
		Expect(actionTaken).To(Equal(false))
	})

	It("should create, delete and reconcile deploy keys", func() {
		// Get the test organization
		orgRef := newOrgRef(testOrgName)
		testOrg, err := client.Organizations().Get(ctx, orgRef)
		Expect(err).ToNot(HaveOccurred())
		//
		testDeployKeyName := "test-deploy-key"
		SharedRepoRef := newOrgRepoRef(testOrg.Organization(), testSharedOrgRepoName)

		orgRepo, err := client.OrgRepositories().Get(ctx, SharedRepoRef)
		Expect(err).ToNot(HaveOccurred())

		// List keys should return 0
		keys, err := orgRepo.DeployKeys().List(ctx)
		Expect(err).ToNot(HaveOccurred())
		Expect(len(keys)).To(Equal(0))

		rsaGen := testutils.NewRSAGenerator(2154)
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
		Expect(string(getKey.Get().Key)).To(Equal(deployKeyStr))
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

	It("should be possible to create a pr for an org repository", func() {
		// Get the test organization
		orgRef := newOrgRef(testOrgName)
		testOrg, err := client.Organizations().Get(ctx, orgRef)
		Expect(err).ToNot(HaveOccurred())

		testRepoName = fmt.Sprintf("test-org-repo2-%03d", rand.Intn(1000))
		repoRef := newOrgRepoRef(testOrg.Organization(), testRepoName)
		description := "test description"
		// Create a new repo
		orgRepo, err := client.OrgRepositories().Create(ctx, repoRef,
			gitprovider.RepositoryInfo{
				Description:   &description,
				Visibility:    gitprovider.RepositoryVisibilityVar(gitprovider.RepositoryVisibilityPrivate),
				DefaultBranch: gitprovider.StringVar(defaultBranch),
			},
			&gitprovider.RepositoryCreateOptions{
				AutoInit: gitprovider.BoolVar(false),
			})
		Expect(err).ToNot(HaveOccurred())

		var commits []gitprovider.Commit = []gitprovider.Commit{}
		retryOp := testutils.NewRetry()
		Eventually(func() bool {
			var err error
			commits, err = orgRepo.Commits().ListPage(ctx, defaultBranch, 1, 0)
			if err == nil && len(commits) == 0 {
				err = errors.New("empty commits list")
			}
			return retryOp.IsRetryable(err, fmt.Sprintf("get commits, repository: %s", orgRepo.Repository().GetRepository()))
		}, retryOp.Timeout(), retryOp.Interval()).Should(BeTrue())

		latestCommit := commits[0]

		branchName := fmt.Sprintf("test-branch-%03d", rand.Intn(1000))
		branchName2 := fmt.Sprintf("test-branch-%03d", rand.Intn(1000))

		err = orgRepo.Branches().Create(ctx, branchName, latestCommit.Get().Sha)
		Expect(err).ToNot(HaveOccurred())

		err = orgRepo.Branches().Create(ctx, branchName2, "wrong-sha")
		Expect(err).To(HaveOccurred())

		path := "setup/config.txt"
		content := "yaml content"
		files := []gitprovider.CommitFile{
			{
				Path:    &path,
				Content: &content,
			},
		}

		_, err = orgRepo.Commits().Create(ctx, branchName, "added config file", files)
		Expect(err).ToNot(HaveOccurred())

		prTitle := "Added config file"
		prDesc := "added config file"
		pr, err := orgRepo.PullRequests().Create(ctx, prTitle, branchName, defaultBranch, prDesc)
		Expect(err).ToNot(HaveOccurred())
		Expect(pr.Get().WebURL).ToNot(BeEmpty())
		Expect(pr.Get().SourceBranch).To(Equal(branchName))
		Expect(pr.Get().Title).To(Equal(prTitle))
		Expect(pr.Get().Description).To(Equal(prDesc))
	})
})

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

func newOrgRepoRef(orgRef gitprovider.OrganizationRef, repoName string) gitprovider.OrgRepositoryRef {
	return gitprovider.OrgRepositoryRef{
		OrganizationRef: orgRef,
		RepositoryName:  repoName,
	}
}
