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
	"time"

	"github.com/fluxcd/go-git-providers/gitprovider"
)

func newCommit(commit *CommitObject) *commitType {
	return &commitType{
		k: *commit,
	}
}

var _ gitprovider.Commit = &commitType{}

type commitType struct {
	k CommitObject
}

func (c *commitType) Get() gitprovider.CommitInfo {
	return commitFromAPI(c.k)
}

func (c *commitType) APIObject() interface{} {
	return &c.k
}

func commitFromAPI(commit CommitObject) gitprovider.CommitInfo {
	t := time.Unix(commit.AuthorTimestamp, 0)
	return gitprovider.CommitInfo{
		Sha:       commit.ID,
		Author:    commit.Author.Name,
		Message:   commit.Message,
		CreatedAt: t,
	}
}
