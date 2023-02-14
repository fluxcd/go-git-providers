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
	"github.com/fluxcd/go-git-providers/gitprovider"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v6/git"
)

var _ gitprovider.OrgRepository = &repository{}

func newRepository(ctx *clientContext, apiObj git.GitRepository, ref gitprovider.RepositoryRef) *repository {
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
	pr  git.GitPullRequest
	r   git.GitRepository
	ref gitprovider.RepositoryRef

	pullRequests *PullRequestClient
	trees        *TreeClient
	branches     *BranchClient
	commits      *CommitClient
}

func (r *repository) TeamAccess() gitprovider.TeamAccessClient {
	//TODO implement me
	panic("implement me")
}

func (r *repository) Get() gitprovider.RepositoryInfo {
	return repositoryFromAPI(&r.r)
}

func repositoryFromAPI(apiObj *git.GitRepository) gitprovider.RepositoryInfo {
	repo := gitprovider.RepositoryInfo{
		Description:   apiObj.Name,
		DefaultBranch: apiObj.DefaultBranch,
	}
	return repo
}
func (r *repository) Trees() gitprovider.TreeClient {
	return r.trees
}

func (r *repository) APIObject() interface{} {
	//TODO implement me
	panic("implement me")
}

func (r *repository) Update(ctx context.Context) error {
	//TODO implement me
	panic("implement me")
}

func (r *repository) Reconcile(ctx context.Context) (actionTaken bool, err error) {
	//TODO implement me
	panic("implement me")
}

func (r *repository) Delete(ctx context.Context) error {
	//TODO implement me
	panic("implement me")
}

func (r *repository) Repository() gitprovider.RepositoryRef {
	//TODO implement me
	panic("implement me")
}

func (r *repository) Set(info gitprovider.RepositoryInfo) error {
	//TODO implement me
	panic("implement me")
}

func (r *repository) DeployKeys() gitprovider.DeployKeyClient {
	//TODO implement me
	panic("implement me")
}

func (r *repository) Commits() gitprovider.CommitClient {
	return r.commits
}

func (r *repository) Branches() gitprovider.BranchClient {
	return r.branches
}

func (r *repository) PullRequests() gitprovider.PullRequestClient {
	panic("implement me")
}

func (r *repository) Files() gitprovider.FileClient {
	//TODO implement me
	panic("implement me")
}
