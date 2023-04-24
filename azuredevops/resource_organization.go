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

package azuredevops

import (
	"github.com/fluxcd/go-git-providers/gitprovider"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/core"
)

func newProject(ctx *clientContext, apiObj core.TeamProject, ref gitprovider.OrganizationRef) *organization {
	return &organization{
		clientContext: ctx,
		p:             apiObj,
		ref:           ref,
	}
}

var _ gitprovider.Organization = &organization{}

type organization struct {
	*clientContext

	p   core.TeamProject
	ref gitprovider.OrganizationRef
}

func (o *organization) Teams() gitprovider.TeamsClient {
	//No implemented for Azure Devops
	return nil
}
func (o *organization) Get() gitprovider.OrganizationInfo {
	return organizationFromAPI(&o.p)
}

func (o *organization) APIObject() interface{} {
	return &o.p
}

func (o *organization) Organization() gitprovider.OrganizationRef {
	return o.ref
}

func organizationFromAPI(apiObj *core.TeamProject) gitprovider.OrganizationInfo {
	return gitprovider.OrganizationInfo{
		Name:        (*apiObj).Name,
		Description: (*apiObj).Description,
	}
}
