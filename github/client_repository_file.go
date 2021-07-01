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
	"github.com/fluxcd/go-git-providers/gitprovider"
	"github.com/google/go-github/v32/github"
	"io/ioutil"
)

// FileClient implements the gitprovider.FileClient interface.
var _ gitprovider.FileClient = &FileClient{}

// FileClient operates on the branch for a specific repository.
type FileClient struct {
	*clientContext
	ref gitprovider.RepositoryRef
}

func (c *FileClient) Get(ctx context.Context, path, branch string) ([]*gitprovider.File, error) {

	opts := &github.RepositoryContentGetOptions{
		Ref: branch,
	}

	_, directoryContent, _, err := c.c.Client().Repositories.GetContents(ctx, c.ref.GetIdentity(), c.ref.GetRepository(), path, opts)
	if err != nil {
		return nil, err
	}

	if len(directoryContent) == 0 {
		return nil, fmt.Errorf("no files found on this path[%s]", path)
	}

	files := make([]*gitprovider.File, 0)

	for _, file := range directoryContent {
		filePath := file.Path
		name := file.Name
		output, err := c.c.Client().Repositories.DownloadContents(ctx, c.ref.GetIdentity(), c.ref.GetRepository(), *filePath, opts)
		if err != nil {
			return nil, err
		}
		content, err := ioutil.ReadAll(output)
		if err != nil {
			return nil, err
		}
		contentStr := string(content)
		files = append(files, &gitprovider.File{
			Path:    filePath,
			Name:    name,
			Content: &contentStr,
		})
	}

	return files, nil
}
