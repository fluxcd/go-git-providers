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
	"github.com/xanzy/go-gitlab"

	"github.com/fluxcd/go-git-providers/gitprovider"
	"github.com/fluxcd/go-git-providers/validation"
)

func newOrganization(ctx *clientContext, apiObj *gitlab.Group, ref gitprovider.OrganizationRef) *organization {
	return &organization{
		clientContext: ctx,
		g:             *apiObj,
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

	g   gitlab.Group
	ref gitprovider.OrganizationRef

	teams *TeamsClient
}

func (o *organization) Get() gitprovider.OrganizationInfo {
	return organizationFromAPI(&o.g)
}

func (o *organization) APIObject() interface{} {
	return &o.g
}

func (o *organization) Organization() gitprovider.OrganizationRef {
	return o.ref
}

func (o *organization) Teams() gitprovider.TeamsClient {
	return o.teams
}

func organizationFromAPI(apiObj *gitlab.Group) gitprovider.OrganizationInfo {
	return gitprovider.OrganizationInfo{
		Name:        &apiObj.Name,
		Description: &apiObj.Description,
	}
}

// validateOrganizationAPI validates the apiObj received from the server, to make sure that it is
// valid for our use.
func validateGroupAPI(apiObj *gitlab.Group) error {
	return validateAPIObject("GitLab.Group", func(validator validation.Validator) {
	})
}
