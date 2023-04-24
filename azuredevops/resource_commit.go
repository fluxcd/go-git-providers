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
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/git"
)

func newCommit(c *CommitClient, commitRef git.GitCommitRef) *commit {
	return &commit{
		ref: commitRef,
		c:   c,
	}
}

var _ gitprovider.Commit = &commit{}

type commit struct {
	ref git.GitCommitRef
	c   *CommitClient
}

func (c commit) APIObject() interface{} {
	return &c.ref
}

func (c commit) Get() gitprovider.CommitInfo {
	return commitFromAPI(&c.ref)
}
func commitFromAPI(apiObj *git.GitCommitRef) gitprovider.CommitInfo {
	return gitprovider.CommitInfo{
		Sha:     *apiObj.CommitId,
		Author:  *apiObj.Author.Name,
		Message: *apiObj.Comment,
		URL:     *apiObj.Url,
	}
}
