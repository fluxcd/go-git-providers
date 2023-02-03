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

var _ gitprovider.UserRepository = &userRepository{}

func newUserRepository(ctx *clientContext, apiObj git.GitRepository, ref gitprovider.RepositoryRef) *userRepository {
	return &userRepository{
		clientContext: ctx,
		r:             apiObj,
		ref:           ref,
		pullRequests: &PullRequestClient{
			clientContext: ctx,
			ref:           ref,
		},
	}
}

type userRepository struct {
	*clientContext
	pr  git.GitPullRequest
	r   git.GitRepository
	ref gitprovider.RepositoryRef

	pullRequests *PullRequestClient
}

func (r *userRepository) TeamAccess() gitprovider.TeamAccessClient {
	//TODO implement me
	panic("implement me")
}

func (r *userRepository) Get() gitprovider.RepositoryInfo {
	return repositoryFromAPI(&r.r)
}

func repositoryFromAPI(apiObj *git.GitRepository) gitprovider.RepositoryInfo {
	repo := gitprovider.RepositoryInfo{
		Description:   apiObj.Name,
		DefaultBranch: apiObj.DefaultBranch,
	}
	return repo
}
func (r userRepository) APIObject() interface{} {
	//TODO implement me
	panic("implement me")
}

func (r userRepository) Update(ctx context.Context) error {
	//TODO implement me
	panic("implement me")
}

func (r userRepository) Reconcile(ctx context.Context) (actionTaken bool, err error) {
	//TODO implement me
	panic("implement me")
}

func (r userRepository) Delete(ctx context.Context) error {
	//TODO implement me
	panic("implement me")
}

func (r userRepository) Repository() gitprovider.RepositoryRef {
	//TODO implement me
	panic("implement me")
}

func (r userRepository) Set(info gitprovider.RepositoryInfo) error {
	//TODO implement me
	panic("implement me")
}

func (r userRepository) DeployKeys() gitprovider.DeployKeyClient {
	//TODO implement me
	panic("implement me")
}

func (r userRepository) Commits() gitprovider.CommitClient {
	//TODO implement me
	panic("implement me")
}

func (r userRepository) Branches() gitprovider.BranchClient {
	//TODO implement me
	panic("implement me")
}

func (r userRepository) PullRequests() gitprovider.PullRequestClient {
	panic("implement me")
}

func (r userRepository) Files() gitprovider.FileClient {
	//TODO implement me
	panic("implement me")
}

func (r userRepository) Trees() gitprovider.TreeClient {
	//TODO implement me
	panic("implement me")
}
