/*
Copyright 2023 The Flux CD contributors.

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

package gitea

import (
	"code.gitea.io/sdk/gitea"
	"github.com/fluxcd/go-git-providers/gitprovider"
)

func newPullRequest(ctx *clientContext, apiObj *gitea.PullRequest) *pullrequest {
	return &pullrequest{
		clientContext: ctx,
		pr:            *apiObj,
	}
}

var _ gitprovider.PullRequest = &pullrequest{}

type pullrequest struct {
	*clientContext

	pr gitea.PullRequest
}

// Get returns the pull request information.
func (pr *pullrequest) Get() gitprovider.PullRequestInfo {
	return pullrequestFromAPI(&pr.pr)
}

// APIObject returns the underlying API object.
func (pr *pullrequest) APIObject() interface{} {
	return &pr.pr
}

func pullrequestFromAPI(apiObj *gitea.PullRequest) gitprovider.PullRequestInfo {
	return gitprovider.PullRequestInfo{
		Merged: apiObj.HasMerged,
		Number: int(apiObj.Index),
		WebURL: apiObj.HTMLURL,
	}
}
