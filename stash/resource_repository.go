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
	"fmt"

	"github.com/fluxcd/go-git-providers/gitprovider"
)

const defaultClonePrefix = "scm"

func newUserRepository(ctx *clientContext, apiObj *Repository, ref gitprovider.RepositoryRef) *userRepository {
	return &userRepository{
		c: &UserRepositoriesClient{
			clientContext: ctx,
		},
		repository: *apiObj,
		ref:        ref,
		deployKeys: &DeployKeyClient{
			clientContext: ctx,
			ref:           ref,
		},
		commits: &CommitClient{
			clientContext: ctx,
			ref:           ref,
		},
		branches: &BranchClient{
			clientContext: ctx,
			ref:           ref,
		},
		pullRequests: &PullRequestClient{
			clientContext: ctx,
			ref:           ref,
		},
	}
}

var _ gitprovider.UserRepository = &userRepository{}

type userRepository struct {
	repository   Repository
	ref          gitprovider.RepositoryRef
	c            *UserRepositoriesClient
	deployKeys   *DeployKeyClient
	branches     *BranchClient
	pullRequests *PullRequestClient
	commits      *CommitClient
}

func (r *userRepository) Branches() gitprovider.BranchClient {
	return r.branches
}

func (r *userRepository) Commits() gitprovider.CommitClient {
	return r.commits
}

func (r *userRepository) PullRequests() gitprovider.PullRequestClient {
	return r.pullRequests
}

func (r *userRepository) Get() gitprovider.RepositoryInfo {
	return repositoryFromAPI(&r.repository)
}

func (r *userRepository) Set(info gitprovider.RepositoryInfo) error {
	if err := info.ValidateInfo(); err != nil {
		return err
	}
	repositoryInfoToAPIObj(&info, &r.repository)
	return nil
}

func (r *userRepository) APIObject() interface{} {
	return &r.repository
}

func (r *userRepository) Repository() gitprovider.RepositoryRef {
	return r.ref
}

func (r *userRepository) DeployKeys() gitprovider.DeployKeyClient {
	return r.deployKeys
}

// The internal API object will be overridden with the received server data.
func (r *userRepository) Update(ctx context.Context) error {
	// update by calling client
	ref := r.ref.(gitprovider.UserRepositoryRef)
	apiObj, err := update(ctx, r.c.client, addTilde(ref.UserLogin), ref.Slug(), &r.repository, "")
	if err != nil {
		// Log the error and return it
		r.c.log.V(1).Error(err, "Error updating repository",
			"org", r.Repository().GetIdentity(),
			"repo", r.Repository().GetRepository())
		return err
	}

	r.repository = *apiObj

	return nil
}

// Reconcile makes sure the desired state in this object (called "req" here) becomes
// the actual state in the backing Git provider.

// If req doesn't exist under the hood, it is created (actionTaken == true).
// If req doesn't equal the actual state, the resource will be updated (actionTaken == true).
// If req is already the actual state, this is a no-op (actionTaken == false).
//
// The internal API object will be overridden with the received server data if actionTaken == true.
func (r *userRepository) Reconcile(ctx context.Context) (bool, error) {
	_, actionTaken, err := r.c.Reconcile(ctx, r.ref.(gitprovider.UserRepositoryRef), repositoryFromAPI(&r.repository))

	if err != nil {
		// Log the error and return it
		r.c.log.V(1).Error(err, "Error reconciling repository",
			"org", r.Repository().GetIdentity(),
			"repo", r.Repository().GetRepository(),
			"actionTaken", actionTaken)
		return actionTaken, err
	}

	return actionTaken, nil
}

// Delete deletes the current resource irreversibly.
// ErrNotFound is returned if the resource doesn't exist anymore.
func (r *userRepository) Delete(ctx context.Context) error {
	ref := r.ref.(gitprovider.UserRepositoryRef)
	return deleteRepository(ctx, r.c.client, addTilde(ref.UserLogin), ref.Slug())
}

// GetCloneURL returns a formatted string that can be used for cloning
// from a remote Git provider.
func (r *userRepository) GetCloneURL(prefix string, transport gitprovider.TransportType) string {
	if prefix == "" {
		prefix = defaultClonePrefix
	}
	ref := r.ref.(gitprovider.UserRepositoryRef)
	switch transport {
	case gitprovider.TransportTypeHTTPS:
		return gitprovider.ParseTypeHTTPS(fmt.Sprintf("%s/%s/%s/%s", gitprovider.GetDomainURL(ref.GetDomain()), prefix, addTilde(ref.UserLogin), ref.Slug()))
	case gitprovider.TransportTypeGit:
		return gitprovider.ParseTypeGit(ref.GetDomain(), addTilde(ref.UserLogin), ref.Slug())
	case gitprovider.TransportTypeSSH:
		return gitprovider.ParseTypeSSH(ref.GetDomain(), addTilde(ref.UserLogin), ref.Slug())
	default:
		return ""
	}
}

