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

	validateUserRepo := func(repo gitprovider.UserRepository, expectedRepoRef gitprovider.RepositoryRef) {
		info := repo.Get()
		// Expect certain fields to be set
		Expect(repo.Repository()).To(Equal(expectedRepoRef))
		Expect(*info.Description).To(Equal(defaultDescription))
		Expect(*info.Visibility).To(Equal(gitprovider.RepositoryVisibilityPrivate))
		Expect(*info.DefaultBranch).To(Equal(defaultBranch))
		// Expect high-level fields to match their underlying data
		internal := repo.APIObject().(*Repository)
		Expect(repo.Repository().GetRepository()).To(Equal(internal.Name))
		Expect(repo.Repository().GetIdentity()).To(Equal(stashUser))
		if !internal.Public {
			Expect(*info.Visibility).To(Equal(gitprovider.RepositoryVisibilityPrivate))
		}
		Expect(*info.DefaultBranch).To(Equal(defaultBranch))
	}

	It("should be possible to create a user repo", func() {
		// First, check what repositories are available
		repos, err := client.UserRepositories().List(ctx, newUserRef(stashUser))
		Expect(err).ToNot(HaveOccurred())

		// Generate a repository name which doesn't exist already
		testRepoName = fmt.Sprintf("test-user-repo-%03d", rand.Intn(1000))
		for findUserRepo(repos, testRepoName) != nil {
			testRepoName = fmt.Sprintf("test-user-repo-%03d", rand.Intn(1000))
		}

		fmt.Print("Creating repository ", testRepoName, "...")
		// We know that a repo with this name doesn't exist in the organization, let's verify we get an
		// ErrNotFound
		repoRef := newUserRepoRef(stashUser, testRepoName)
		_, err = client.UserRepositories().Get(ctx, repoRef)
		Expect(err).To(MatchError(gitprovider.ErrNotFound))

		// Create a new repo
		repo, err := client.UserRepositories().Create(ctx, repoRef, gitprovider.RepositoryInfo{
			Description: gitprovider.StringVar(defaultDescription),
			// Default visibility is private, no need to set this at least now
			Visibility:    gitprovider.RepositoryVisibilityVar(gitprovider.RepositoryVisibilityPrivate),
			DefaultBranch: gitprovider.StringVar(defaultBranch),
		}, &gitprovider.RepositoryCreateOptions{
			AutoInit:        gitprovider.BoolVar(true),
			LicenseTemplate: gitprovider.LicenseTemplateVar(gitprovider.LicenseTemplateApache2),
		})
		Expect(err).ToNot(HaveOccurred())

		getRepoRef, err := client.UserRepositories().Get(ctx, repoRef)
		Expect(err).ToNot(HaveOccurred())

		// Verify that we can get clone url for the repo
		if cloner, ok := getRepoRef.(gitprovider.CloneableURL); ok {
			url := cloner.GetCloneURL("scm", gitprovider.TransportTypeHTTPS)
			Expect(url).ToNot(BeEmpty())
			fmt.Println("Clone URL: ", url)

			sshURL := cloner.GetCloneURL("scm", gitprovider.TransportTypeSSH)
			Expect(url).ToNot(BeEmpty())
			fmt.Println("Clone ssh URL: ", sshURL)
		}

		validateUserRepo(repo, getRepoRef.Repository())

		getRepo, err := client.UserRepositories().Get(ctx, repoRef)
		Expect(err).ToNot(HaveOccurred())
		// Expect the two responses (one from POST and one from GET to have equal "spec")
		getSpec := repositoryFromAPI(getRepo.APIObject().(*Repository))
		postSpec := repositoryFromAPI(repo.APIObject().(*Repository))
		Expect(getSpec.Equals(postSpec)).To(BeTrue())
	})

	It("should error at creation time if the user repo already does exist", func() {
		repoRef := newUserRepoRef(stashUser, testRepoName)
		_, err := client.UserRepositories().Get(ctx, repoRef)
		Expect(err).ToNot(HaveOccurred())

		_, err = client.UserRepositories().Create(ctx, repoRef, gitprovider.RepositoryInfo{})
		Expect(err).To(MatchError(gitprovider.ErrAlreadyExists))
	})

	It("should update if the user repo already exists when reconciling", func() {
		// get the repo first to be sure to get the slug
		repoRef := newUserRepoRef(stashUser, testRepoName)
		repo, err := client.UserRepositories().Get(ctx, repoRef)
		Expect(err).ToNot(HaveOccurred())
		// No-op reconcile
		resp, actionTaken, err := client.UserRepositories().Reconcile(ctx, repo.Repository().(gitprovider.UserRepositoryRef), gitprovider.RepositoryInfo{
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
			// Reconcile and create
			newRepo, actionTaken, err = client.UserRepositories().Reconcile(ctx, repoRef, gitprovider.RepositoryInfo{
				Description:   gitprovider.StringVar(defaultDescription),
				DefaultBranch: gitprovider.StringVar(defaultBranch),
			}, &gitprovider.RepositoryCreateOptions{
				AutoInit:        gitprovider.BoolVar(true),
				LicenseTemplate: gitprovider.LicenseTemplateVar(gitprovider.LicenseTemplateMIT),
			})
			return retryOp.IsRetryable(err, fmt.Sprintf("new user repository: %s", repoRef.RepositoryName))
		}, retryOp.Timeout(), retryOp.Interval()).Should(BeTrue())

		// Expect the create to succeed, and have modified the state. Also validate the newRepo data
		Expect(actionTaken).To(BeTrue())
		validateUserRepo(newRepo, repo.Repository().(gitprovider.UserRepositoryRef))
	})

	It("should be possible to create a pr for a user repository", func() {
		testRepoName = fmt.Sprintf("test-user-repo2-%03d", rand.Intn(1000))
		repoRef := newUserRepoRef(stashUser, testRepoName)
		description := "test description"
		// Create a new repo
		userRepo, err := client.UserRepositories().Create(ctx, repoRef,
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

		prTitle := "Added config file"
		prDesc := "added config file"
		pr, err := userRepo.PullRequests().Create(ctx, prTitle, branchName, defaultBranch, prDesc)
		Expect(err).ToNot(HaveOccurred())
		Expect(pr.Get().WebURL).ToNot(BeEmpty())
		Expect(pr.Get().SourceBranch).To(Equal(branchName))
		Expect(pr.Get().Title).To(Equal(prTitle))
		Expect(pr.Get().Description).To(Equal(prDesc))

		editedPR, err := userRepo.PullRequests().Edit(ctx, pr.Get().Number, gitprovider.EditOptions{
			Title: gitprovider.StringVar("a new title"),
		})
		Expect(err).ToNot(HaveOccurred(), "error editing PR")
		Expect(editedPR).ToNot(BeNil(), "returned PR should never be nil if no error was returned")

		// edit one more time to make sure the version number is taken into account

		editedPR, err = userRepo.PullRequests().Edit(ctx, pr.Get().Number, gitprovider.EditOptions{
			Title: gitprovider.StringVar("another new title"),
		})
		Expect(err).ToNot(HaveOccurred(), "error editing PR a second time")
		Expect(editedPR).ToNot(BeNil(), "returned PR should never be nil if no error was returned")

		// List PRs
		prs, err := userRepo.PullRequests().List(ctx)
		Expect(err).ToNot(HaveOccurred())
		Expect(len(prs)).To(Equal(1))

		apiObject := prs[0].APIObject()
		stashPR, ok := apiObject.(*PullRequest)
		Expect(ok).To(BeTrue(), "API object of PullRequest has unexpected type %q", reflect.TypeOf(apiObject))
		Expect(stashPR.Title).To(Equal("another new title"))

		// Merge PR
		id := pr.APIObject().(*PullRequest).ID
		err = userRepo.PullRequests().Merge(ctx, id, "merge", "merged")
		Expect(err).ToNot(HaveOccurred())
	})
})

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

func newUserRepoRef(userLogin, repoName string) gitprovider.UserRepositoryRef {
	return gitprovider.UserRepositoryRef{
		UserRef:        newUserRef(userLogin),
		RepositoryName: repoName,
	}
}
