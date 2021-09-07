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

// Team implements the gitprovider.Team interface.
var _ gitprovider.Team = &Team{}

// Team represents a group in the Stash provider.
type Team struct {
	users []*gostash.User
	info  gitprovider.TeamInfo
	ref   gitprovider.OrganizationRef
}

// Get returns the team's information, Name and members.
func (t *Team) Get() gitprovider.TeamInfo {
	return t.info
}

// APIObject returns the Users that ware part of this team.
func (t *Team) APIObject() interface{} {
	return t.users
}

// Organization returns the organization that this team belongs to.
func (t *Team) Organization() gitprovider.OrganizationRef {
	return t.ref
}
