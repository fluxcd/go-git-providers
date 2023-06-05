//go:build e2e

/*
Copyright 2023 The Flux CD contributors.

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

package gitea

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"code.gitea.io/sdk/gitea"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/fluxcd/go-git-providers/gitprovider"
	"github.com/fluxcd/go-git-providers/gitprovider/testutils"
)

var _ = Describe("Gitea Provider", func() {
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
		internal := repo.APIObject().(*gitea.Repository)
		Expect(repo.Repository().GetRepository()).To(Equal(internal.Name))
		Expect(repo.Repository().GetIdentity()).To(Equal(testOrgName))
		Expect(*info.Visibility).To(Equal(gitprovider.RepositoryVisibilityPrivate))
		Expect(*info.DefaultBranch).To(Equal(defaultBranch))
	}

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
		getSpec := newGiteaRepositorySpec(getRepo.APIObject().(*gitea.Repository))
		postSpec := newGiteaRepositorySpec(repo.APIObject().(*gitea.Repository))
		Expect(getSpec.Equals(postSpec)).To(BeTrue())
	})

	It("should error at creation time if the repo already does exist", func() {
		repoRef := newOrgRepoRef(testOrgName, testOrgRepoName)
		_, err := c.OrgRepositories().Create(ctx, repoRef, gitprovider.RepositoryInfo{})
		Expect(errors.Is(err, gitprovider.ErrAlreadyExists)).To(BeTrue())
	})

	It("should be possible to add org repo to 20 teams", func() {
		testOrgRef := newOrgRef(testOrgName)
		testOrg, err := c.Organizations().Get(ctx, testOrgRef)
		Expect(err).ToNot(HaveOccurred())
		testOrgRepoRef := newOrgRepoRef(testOrgName, testOrgRepoName)
		testOrgRepos, err := c.OrgRepositories().Get(ctx, testOrgRepoRef)
		Expect(err).ToNot(HaveOccurred())
		ta, err := testOrgRepos.TeamAccess().Get(ctx, "Owners")
		Expect(err).ToNot(HaveOccurred())
		teamsClient := testOrg.Teams()
		ownerTaInfo := ta.Get()
		for i := 0; i < 10; i++ {
			newTeamName := fmt.Sprintf("test-team-%d", i+1)
			if findOrgTeam(ctx, teamsClient, newTeamName) == nil {
				continue
			}
			ownerTaInfo.Name = newTeamName
			testOrgRepos.TeamAccess().Create(ctx, ownerTaInfo)
		}
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
		r := newRepo.APIObject().(*gitea.Repository)
		r.Internal = true

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
	It("should create, delete and reconcile deploy keys", func() {
		// Get the test organization
		orgRef := newOrgRef(testOrgName)
		testOrg, err := c.Organizations().Get(ctx, orgRef)
		Expect(err).ToNot(HaveOccurred())

		testDeployKeyName := "test-deploy-key"
		repoRef := newOrgRepoRef(*testOrg.Get().Name, testOrgRepoName)

		orgRepo, err := c.OrgRepositories().Get(ctx, repoRef)
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

		deployKeyStr := strings.TrimSpace(string(testDeployKeyInfo.Key))
		key := strings.TrimSpace(string(getKey.Get().Key))
		Expect(key).To(Equal(deployKeyStr))
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
	It("should update teams with access and permissions when reconciling", func() {
		// Get the test organization
		orgRef := newOrgRef(testOrgName)
		testOrg, err := c.Organizations().Get(ctx, orgRef)
		Expect(err).ToNot(HaveOccurred())

		// List all the teams with access to the org
		// There should be 2 existing subgroup already
		teams, err := testOrg.Teams().List(ctx)
		Expect(err).ToNot(HaveOccurred())
		Expect(len(teams)).To(Equal(2), "The 2 team wasn't there...")

		// First, check what repositories are available
		repos, err := c.OrgRepositories().List(ctx, testOrg.Organization())
		Expect(err).ToNot(HaveOccurred())

		// Generate an org repo name which doesn't exist already
		testSharedOrgRepoName = fmt.Sprintf("test-shared-org-repo-%03d", rand.Intn(1000))
		for findOrgRepo(repos, testSharedOrgRepoName) != nil {
			testSharedOrgRepoName = fmt.Sprintf("test-shared-org-repo-%03d", rand.Intn(1000))
		}

		// We know that a repo with this name doesn't exist in the organization, let's verify we get an
		// ErrNotFound
		sharedRepoRef := newOrgRepoRef(*testOrg.Get().Name, testSharedOrgRepoName)
		_, err = c.OrgRepositories().Get(ctx, sharedRepoRef)
		Expect(err).To(MatchError(gitprovider.ErrNotFound))

		// Create a new repo
		repo, err := c.OrgRepositories().Create(ctx, sharedRepoRef, gitprovider.RepositoryInfo{
			Description: gitprovider.StringVar(defaultDescription),
			// Default visibility is private, no need to set this at least now
			Visibility:    gitprovider.RepositoryVisibilityVar(gitprovider.RepositoryVisibilityPrivate),
			DefaultBranch: gitprovider.StringVar(defaultBranch),
		}, &gitprovider.RepositoryCreateOptions{
			AutoInit:        gitprovider.BoolVar(true),
			LicenseTemplate: gitprovider.LicenseTemplateVar(gitprovider.LicenseTemplateApache2),
		})
		Expect(err).ToNot(HaveOccurred())

		getRepoRef, err := c.OrgRepositories().Get(ctx, sharedRepoRef)
		Expect(err).ToNot(HaveOccurred())

		validateOrgRepo(repo, getRepoRef.Repository())

		// 1 teams should have access to the repo
		projectTeams, err := repo.TeamAccess().List(ctx)
		Expect(err).ToNot(HaveOccurred())
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
	})
})
