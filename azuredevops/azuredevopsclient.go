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

package azuredevops

import (
	"context"
	"github.com/microsoft/azure-devops-go-api/azuredevops/core"
)

type azureDevopsClient interface {
	Client() core.Client
	ListProjects(ctx context.Context) (*core.GetProjectsResponseValue, error)
}

// azureDevopsClientImpl is a wrapper around *azureDevops.Client, which implements higher-level methods,
// Pagination is implemented for all List* methods, all returned
// objects are validated, and HTTP errors are handled/wrapped using handleHTTPError.
type azureDevopsClientImpl struct {
	c                  core.Client
	destructiveActions bool
}

var _ azureDevopsClient = &azureDevopsClientImpl{}

func (c *azureDevopsClientImpl) Client() core.Client {
	return c.c
}
func (c *azureDevopsClientImpl) ListProjects(ctx context.Context) (*core.GetProjectsResponseValue, error) {
	apiObj, err := c.c.GetProjects(ctx, core.GetProjectsArgs{})
	if err != nil {
		return nil, err
	}
	return apiObj, nil
}
