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
	"github.com/fluxcd/go-git-providers/gitprovider"
)

// The value of the "State" field of a Stash pull request after it has been merged"
const mergedState = "MERGED"

func newPullRequest(apiObj *PullRequest) *pullrequest {
	return &pullrequest{
		pr: *apiObj,
	}
}

var _ gitprovider.PullRequest = &pullrequest{}

type pullrequest struct {
	pr PullRequest
}

func (pr *pullrequest) Get() gitprovider.PullRequestInfo {
	return pullrequestFromAPI(&pr.pr)
}

func (pr *pullrequest) APIObject() interface{} {
	return &pr.pr
}

func pullrequestFromAPI(apiObj *PullRequest) gitprovider.PullRequestInfo {
	return gitprovider.PullRequestInfo{
		WebURL: getSelfref(apiObj.Self),
		Number: apiObj.ID,
		Merged: apiObj.State == mergedState,
	}
}

func getSelfref(selves []Self) string {
	if len(selves) == 0 {
		return "no http ref found"
	}
	return selves[0].Href
}
