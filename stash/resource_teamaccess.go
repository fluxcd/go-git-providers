/*
Copyright 2021 The Flux authors

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
	"context"

	"github.com/fluxcd/go-git-providers/gitprovider"
)

const (
	stashPermissionRead  = "REPO_READ"
	stashPermissionWrite = "REPO_WRITE"
	stashPermissionAdmin = "REPO_ADMIN"
)

var (
	permissionPriority = map[int]gitprovider.RepositoryPermission{
		10: gitprovider.RepositoryPermissionPull,
		20: gitprovider.RepositoryPermissionTriage,
		30: gitprovider.RepositoryPermissionPush,
		40: gitprovider.RepositoryPermissionMaintain,
		50: gitprovider.RepositoryPermissionAdmin,
	}

	stashPriority = map[string]int{
		stashPermissionRead:  10,
		stashPermissionWrite: 30,
		stashPermissionAdmin: 50,
	}
)

func newTeamAccess(c *TeamAccessClient, ta gitprovider.TeamAccessInfo) *teamAccess {
	return &teamAccess{
		ta: ta,
		c:  c,
	}
}

var _ gitprovider.TeamAccess = &teamAccess{}

type teamAccess struct {
	ta gitprovider.TeamAccessInfo
	c  *TeamAccessClient
}

func (ta *teamAccess) Get() gitprovider.TeamAccessInfo {
	return ta.ta
}

func (ta *teamAccess) Set(info gitprovider.TeamAccessInfo) error {
	if err := info.ValidateInfo(); err != nil {
		return err
	}
	ta.ta = info
	return nil
}

func (ta *teamAccess) APIObject() interface{} {
	return nil
}

func (ta *teamAccess) Repository() gitprovider.RepositoryRef {
	return ta.c.ref
}

func (ta *teamAccess) Delete(ctx context.Context) error {
	return gitprovider.ErrNoProviderSupport
}

func (ta *teamAccess) Update(ctx context.Context) error {
	// Update the actual state to be the desired state
	// by issuing a Create, which uses a PUT underneath.
	resp, err := ta.c.Create(ctx, ta.Get())
	if err != nil {
		// Log the error and return it
		ta.c.log.V(1).Error(err, "Error updating team access",
			"org", ta.Repository().GetIdentity(),
			"repo", ta.Repository().GetRepository())
		return err
	}
	return ta.Set(resp.Get())
}

// Reconcile makes sure the given desired state (req) becomes the actual state in the backing Git provider.
//
// If req doesn't exist under the hood, it is created (actionTaken == true).
// If req doesn't equal the actual state, the resource will be deleted and recreated (actionTaken == true).
// If req is already the actual state, this is a no-op (actionTaken == false).
func (ta *teamAccess) Reconcile(ctx context.Context) (bool, error) {
	_, actionTaken, err := ta.c.Reconcile(ctx, ta.ta)

	if err != nil {
		// Log the error and return it
		ta.c.log.V(1).Error(err, "Error reconciling team access",
			"org", ta.Repository().GetIdentity(),
			"repo", ta.Repository().GetRepository(),
			"actionTaken", actionTaken)
		return actionTaken, err
	}

	return actionTaken, nil
}

func getGitProviderPermission(permissionLevel int) (*gitprovider.RepositoryPermission, error) {
	var permissionObj gitprovider.RepositoryPermission
	var ok bool

	if permissionObj, ok = permissionPriority[permissionLevel]; ok {
		return &permissionObj, nil
	}
	return nil, gitprovider.ErrInvalidPermissionLevel
}

func getStashPermissionFromMap(permissionMap map[string]bool) int {
	lastPriority := 0
	for key, ok := range permissionMap {
		if ok {
			priority, ok := stashPriority[key]
			if ok && priority > lastPriority {
				lastPriority = priority
			}
		}
	}
	return lastPriority
}

func getStashPermission(permission gitprovider.RepositoryPermission) (string, error) {
	for key, value := range permissionPriority {
		if value == permission {
			for stashPerm, v := range stashPriority {
				if v == key {
					return stashPerm, nil
				}
			}
		}
	}
	return "", gitprovider.ErrInvalidPermissionLevel
}
