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
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/git"
)

var _ gitprovider.PullRequest = &pullrequest{}

type pullrequest struct {
	*clientContext

	pr git.GitPullRequest
}

func (pr pullrequest) APIObject() interface{} {
	return &pr.pr
}

func (pr pullrequest) Get() gitprovider.PullRequestInfo {
	return pullrequestFromAPI(&pr.pr)
}

func newPullRequest(ctx *clientContext, apiObj *git.GitPullRequest) *pullrequest {
	return &pullrequest{
		clientContext: ctx,
		pr:            *apiObj,
	}
}
func pullrequestFromAPI(apiObj *git.GitPullRequest) gitprovider.PullRequestInfo {
	var sourceBranch string
	head := apiObj.SourceRefName
	if head != nil {
		if head != nil {
			sourceBranch = *head
		}
	}
	status := false
	if apiObj.MergeStatus != nil && apiObj.MergeStatus == &git.PullRequestAsyncStatusValues.Succeeded {
		status = true
	}

	return gitprovider.PullRequestInfo{
		Title:        *apiObj.Title,
		Description:  *apiObj.Description,
		Merged:       status,
		Number:       *apiObj.PullRequestId,
		WebURL:       *apiObj.Url,
		SourceBranch: sourceBranch,
	}
}
