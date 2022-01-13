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
	"context"
	"encoding/base64"
	"io/ioutil"
	"strings"

	"github.com/fluxcd/go-git-providers/gitprovider"
	"github.com/xanzy/go-gitlab"
)

// FileClient implements the gitprovider.FileClient interface.
var _ gitprovider.FileClient = &FileClient{}

// FileClient operates on the branch for a specific repository.
type FileClient struct {
	*clientContext
	ref gitprovider.RepositoryRef
}

// Get fetches and returns the contents of a file from a given branch and path
func (c *FileClient) Get(_ context.Context, path, branch string) ([]*gitprovider.CommitFile, error) {

	opts := &gitlab.ListTreeOptions{
		Path: &path,
		Ref:  &branch,
	}

	listFiles, _, err := c.c.Client().Repositories.ListTree(getRepoPath(c.ref), opts)
	if err != nil {
		return nil, err
	}

	fileOpts := &gitlab.GetFileOptions{
		Ref: &branch,
	}

	files := make([]*gitprovider.CommitFile, 0)
	for _, file := range listFiles {
		fileDownloaded, _, err := c.c.Client().RepositoryFiles.GetFile(getRepoPath(c.ref), file.Path, fileOpts)
		if err != nil {
			return nil, err
		}
		filePath := fileDownloaded.FilePath
		fileContentDecoded := base64.NewDecoder(base64.RawStdEncoding, strings.NewReader(fileDownloaded.Content))
		fileBytes, err := ioutil.ReadAll(fileContentDecoded)
		if err != nil {
			return nil, err
		}
		fileStr := string(fileBytes)
		files = append(files, &gitprovider.CommitFile{
			Path:    &filePath,
			Content: &fileStr,
		})
	}

	return files, nil
}
