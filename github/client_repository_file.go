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
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/fluxcd/go-git-providers/gitprovider"
	"github.com/google/go-github/v82/github"
)

// FileClient implements the gitprovider.FileClient interface.
var _ gitprovider.FileClient = &FileClient{}

// FileClient operates on the branch for a specific repository.
type FileClient struct {
	*clientContext
	ref gitprovider.RepositoryRef
}

// Get fetches and returns the contents of a file or multiple files in a directory from a given branch and path with possible options of FilesGetOption
// If a file path is given, the contents of the file are returned
// If a directory path is given, the contents of the files in the path's root are returned
func (c *FileClient) Get(ctx context.Context, path, branch string, optFns ...gitprovider.FilesGetOption) ([]*gitprovider.CommitFile, error) {
	opts := &github.RepositoryContentGetOptions{
		Ref: branch,
	}

	fileOpts := gitprovider.FilesGetOptions{}
	for _, opt := range optFns {
		opt.ApplyFilesGetOptions(&fileOpts)
	}

	_, directoryContent, _, err := c.c.Client().Repositories.GetContents(ctx, c.ref.GetIdentity(), c.ref.GetRepository(), path, opts)
	if err != nil {
		return nil, err
	}

	if len(directoryContent) == 0 {
		return nil, fmt.Errorf("no files found on this path[%s]", path)
	}

	files := make([]*gitprovider.CommitFile, 0)

	// For handling to close the output [io.ReadCloser] errors.
	var errs error
	for _, file := range directoryContent {
		filePath := file.Path
		output, _, err := c.c.Client().Repositories.DownloadContents(ctx, c.ref.GetIdentity(), c.ref.GetRepository(), *filePath, opts)
		if err != nil {
			return nil, err
		}
		content, err := io.ReadAll(output)
		if err != nil {
			return nil, err
		}
		// Don't use defer in the for loop. Checks errors lazily.
		errs = errors.Join(errs, output.Close())

		contentStr := string(content)
		files = append(files, &gitprovider.CommitFile{
			Path:    filePath,
			Content: &contentStr,
		})
	}
	if errs != nil {
		return nil, errs
	}

	return files, nil
}
