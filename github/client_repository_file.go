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
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/fluxcd/go-git-providers/gitprovider"
	"github.com/google/go-github/v45/github"
)

// FileClient implements the gitprovider.FileClient interface.
var _ gitprovider.FileClient = &FileClient{}

// FileClient operates on the branch for a specific repository.
type FileClient struct {
	*clientContext
	ref gitprovider.RepositoryRef
}

// if number of files retrieved exceed the max number of files, return slice only until the number of maxFiles with bool true for limiting succeeded
func limitSliceSize(s []*gitprovider.CommitFile, size int) ([]*gitprovider.CommitFile, bool) {
	if size != 0 && len(s) > size {
		return s[:size], true
	}
	return s, false
}

// Get fetches and returns the contents of a file or directory from a given branch and path with possible options of FilesGetOption
// If recursive option is provided, the files are retrieved recursively from subdirectories of the base path.
// If recursive and MaxDepth options are provided, the files are retrieved recursively from subdirectories until reaching the max depth of levels
// Recursive and MaxDepth should be provided together, default MaxDepth : 0
// If maxFiles option is provided, the number of files are maximized to the number provided

// Uses https://docs.github.com/en/rest/repos/contents#get-repository-content
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

	for _, file := range directoryContent {
		filePath := file.Path
		if *file.Type == "dir" {
			if fileOpts.Recursive {
				// stop recursive calls when level is the max level reached
				if fileOpts.MaxDepth <= 1 {
					continue
				}

				if !strings.HasSuffix(path, "/") {
					path = path + "/"
				}
				subdirectoryPath := fmt.Sprintf("%s%s/", path, *file.Name)

				// recursive call for child directories to get their content
				childMaxFiles := fileOpts.MaxFiles - len(files)
				childOptFns := gitprovider.FilesGetOptions{Recursive: fileOpts.Recursive, MaxFiles: childMaxFiles, MaxDepth: fileOpts.MaxDepth - 1}
				childFiles, err := c.Get(ctx, subdirectoryPath, branch, &childOptFns)

				if err != nil {
					return nil, err
				}
				files = append(files, childFiles...)
				files, limitSuccess := limitSliceSize(files, fileOpts.MaxFiles)
				if limitSuccess {
					return files, nil
				}
			}
			continue

		}
		output, _, err := c.c.Client().Repositories.DownloadContents(ctx, c.ref.GetIdentity(), c.ref.GetRepository(), *filePath, opts)
		if err != nil {
			return nil, err
		}
		content, err := ioutil.ReadAll(output)
		if err != nil {
			return nil, err
		}
		err = output.Close()
		if err != nil {
			return nil, err
		}
		contentStr := string(content)
		files = append(files, &gitprovider.CommitFile{
			Path:    filePath,
			Content: &contentStr,
		})
		files, limitSuccess := limitSliceSize(files, fileOpts.MaxFiles)
		if limitSuccess {
			return files, nil
		}

	}

	return files, nil
}
