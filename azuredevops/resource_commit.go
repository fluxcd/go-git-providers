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

package azuredevops

import (
	"github.com/fluxcd/go-git-providers/gitprovider"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v6/git"
)

func newCommit(c *CommitClient, commit *git.GitPush) *commitType {
	return &commitType{
		k: *commit,
		c: c,
	}
}

var _ gitprovider.Commit = &commitType{}

type commitType struct {
	k git.GitPush
	c *CommitClient
}

func (c commitType) APIObject() interface{} {
	return &c.k
}

func (c commitType) Get() gitprovider.CommitInfo {
	return commitFromAPI(&c.k)
}
func commitFromAPI(apiObj *git.GitPush) gitprovider.CommitInfo {
	allCommits := gitprovider.CommitInfo{}
	for _, commit := range *apiObj.Commits {

		allCommits = gitprovider.CommitInfo{
			Sha:       *commit.CommitId,
			Author:    *commit.Author.Name,
			Message:   *commit.Comment,
			CreatedAt: commit.Author.Date.Time,
			URL:       *commit.Url,
		}
	}
	return allCommits
}
