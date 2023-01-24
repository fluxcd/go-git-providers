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
	"github.com/fluxcd/go-git-providers/gitprovider"
)

// OrgRepositoriesClient implements the gitprovider.OrgRepositoriesClient interface.
var _ gitprovider.OrgRepositoriesClient = &OrgRepositoriesClient{}

// OrgRepositoriesClient operates on repositories the user has access to.
type OrgRepositoriesClient struct {
	*clientContext
}

func (o OrgRepositoriesClient) Get(ctx context.Context, r gitprovider.OrgRepositoryRef) (gitprovider.OrgRepository, error) {
	//TODO implement me
	panic("implement me")
}

func (o OrgRepositoriesClient) List(ctx context.Context, orgRef gitprovider.OrganizationRef) ([]gitprovider.OrgRepository, error) {
	//TODO implement me
	panic("implement me")
}

func (o OrgRepositoriesClient) Create(ctx context.Context, r gitprovider.OrgRepositoryRef, req gitprovider.RepositoryInfo, opts ...gitprovider.RepositoryCreateOption) (gitprovider.OrgRepository, error) {
	//TODO implement me
	panic("implement me")
}

func (o OrgRepositoriesClient) Reconcile(ctx context.Context, r gitprovider.OrgRepositoryRef, req gitprovider.RepositoryInfo, opts ...gitprovider.RepositoryReconcileOption) (resp gitprovider.OrgRepository, actionTaken bool, err error) {
	//TODO implement me
	panic("implement me")
}
