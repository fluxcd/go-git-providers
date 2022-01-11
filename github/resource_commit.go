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

package github

import (
	"github.com/google/go-github/v41/github"

	"github.com/fluxcd/go-git-providers/gitprovider"
)

func newCommit(c *CommitClient, commit *github.Commit) *commitType {
	return &commitType{
		k: *commit,
		c: c,
	}
}

var _ gitprovider.Commit = &commitType{}

type commitType struct {
	k github.Commit
	c *CommitClient
}

func (c *commitType) Get() gitprovider.CommitInfo {
	return commitFromAPI(&c.k)
}

func (c *commitType) APIObject() interface{} {
	return &c.k
}

func commitFromAPI(apiObj *github.Commit) gitprovider.CommitInfo {
	return gitprovider.CommitInfo{
		Sha:       *apiObj.SHA,
		TreeSha:   *apiObj.Tree.SHA,
		Author:    *apiObj.Author.Name,
		Message:   *apiObj.Message,
		CreatedAt: *apiObj.Author.Date,
		URL:       *apiObj.URL,
	}
}
