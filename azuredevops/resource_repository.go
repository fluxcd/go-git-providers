/*
Copyright 2021 The Flux CD contributors.

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

	"github.com/fluxcd/go-git-providers/gitprovider"
	"github.com/google/go-cmp/cmp"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/core"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/git"
)

var _ gitprovider.OrgRepository = &repository{}

func newRepository(ctx *clientContext, apiObj git.GitRepository, ref gitprovider.OrgRepositoryRef) *repository {
	return &repository{
		clientContext: ctx,
		r:             apiObj,
		ref:           ref,
		pullRequests: &PullRequestClient{
			clientContext: ctx,
			ref:           ref,
		},
		trees: &TreeClient{
			clientContext: ctx,
			ref:           ref,
		},
		branches: &BranchClient{
			clientContext: ctx,
			ref:           ref,
		},
		commits: &CommitClient{
			clientContext: ctx,
			ref:           ref,
		},
	}
}

type repository struct {
	*clientContext
	pr        git.GitPullRequest
	r         git.GitRepository
	topUpdate *git.GitRepository
	ref       gitprovider.OrgRepositoryRef

	pullRequests *PullRequestClient
	trees        *TreeClient
	branches     *BranchClient
	commits      *CommitClient
}

func (r *repository) TeamAccess() gitprovider.TeamAccessClient {
	//No implemented for Azure Devops
	return nil
}

func (r *repository) Get() gitprovider.RepositoryInfo {
	return repositoryFromAPI(&r.r)
}

func repositoryFromAPI(apiObj *git.GitRepository) gitprovider.RepositoryInfo {
	repo := gitprovider.RepositoryInfo{
		Name: apiObj.Name,
	}
	if apiObj.Project.Description != nil {
		repo.Description = apiObj.Project.Description
	}
	if apiObj.DefaultBranch != nil {
		repo.DefaultBranch = apiObj.DefaultBranch
	}
	repo.Visibility = gitprovider.RepositoryVisibilityVar(gitprovider.RepositoryVisibilityPrivate)
	if *apiObj.Project.Visibility == "public" {
		repo.Visibility = gitprovider.RepositoryVisibilityVar(gitprovider.RepositoryVisibilityPublic)
	}
	return repo
}
func (r *repository) Trees() gitprovider.TreeClient {
	return r.trees
}

func (r *repository) APIObject() interface{} {
	return &r.r
}

// The internal API object will be overridden with the received server data.
func (r *repository) Update(ctx context.Context) error {

	apiObj, err := r.g.UpdateRepository(ctx, git.UpdateRepositoryArgs{
		Project:      r.r.Project.Name,
		RepositoryId: r.r.Id,
		NewRepositoryInfo: &git.GitRepository{
			Name: r.r.Name,
		},
	})

	if err != nil {
		return err
	}
	r.r = *apiObj
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

func (r *repository) Reconcile(ctx context.Context) (bool, error) {

	projectName := r.ref.GetIdentity()
	repositoryID := r.ref.GetRepository()
	apiObj, err := r.g.GetRepository(ctx, git.GetRepositoryArgs{
		RepositoryId: &repositoryID,
		Project:      &projectName,
	})
	if err != nil {
		// Create if not found
		if errors.Is(handleHTTPError(err), gitprovider.ErrNotFound) {
			CreateRepositoryArgs := git.CreateRepositoryArgs{
				GitRepositoryToCreate: &git.GitRepositoryCreateOptions{
					Name: &repositoryID,
				},
				Project: &r.ref.Organization,
			}

			// Create a git repository in a team project.
			repo, err := r.g.CreateRepository(ctx, CreateRepositoryArgs)

			if err != nil {
				return true, err
			}
			r.r = *repo
			return true, nil
		}

		return false, err
	}

	// Use wrappers here to extract the "spec" part of the object for comparison
	desiredSpec := newAzureDevopsRepoSpec(&r.r)
	actualSpec := newAzureDevopsRepoSpec(apiObj)

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
func (r *repository) Delete(ctx context.Context) error {

	return r.g.DeleteRepository(ctx, git.DeleteRepositoryArgs{
		RepositoryId: r.r.Id,
		Project:      r.r.Project.Name,
	})
}

func (r *repository) Repository() gitprovider.RepositoryRef {
	return r.ref
}

// Set sets the desired state of this object.
// User have to call Update() to apply the changes to the server.
// The changes will then be reflected in the internal API object.
func (r *repository) Set(info gitprovider.RepositoryInfo) error {
	if err := info.ValidateInfo(); err != nil {
		return err
	}
	repositoryInfoToAPIObj(&info, &r.r)
	return nil
}

func (r *repository) DeployTokens() (gitprovider.DeployTokenClient, error) {
	///No implemented for Azure Devops
	return nil, nil
}

func (r *repository) DeployKeys() gitprovider.DeployKeyClient {
	///No implemented for Azure Devops
	return nil
}

func (r *repository) Commits() gitprovider.CommitClient {
	return r.commits
}

func (r *repository) Branches() gitprovider.BranchClient {
	return r.branches
}

func (r *repository) PullRequests() gitprovider.PullRequestClient {
	return r.pullRequests
}

func (r *repository) Files() gitprovider.FileClient {
	//No implemented for Azure Devops
	return nil
}
func repositoryInfoToAPIObj(repo *gitprovider.RepositoryInfo, apiObj *git.GitRepository) {
	if repo.Visibility != nil {
		*apiObj.Project.Visibility = core.ProjectVisibility(*gitprovider.StringVar(string(*repo.Visibility)))
	}
	if repo.Name != nil {
		apiObj.Name = repo.Name
	}

}

// This function copies over the fields that are part of create/update requests of a project
// i.e. the desired spec of the repository. This allows us to separate "spec" from "status" fields.
func newAzureDevopsRepoSpec(repo *git.GitRepository) *azureDevopsRepoSpec {
	return &azureDevopsRepoSpec{
		&git.GitRepository{
			Name: repo.Name,
		},
	}
}

type azureDevopsRepoSpec struct {
	*git.GitRepository
}

func (s *azureDevopsRepoSpec) Equals(other *azureDevopsRepoSpec) bool {
	return cmp.Equal(s, other)
}
