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
	"testing"
	"time"

	"code.gitea.io/sdk/gitea"
	"github.com/fluxcd/go-git-providers/gitprovider"
	"github.com/google/go-cmp/cmp"
)

func Test_CommitFromAPI(t *testing.T) {
	genTime, err := time.Parse(time.RFC3339, "2021-01-01T00:00:00Z")
	if err != nil {
		t.Errorf("failed to parse time: %v", err)
	}
	testCases := []struct {
		name   string
		apiObj *gitea.Commit
		want   gitprovider.CommitInfo
	}{
		{
			name: "full",
			apiObj: &gitea.Commit{
				CommitMeta: &gitea.CommitMeta{
					SHA: "sha",
					URL: "commitURL",
				},
				Author: &gitea.User{
					UserName: "username",
					Created:  genTime,
				},
				Committer: &gitea.User{
					UserName: "username",
					Created:  genTime,
				},
				RepoCommit: &gitea.RepoCommit{
					Message: "message",
					Tree: &gitea.CommitMeta{
						SHA: "treesha",
					},
				},
			},
			want: gitprovider.CommitInfo{
				Sha:       "sha",
				Author:    "username",
				CreatedAt: genTime,
				URL:       "commitURL",
				Message:   "message",
				TreeSha:   "treesha",
			},
		},
		{
			name: "nil repo commit",
			apiObj: &gitea.Commit{
				CommitMeta: &gitea.CommitMeta{
					SHA: "sha",
					URL: "commitURL",
				},
				Author: &gitea.User{
					UserName: "username",
					Created:  genTime,
				},
				Committer: &gitea.User{
					UserName: "username",
					Created:  genTime,
				},
			},
			want: gitprovider.CommitInfo{
				Sha:       "sha",
				Author:    "username",
				CreatedAt: genTime,
				URL:       "commitURL",
			},
		},
		{
			name: "nil tree",
			apiObj: &gitea.Commit{
				CommitMeta: &gitea.CommitMeta{
					SHA: "sha",
					URL: "commitURL",
				},
				Author: &gitea.User{
					UserName: "username",
					Created:  genTime,
				},
				Committer: &gitea.User{
					UserName: "username",
					Created:  genTime,
				},
				RepoCommit: &gitea.RepoCommit{
					Message: "message",
				},
			},
			want: gitprovider.CommitInfo{
				Sha:       "sha",
				Author:    "username",
				CreatedAt: genTime,
				URL:       "commitURL",
				Message:   "message",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := commitFromAPI(tc.apiObj)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("commitFromAPI() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
