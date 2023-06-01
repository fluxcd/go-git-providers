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

package gitea

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
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

	validateUserRepo := func(repo gitprovider.UserRepository, expectedRepoRef gitprovider.RepositoryRef) {
		info := repo.Get()
		// Expect certain fields to be set
		Expect(repo.Repository()).To(Equal(expectedRepoRef))
		Expect(*info.Description).To(Equal(defaultDescription))
		Expect(*info.Visibility).To(Equal(gitprovider.RepositoryVisibilityPrivate))
		Expect(*info.DefaultBranch).To(Equal(defaultBranch))
		// Expect high-level fields to match their underlying data
		internal := repo.APIObject().(*gitea.Repository)
		Expect(repo.Repository().GetRepository()).To(Equal(internal.Name))
		Expect(repo.Repository().GetIdentity()).To(Equal(giteaUser))
		internalPrivatestr := gitea.VisibleTypePublic
		if internal.Private {
			internalPrivatestr = gitea.VisibleTypePrivate
		}
		Expect(string(*info.Visibility)).To(Equal(string(internalPrivatestr)))
		Expect(*info.DefaultBranch).To(Equal(defaultBranch))
	}

	It("should be possible to create a user repository", func() {
		// First, check what repositories are available
		repos, err := c.UserRepositories().List(ctx, newUserRef(giteaUser))
		Expect(err).ToNot(HaveOccurred())

		// Generate a repository name which doesn't exist already
		testRepoName = fmt.Sprintf("test-user-repo-%03d", rand.Intn(1000))
		for findUserRepo(repos, testRepoName) != nil {
			testRepoName = fmt.Sprintf("test-user-repo-%03d", rand.Intn(1000))
		}

		fmt.Print("Creating repository ", testRepoName, "...")
		repoRef := newUserRepoRef(giteaUser, testRepoName)

		// Check that the repository doesn't exist
		_, err = c.UserRepositories().Get(ctx, repoRef)
		Expect(err).To(MatchError(gitprovider.ErrNotFound))

		repoInfo := gitprovider.RepositoryInfo{
			Description:   gitprovider.StringVar(defaultDescription),
			Visibility:    gitprovider.RepositoryVisibilityVar(gitprovider.RepositoryVisibilityPrivate),
			DefaultBranch: gitprovider.StringVar(defaultBranch),
		}

		// Create a repository
		repo, err := c.UserRepositories().Create(ctx, repoRef, repoInfo, &gitprovider.RepositoryCreateOptions{
			AutoInit:        gitprovider.BoolVar(true),
			LicenseTemplate: gitprovider.LicenseTemplateVar(gitprovider.LicenseTemplateApache2),
		})
		Expect(err).ToNot(HaveOccurred())

		getRepo, err := c.UserRepositories().Get(ctx, repoRef)
		Expect(errors.Is(err, gitprovider.ErrNotFound)).To(BeTrue())

		validateUserRepo(repo, getRepo.Repository())

		// Expect the two responses (one from POST and one from GET to have equal "spec")
		getSpec := newGiteaRepositorySpec(getRepo.APIObject().(*gitea.Repository))
		postSpec := newGiteaRepositorySpec(repo.APIObject().(*gitea.Repository))
		Expect(getSpec.Equals(postSpec)).To(BeTrue())
	})

	It("should error at creation time if the repo already does exist", func() {
		repoRef := newUserRepoRef(giteaUser, testRepoName)
		_, err := c.UserRepositories().Get(ctx, repoRef)
		Expect(err).ToNot(HaveOccurred())

		_, err = c.UserRepositories().Create(ctx, repoRef, gitprovider.RepositoryInfo{})
		Expect(errors.Is(err, gitprovider.ErrAlreadyExists)).To(BeTrue())
	})

	It("should update if the repository already exists when reconciling", func() {
		repoRef := newUserRepoRef(giteaUser, testRepoName)
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

		var newRepo gitprovider.UserRepository
		retryOp := testutils.NewRetry()
		Eventually(func() bool {
			var err error
			newRepo, actionTaken, err = c.UserRepositories().Reconcile(ctx, repoRef, gitprovider.RepositoryInfo{
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
		validateUserRepo(newRepo, repoRef)

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

	It("should validate that the token has the correct permissions", func() {
		hasPermission, err := c.HasTokenPermission(ctx, 0)
		Expect(err).To(Equal(gitprovider.ErrNoProviderSupport))
		Expect(hasPermission).To(Equal(false))

		hasPermission, err = c.HasTokenPermission(ctx, gitprovider.TokenPermissionRWRepository)
		// Gitea doesn't yet support token permissions
		Expect(err).To(Equal(gitprovider.ErrNoProviderSupport))
		Expect(hasPermission).To(Equal(false))
	})

	It("should be possible to create a pr for a user repository", func() {
		testRepoName = fmt.Sprintf("test-user-repo2-%03d", rand.Intn(1000))
		repoRef := newUserRepoRef(giteaUser, testRepoName)
		description := "test description"
		// Create a new repo
		userRepo, err := c.UserRepositories().Create(ctx, repoRef,
			gitprovider.RepositoryInfo{
				Description:   &description,
				Visibility:    gitprovider.RepositoryVisibilityVar(gitprovider.RepositoryVisibilityPrivate),
				DefaultBranch: gitprovider.StringVar(defaultBranch),
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

		err = userRepo.Branches().Create(ctx, branchName, latestCommit.Get().Sha)
		Expect(err).ToNot(HaveOccurred())

		err = userRepo.Branches().Create(ctx, branchName2, "wrong-sha")
		Expect(err).To(HaveOccurred())

		// see issue https://github.com/go-gitea/gitea/issues/14619#
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

		pr, err := userRepo.PullRequests().Create(ctx, "Added config file", branchName, defaultBranch, "added config file")
		Expect(err).ToNot(HaveOccurred())
		Expect(pr.Get().WebURL).ToNot(BeEmpty())
		Expect(pr.Get().Merged).To(BeFalse())

		prs, err := userRepo.PullRequests().List(ctx)
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

		pr, err = userRepo.PullRequests().Create(ctx, "Added second config file", branchName, defaultBranch, "added second config file")
		Expect(err).ToNot(HaveOccurred())
		Expect(pr.Get().WebURL).ToNot(BeEmpty())
		Expect(pr.Get().Merged).To(BeFalse())

		err = userRepo.PullRequests().Merge(ctx, pr.Get().Number, gitprovider.MergeMethodMerge, "merged")
		Expect(err).ToNot(HaveOccurred())

		getPR, err = userRepo.PullRequests().Get(ctx, pr.Get().Number)
		Expect(err).ToNot(HaveOccurred())

		Expect(getPR.Get().Merged).To(BeTrue())
	})

	It("should be possible to download files from path and branch specified", func() {
		testRepoName = fmt.Sprintf("test-repo-tree-%03d", rand.Intn(1000))
		userRepoRef := newUserRepoRef(giteaUser, testRepoName)
		repo, err := c.UserRepositories().Create(ctx, userRepoRef, gitprovider.RepositoryInfo{
			DefaultBranch: gitprovider.StringVar(defaultBranch),
			Description:   gitprovider.StringVar(defaultDescription),
		}, &gitprovider.RepositoryCreateOptions{
			AutoInit: gitprovider.BoolVar(false),
		})
		Expect(err).ToNot(HaveOccurred())

		validateUserRepo(repo, userRepoRef)

		userRepo, err := c.UserRepositories().Get(ctx, userRepoRef)
		Expect(err).ToNot(HaveOccurred())

		defaultBranch := userRepo.Get().DefaultBranch

		// see commit/pr issue above https://github.com/go-gitea/gitea/issues/14619#
		path0 := "cluster/machine1.yaml"
		content0 := "machine1 yaml content"
		path1 := "cluster/machine2.yaml"
		content1 := "machine2 yaml content"

		// first commit
		cf := []gitprovider.CommitFile{
			{
				Path:    &path0,
				Content: &content0,
			},
		}

		_, err = userRepo.Commits().Create(ctx, *defaultBranch, "added config files", cf)
		Expect(err).ToNot(HaveOccurred())

		// second commit
		cf = []gitprovider.CommitFile{
			{
				Path:    &path1,
				Content: &content1,
			},
		}

		_, err = userRepo.Commits().Create(ctx, *defaultBranch, "added config files", cf)
		Expect(err).ToNot(HaveOccurred())

		downloadedFiles, err := userRepo.Files().Get(ctx, "cluster", *defaultBranch)
		Expect(err).ToNot(HaveOccurred())

		for ind, downloadedFile := range downloadedFiles {
			// Expect(*downloadedFile).To(Equal(files[ind]))
			Expect(*downloadedFile).ToNot(BeNil())
			Expect(ind).ToNot(BeZero())
		}

	})
})
