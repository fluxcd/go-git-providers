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
	"context"
	"testing"
	"time"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/google/go-cmp/cmp"
)

func TestNewCommit(t *testing.T) {
	path := "/test/path"
	content := "test content"
	key, err := openpgp.NewEntity("user1", "test key", "user1@users.com", nil)
	if err != nil {
		t.Fatalf("Generating a signing Key returned error: %v", err)
	}

	tests := []struct {
		name   string
		input  CreateCommit
		output string
	}{
		{
			name: "Create Valid Commit",
			input: CreateCommit{
				Author: &CommitAuthor{
					Name:  "user1",
					Email: "user1@users.com",
				},
				Committer: &CommitAuthor{
					Name:  "user2",
					Email: "user2@users.com",
				},
				Message: "test message",
				Parents: []string{"bc272406ac86a7c6ff3b8eab09a61f4e88764312"},
				URL:     "https://github.com/fluxcd/go-git-providers.git",
				Files: []CommitFile{
					{
						Path:    &path,
						Content: &content,
					},
				},
				SignKey: key,
			},
			output: "valid",
		},
		{
			name: "CreateCommit Invalid Commit",
			input: CreateCommit{
				Author: &CommitAuthor{
					Name:  "user1",
					Email: "user1@users.com",
				},
				Committer: &CommitAuthor{
					Name:  "user2",
					Email: "user2@users.com",
				},
				URL: "https://github.com/fluxcd/go-git-providers.git",
				Files: []CommitFile{
					{
						Path:    &path,
						Content: &content,
					},
				},
				SignKey: key,
			},
			output: "message is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := NewCommit(WithAuthor(tt.input.Author),
				WithCommitter(tt.input.Committer),
				WithURL(tt.input.URL),
				WithMessage(tt.input.Message),
				WithFiles(tt.input.Files),
				WithSignature(tt.input.SignKey))

			if err != nil {
				if err.Error() != tt.output {
					t.Fatalf("generating a Commit returned error: %v", err)
				}
				return
			}

			if diff := cmp.Diff(tt.input.Author, c.Author); diff != "" {
				t.Fatalf("Author mismatch (-want +got):\n%s", diff)
			}

			if diff := cmp.Diff(tt.input.Committer, c.Committer); diff != "" {
				t.Fatalf("Committer mismatch (-want +got):\n%s", diff)
			}

			if diff := cmp.Diff(tt.input.URL, c.URL); diff != "" {
				t.Fatalf("URL mismatch (-want +got):\n%s", diff)
			}

			if diff := cmp.Diff(tt.input.Message, c.Message); diff != "" {
				t.Fatalf("Message mismatch (-want +got):\n%s", diff)
			}

			if diff := cmp.Diff(tt.input.Files, c.Files); diff != "" {
				t.Fatalf("Files mismatch (-want +got):\n%s", diff)
			}

			if c.SignKey != tt.input.SignKey {
				t.Errorf("SignKey mismatch Got %v wanted %v", c, tt.input)
			}
		})

	}
}

func TestCreateCommit(t *testing.T) {
	path := "testpath"
	readmePath, readmeContent := "README.md", "# GO GIT REPO"
	licensePath := "LICENSE.md"
	licenseContent, err := getLicense("apache-2.0")
	if err != nil {
		t.Fatalf("Generating a license returned error: %v", err)
	}
	content := "test content"
	key, err := openpgp.NewEntity("user1", "test key", "user1@users.com", nil)
	if err != nil {
		t.Fatalf("generating a signing Key returned error: %v", err)
	}

	date := time.Now().Unix()

	initCommit := CreateCommit{
		Author: &CommitAuthor{
			Name:  "user1",
			Email: "user1@users.com",
			Date:  date,
		},
		Committer: &CommitAuthor{
			Name:  "user2",
			Email: "user2@users.com",
			Date:  date,
		},
		Message: "test message",
		URL:     "https://github.com/fluxcd/go-git-providers.git",
		Files: []CommitFile{
			{
				Path:    &readmePath,
				Content: &readmeContent,
			},
			{
				Path:    &licensePath,
				Content: &licenseContent,
			},
		},
		SignKey: key,
	}

	testCommit := CreateCommit{
		Author: &CommitAuthor{
			Name:  "user1",
			Email: "user1@users.com",
			Date:  date,
		},
		Committer: &CommitAuthor{
			Name:  "user2",
			Email: "user2@users.com",
			Date:  date,
		},
		Message: "test message",
		URL:     "https://github.com/fluxcd/go-git-providers.git",
		Files: []CommitFile{
			{
				Path:    &path,
				Content: &content,
			},
		},
		SignKey: key,
	}

	c, err := NewClient(nil, defaultHost, nil, initLogger(t))
	if err != nil {
		t.Fatalf("unexpected error while declaring a client: %v", err)
	}

	ctx := context.Background()
	//Init repo
	r, dir, err := c.Git.InitRepository(ctx, &initCommit, false)
	if err != nil {
		t.Fatalf("unexpected error while init repo: %v", err)
	}
	defer c.Git.Cleanup(dir)

	////Create a branch
	//err = c.Git.CreateBranch("testbranch", r, "")
	//if err != nil {
	//	t.Fatalf("unexpected error while creating a branch: %v", err)
	//}

	// create a commit
	obj, err := c.Git.CreateCommit(ctx, dir, r, "testbranch", &testCommit)

	if diff := cmp.Diff(testCommit.Author, obj.Author); diff != "" {
		t.Errorf("Author mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(testCommit.Committer, obj.Committer); diff != "" {
		t.Errorf("Committer mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(testCommit.Message, obj.Message); diff != "" {
		t.Errorf("Message mismatch (-want +got):\n%s", diff)
	}
}
