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

package stash

import (
	"github.com/fluxcd/go-git-providers/gitprovider"
	gostash "github.com/fluxcd/go-git-providers/go-stash"
)

// Organization implements the gitprovider.Organization interface.
var _ gitprovider.Organization = &Organization{}

// Organization represents a project in the Stash provider.
type Organization struct {
	*clientContext

	p   gostash.Project
	ref gitprovider.OrganizationRef

	teams *TeamsClient
}

// Get returns the organization's information, Name and description.
func (o *Organization) Get() gitprovider.OrganizationInfo {
	return organizationFromAPI(&o.p)
}

// APIObject returns the underlying value that was returned from the server.
func (o *Organization) APIObject() interface{} {
	return &o.p
}

// Organization returns the organization reference.
func (o *Organization) Organization() gitprovider.OrganizationRef {
	return o.ref
}

//Teams gives access to the TeamsClient for this specific organization
func (o *Organization) Teams() gitprovider.TeamsClient {
	return o.teams
}

func organizationFromAPI(apiObj *gostash.Project) gitprovider.OrganizationInfo {
	return gitprovider.OrganizationInfo{
		Name:        &apiObj.Name,
		Description: &apiObj.Description,
	}
}

func newOrganization(ctx *clientContext, apiObj *gostash.Project, ref gitprovider.OrganizationRef) *Organization {
	return &Organization{
		clientContext: ctx,
		p:             *apiObj,
		ref:           ref,
		teams: &TeamsClient{
			clientContext: ctx,
			ref:           ref,
		},
	}
}
