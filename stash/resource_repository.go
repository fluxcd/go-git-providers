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

	"github.com/google/go-cmp/cmp"

	"github.com/fluxcd/go-git-providers/gitprovider"
	"github.com/fluxcd/go-git-providers/validation"
)

func newUserRepository(ctx *clientContext, apiObj *Repository, ref gitprovider.RepositoryRef) *userRepository {
	return &userRepository{
		clientContext: ctx,
		p:             *apiObj,
		ref:           ref,
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
	*clientContext

	p   Repository
	ref gitprovider.RepositoryRef

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
	return repositoryFromAPI(&r.p)
}

func (r *userRepository) Set(info gitprovider.RepositoryInfo) error {
	if err := info.ValidateInfo(); err != nil {
		return err
	}
	repositoryInfoToAPIObj(&info, &r.p)
	return nil
}

func (r *userRepository) APIObject() interface{} {
	return &r.p
}

func (r *userRepository) Repository() gitprovider.RepositoryRef {
	return r.ref
}

func (r *userRepository) DeployKeys() gitprovider.DeployKeyClient {
	return r.deployKeys
}

func orgUserName(project *Project) string {
	if len(project.User.Name) > 0 {
		return project.User.Name
	}
	return project.Name
}

// The internal API object will be overridden with the received server data.
func (r *userRepository) Update(ctx context.Context) error {
	apiObj, err := r.c.UpdateRepository(ctx, orgUserName(&r.p.Project), &r.p)
	if err != nil {
		return err
	}
	r.p = *apiObj
	return nil
}

// Reconcile makes sure the desired state in this object (called "req" here) becomes
// the actual state in the backing Git provider.
//
// If req doesn't exist under the hood, it is created (actionTaken == true).
// If req doesn't equal the actual state, the resource will be updated (actionTaken == true).
// If req is already the actual state, this is a no-op (actionTaken == false).
//
// The internal API object will be overridden with the received server data if actionTaken == true.
func (r *userRepository) Reconcile(ctx context.Context) (bool, error) {
	apiObj, err := r.c.GetRepository(ctx, r.ref.GetIdentity(), r.ref.GetRepository())
	if err != nil {
		// Create if not found
		if errors.Is(err, gitprovider.ErrNotFound) {
			// orgName := ""
			// if orgRef, ok := r.ref.(gitprovider.OrgRepositoryRef); ok {
			// 	orgName = orgRef.Organization
			// }
			Repository, err := r.c.CreateRepository(ctx, r.p.Project.User.Name, &r.p)
			if err != nil {
				return true, err
			}
			r.p = *Repository
			return true, nil
		}

		return false, err
	}

	// Use wrappers here to extract the "spec" part of the object for comparison
	desiredSpec := newStashRepositorySpec(&r.p)
	actualSpec := newStashRepositorySpec(apiObj)

	// If desired state already is the actual state, do nothing
	if desiredSpec.Equals(actualSpec) {
		return false, nil
	}
	// Otherwise, make the desired state the actual state
	return true, r.Update(ctx)
}

// Delete deletes the current resource irreversibly.
//
// ErrNotFound is returned if the resource doesn't exist anymore.
func (r *userRepository) Delete(ctx context.Context) error {
	return r.c.DeleteRepository(ctx, r.ref.GetIdentity(), r.ref.GetRepository())
}

func newProjectRepository(ctx *clientContext, apiObj *Repository, ref gitprovider.RepositoryRef) *orgRepository {
	return &orgRepository{
		userRepository: *newUserRepository(ctx, apiObj, ref),
		teamAccess: &TeamAccessClient{
			clientContext: ctx,
			ref:           ref,
		},
	}
}

var _ gitprovider.OrgRepository = &orgRepository{}

type orgRepository struct {
	userRepository

	teamAccess  *TeamAccessClient
	branch      *BranchClient
	commit      *CommitClient
	pullRequest *PullRequestClient
}

func (r *orgRepository) Branches() gitprovider.BranchClient {
	return r.branch
}

func (r *orgRepository) Commits() gitprovider.CommitClient {
	return r.commit
}

func (r *orgRepository) PullRequests() gitprovider.PullRequestClient {
	return r.pullRequest
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
	apiObj, err := r.c.GetRepository(ctx, r.ref.GetIdentity(), r.ref.GetRepository())
	if err != nil {
		// Create if not found
		if errors.Is(err, gitprovider.ErrNotFound) {
			Repository, err := r.c.CreateRepository(ctx, r.p.Name, &r.p)
			if err != nil {
				return true, err
			}
			r.p = *Repository
			return true, nil
		}

		return false, err
	}

	// Use wrappers here to extract the "spec" part of the object for comparison
	desiredSpec := newStashRepositorySpec(&r.p)
	actualSpec := newStashRepositorySpec(apiObj)

	// If desired state already is the actual state, do nothing
	if desiredSpec.Equals(actualSpec) {
		return false, nil
	}
	// Otherwise, make the desired state the actual state
	return true, r.Update(ctx)
}

func repositoryFromAPI(apiObj *Repository) gitprovider.RepositoryInfo {
	repo := gitprovider.RepositoryInfo{
		Description:   &apiObj.Description,
		DefaultBranch: &masterBranchName, // ToDo get default branch
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
		ScmId: "git",
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
}

// This function copies over the fields that are part of create/update requests of a Repository
// i.e. the desired spec of the repository. This allows us to separate "spec" from "status" fields.
func newStashRepositorySpec(repository *Repository) *stashRepositorySpec {
	return &stashRepositorySpec{
		&Repository{
			// Generic
			Name:        repository.Name,
			Description: repository.Description,
			Project:     repository.Project,
		},
	}
}

type stashRepositorySpec struct {
	*Repository
}

func (s *stashRepositorySpec) Equals(other *stashRepositorySpec) bool {
	return cmp.Equal(s, other)
}

// validateRepositoryAPI validates the apiObj received from the server, to make sure that it is
// valid for our use.
func validateRepositoryAPI(apiObj *Repository) error {
	return validateAPIObject("Stash.Repository", func(validator validation.Validator) {
		// Make sure name is set
		if apiObj.Name == "" {
			validator.Required("Name")
		}
		// Make sure slug is set
		if apiObj.Slug == "" {
			validator.Required("Slug")
		}
	})
}
