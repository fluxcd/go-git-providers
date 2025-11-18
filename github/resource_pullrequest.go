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

package github

import (
	"github.com/fluxcd/go-git-providers/gitprovider"
	"github.com/google/go-github/v75/github"
)

func newPullRequest(ctx *clientContext, apiObj *github.PullRequest) *pullrequest {
	return &pullrequest{
		clientContext: ctx,
		pr:            *apiObj,
	}
}

var _ gitprovider.PullRequest = &pullrequest{}

type pullrequest struct {
	*clientContext

	pr github.PullRequest
}

func (pr *pullrequest) Get() gitprovider.PullRequestInfo {
	return pullrequestFromAPI(&pr.pr)
}

func (pr *pullrequest) APIObject() interface{} {
	return &pr.pr
}

func pullrequestFromAPI(apiObj *github.PullRequest) gitprovider.PullRequestInfo {
	var sourceBranch string
	head := apiObj.Head
	if head != nil {
		if head.Ref != nil {
			sourceBranch = *head.Ref
		}
	}
	return gitprovider.PullRequestInfo{
		Title:        apiObj.GetTitle(),
		Description:  apiObj.GetBody(),
		Merged:       apiObj.GetMerged(),
		Number:       apiObj.GetNumber(),
		WebURL:       apiObj.GetHTMLURL(),
		SourceBranch: sourceBranch,
	}
}
