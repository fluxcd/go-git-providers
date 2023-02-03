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
)

// PullRequestClient implements the gitprovider.PullRequestClient interface.
//var _ gitprovider.PullRequestClient = &PullRequestClient{}

// PullRequestClient operates on the pull requests for a specific repository.
type PullRequestClient struct {
	*clientContext
	ref gitprovider.RepositoryRef
}

func (c *PullRequestClient) Create(ctx context.Context, title, branch, baseBranch, description string) (gitprovider.PullRequest, error) {
	//TODO implement me
	panic("implement me")
}

func (c *PullRequestClient) Edit(ctx context.Context, number int, opts gitprovider.EditOptions) (gitprovider.PullRequest, error) {
	//TODO implement me
	panic("implement me")
}

func (c *PullRequestClient) Get(ctx context.Context, number int) (gitprovider.PullRequest, error) {
	//TODO implement me
	panic("implement me")
}

func (c *PullRequestClient) Merge(ctx context.Context, number int, mergeMethod gitprovider.MergeMethod, message string) error {
	//TODO implement me
	panic("implement me")
}
