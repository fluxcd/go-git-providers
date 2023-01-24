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
	"github.com/fluxcd/go-git-providers/gitprovider"
	"github.com/microsoft/azure-devops-go-api/azuredevops/git"
)

var _ gitprovider.PullRequest = &pullrequest{}

type pullrequest struct {
	*clientContext

	pr git.GitPullRequest
}

func (p pullrequest) APIObject() interface{} {
	//TODO implement me
	panic("implement me")
}

func (p pullrequest) Get() gitprovider.PullRequestInfo {
	//TODO implement me
	panic("implement me")
}

func newPullRequest(ctx *clientContext, apiObj git.GitPullRequest) *pullrequest {
	return &pullrequest{
		clientContext: ctx,
		pr:            apiObj,
	}
}
