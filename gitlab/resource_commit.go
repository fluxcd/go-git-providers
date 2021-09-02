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

package gitlab

import (
	"github.com/xanzy/go-gitlab"

	"github.com/fluxcd/go-git-providers/gitprovider"
)

func newCommit(c *CommitClient, commit *gitlab.Commit) *commitType {
	return &commitType{
		k: *commit,
		c: c,
	}
}

var _ gitprovider.Commit = &commitType{}

type commitType struct {
	k gitlab.Commit
	c *CommitClient
}

func (c *commitType) Get() gitprovider.CommitInfo {
	return commitFromAPI(&c.k)
}

func (c *commitType) APIObject() interface{} {
	return &c.k
}

func commitFromAPI(apiObj *gitlab.Commit) gitprovider.CommitInfo {
	return gitprovider.CommitInfo{
		Sha:       apiObj.ID,
		Author:    apiObj.AuthorName,
		Message:   apiObj.Message,
		CreatedAt: *apiObj.CreatedAt,
		WebURL:    apiObj.WebURL,
	}
}
