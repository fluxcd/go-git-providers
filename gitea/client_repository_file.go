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
	"context"
	"fmt"

	"github.com/fluxcd/go-git-providers/gitprovider"
)

// FileClient implements the gitprovider.FileClient interface.
var _ gitprovider.FileClient = &FileClient{}

// FileClient operates on the branch for a specific repository.
type FileClient struct {
	*clientContext
	ref gitprovider.RepositoryRef
}

// Get fetches and returns the contents of a file or multiple files in a directory from a given branch and path.
// If a file path is given, the contents of the file are returned
// If a directory path is given, the contents of the files in the path's root are returned
func (c *FileClient) Get(ctx context.Context, path, branch string, optFns ...gitprovider.FilesGetOption) ([]*gitprovider.CommitFile, error) {
	fileOpts := gitprovider.FilesGetOptions{}
	for _, opt := range optFns {
		opt.ApplyFilesGetOptions(&fileOpts)
	}

	listFiles, _, err := c.c.ListContents(c.ref.GetIdentity(), c.ref.GetRepository(), branch, path)
	if err != nil {
		return nil, err
	}

	if len(listFiles) == 0 {
		return nil, fmt.Errorf("no files found on this path[%s]", path)
	}

	files := make([]*gitprovider.CommitFile, 0)
	for _, file := range listFiles {
		if file.Type != "file" {
			continue
		}
		filePath := file.Path
		fileBytes, _, err := c.c.GetFile(c.ref.GetIdentity(), c.ref.GetRepository(), branch, filePath)
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
