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

package gitea

import (
	"code.gitea.io/sdk/gitea"

	"github.com/fluxcd/go-git-providers/gitprovider"
	"github.com/fluxcd/go-git-providers/validation"
)

func newOrganization(ctx *clientContext, apiObj *gitea.Organization, ref gitprovider.OrganizationRef) *organization {
	return &organization{
		clientContext: ctx,
		o:             *apiObj,
		ref:           ref,
		teams: &TeamsClient{
			clientContext: ctx,
			ref:           ref,
		},
	}
}

var _ gitprovider.Organization = &organization{}

type organization struct {
	*clientContext

	o   gitea.Organization
	ref gitprovider.OrganizationRef

	teams *TeamsClient
}

// Get returns the organization information.
func (o *organization) Get() gitprovider.OrganizationInfo {
	return organizationFromAPI(&o.o)
}

// APIObject returns the underlying API object.
func (o *organization) APIObject() interface{} {
	return &o.o
}


// Organization returns the organization reference.
func (o *organization) Organization() gitprovider.OrganizationRef {
	return o.ref
}

// Teams returns the teams client.
func (o *organization) Teams() gitprovider.TeamsClient {
	return o.teams
}

func organizationFromAPI(apiObj *gitea.Organization) gitprovider.OrganizationInfo {
	return gitprovider.OrganizationInfo{
		Name:        &apiObj.UserName,
		Description: &apiObj.Description,
	}
}

// validateOrganizationAPI validates the apiObj received from the server, to make sure that it is
// valid for our use.
func validateOrganizationAPI(apiObj *gitea.Organization) error {
	return validateAPIObject("Gitea.Organization", func(validator validation.Validator) {})
}