func newOrgRepository(ctx *clientContext, apiObj *Repository, ref gitprovider.RepositoryRef) *orgRepository {
	return &orgRepository{
		userRepository: *newUserRepository(ctx, apiObj, ref),
		teamAccess: &TeamAccessClient{
			clientContext: ctx,
			ref:           ref,
		},
		c: &OrgRepositoriesClient{
			clientContext: ctx,
		},
	}
}

var _ gitprovider.OrgRepository = &orgRepository{}

type orgRepository struct {
	userRepository
	teamAccess *TeamAccessClient
	c          *OrgRepositoriesClient
}

func (r *orgRepository) TeamAccess() gitprovider.TeamAccessClient {
	return r.teamAccess
}

// Reconcile makes sure the desired state in this object (called "req" here) becomes
// the actual state in the backing Git provider.
//
// If req doesn't exist under the hood, it is created (actionTaken == true).
// If req doesn't equal the actual state, the resource will be updated (actionTaken == true).
// If req is already the actual state, this is a no-op (actionTaken == false).
//
// The internal API object will be overridden with the received server data if actionTaken == true.
func (r *orgRepository) Reconcile(ctx context.Context) (bool, error) {
	_, actionTaken, err := r.c.Reconcile(ctx, r.ref.(gitprovider.OrgRepositoryRef), repositoryFromAPI(&r.repository))

	if err != nil {
		// Log the error and return it
		r.c.log.V(1).Error(err, "Error reconciling repository",
			"org", r.Repository().GetIdentity(),
			"repo", r.Repository().GetRepository(),
			"actionTaken", actionTaken)
		return actionTaken, err
	}

	return actionTaken, nil

}

// The internal API object will be overridden with the received server data.
func (r *orgRepository) Update(ctx context.Context) error {
	ref := r.ref.(gitprovider.OrgRepositoryRef)
	// update by calling client
	apiObj, err := update(ctx, r.c.client, ref.Key(), ref.Slug(), &r.repository, "")
	if err != nil {
		// Log the error and return it
		r.c.log.V(1).Error(err, "Error updating repository",
			"org", r.Repository().GetIdentity(),
			"repo", r.Repository().GetRepository())
		return err
	}

	r.repository = *apiObj

	return nil

}

// Delete deletes the current resource irreversibly.
// ErrNotFound is returned if the resource doesn't exist anymore.
func (r *orgRepository) Delete(ctx context.Context) error {
	ref := r.ref.(gitprovider.OrgRepositoryRef)
	return deleteRepository(ctx, r.c.client, ref.Key(), ref.Slug())
}

func repositoryFromAPI(apiObj *Repository) gitprovider.RepositoryInfo {
	repo := gitprovider.RepositoryInfo{
		Description:   &apiObj.Description,
		DefaultBranch: &apiObj.DefaultBranch,
	}
	repo.Visibility = gitprovider.RepositoryVisibilityVar(gitprovider.RepositoryVisibilityPrivate)
	if apiObj.Public {
		repo.Visibility = gitprovider.RepositoryVisibilityVar(gitprovider.RepositoryVisibilityPublic)
	}
	return repo
}

func repositoryToAPI(repo *gitprovider.RepositoryInfo, ref gitprovider.RepositoryRef) *Repository {
	apiObj := &Repository{
		Name:  *gitprovider.StringVar(ref.GetRepository()),
		ScmID: "git",
	}
	repositoryInfoToAPIObj(repo, apiObj)
	return apiObj
}

func repositoryInfoToAPIObj(repo *gitprovider.RepositoryInfo, apiObj *Repository) {
	if repo.Description != nil {
		apiObj.Description = *repo.Description
	}
	if repo.Visibility != nil {
		apiObj.Public = *gitprovider.StringVar(string(*repo.Visibility)) == "true"
	}

	if repo.DefaultBranch != nil {
		apiObj.DefaultBranch = *gitprovider.StringVar(*repo.DefaultBranch)
	}
}

// GetCloneURL returns a formatted string that can be used for cloning
// from a remote Git provider.
func (r *orgRepository) GetCloneURL(prefix string, transport gitprovider.TransportType) string {
	if prefix == "" {
		prefix = defaultClonePrefix
	}
	ref := r.ref.(gitprovider.OrgRepositoryRef)
	switch transport {
	case gitprovider.TransportTypeHTTPS:
		return gitprovider.ParseTypeHTTPS(fmt.Sprintf("%s/%s/%s/%s", gitprovider.GetDomainURL(ref.GetDomain()), prefix, ref.Key(), ref.Slug()))
	case gitprovider.TransportTypeGit:
		return gitprovider.ParseTypeGit(ref.GetDomain(), ref.Key(), ref.Slug())
	case gitprovider.TransportTypeSSH:
		return gitprovider.ParseTypeSSH(ref.GetDomain(), ref.Key(), ref.Slug())
	default:
		return ""
	}
}
