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

func newCommit(c *CommitClient, commit *gitea.Commit) *commitType {
	return &commitType{
		k: *commit,
		c: c,
	}
}

var _ gitprovider.Commit = &commitType{}

type commitType struct {
	k gitea.Commit
	c *CommitClient
}

// Get returns the commit information.
func (c *commitType) Get() gitprovider.CommitInfo {
	return commitFromAPI(&c.k)
}

// APIObject returns the underlying API object.
func (c *commitType) APIObject() interface{} {
	return &c.k
}

func commitFromAPI(apiObj *gitea.Commit) gitprovider.CommitInfo {
	info := gitprovider.CommitInfo{
		Sha:       apiObj.SHA,
		Author:    apiObj.Author.UserName,
		CreatedAt: apiObj.Author.Created,
		URL:       apiObj.URL,
	}

	if apiObj.RepoCommit != nil {
		if apiObj.RepoCommit.Tree != nil {
			info.TreeSha = apiObj.RepoCommit.Tree.SHA
		}
		info.Message = apiObj.RepoCommit.Message
	}
	return info
}
