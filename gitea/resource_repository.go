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

package gitea

import (
	"context"
	"errors"
	"reflect"

	"code.gitea.io/sdk/gitea"

	"github.com/fluxcd/go-git-providers/gitprovider"
	"github.com/fluxcd/go-git-providers/validation"
)

func newUserRepository(ctx *clientContext, apiObj *gitea.Repository, ref gitprovider.RepositoryRef) *userRepository {
	return &userRepository{
		clientContext: ctx,
		r:             *apiObj,
		ref:           ref,
		deployKeys: &DeployKeyClient{
			clientContext: ctx,
			ref:           ref,
		},
		commits: &CommitClient{
			clientContext: ctx,
			ref:           ref,
		},
		branches: &BranchClient{
			clientContext: ctx,
			ref:           ref,
		},
		pullRequests: &PullRequestClient{
			clientContext: ctx,
			ref:           ref,
		},
		files: &FileClient{
			clientContext: ctx,
			ref:           ref,
		},
	}
}

var _ gitprovider.UserRepository = &userRepository{}

type userRepository struct {
	*clientContext

	r   gitea.Repository // gitea
	ref gitprovider.RepositoryRef

	deployKeys   *DeployKeyClient
	commits      *CommitClient
	branches     *BranchClient
	pullRequests *PullRequestClient
	files        *FileClient
	trees        *TreeClient
}

// Get returns the repository information.
func (r *userRepository) Get() gitprovider.RepositoryInfo {
	return repositoryFromAPI(&r.r)
}

// Set sets the repository information.
func (r *userRepository) Set(info gitprovider.RepositoryInfo) error {
	if err := info.ValidateInfo(); err != nil {
		return err
	}
	repositoryInfoToAPIObj(&info, &r.r)
	return nil
}

// APIObject returns the underlying API object.
func (r *userRepository) APIObject() interface{} {
	return &r.r
}

// Repository returns the repository reference.
func (r *userRepository) Repository() gitprovider.RepositoryRef {
	return r.ref
}

// DeployKeys returns the deploy key client.
func (r *userRepository) DeployKeys() gitprovider.DeployKeyClient {
	return r.deployKeys
}

// DeployTokens returns the deploy token client.
// ErrNoProviderSupport is returned as the provider does not support deploy tokens.
func (r *userRepository) DeployTokens() (gitprovider.DeployTokenClient, error) {
	return nil, gitprovider.ErrNoProviderSupport
}

// Commits returns the commit client.
func (r *userRepository) Commits() gitprovider.CommitClient {
	return r.commits
}

// Branches returns the branch client.
func (r *userRepository) Branches() gitprovider.BranchClient {
	return r.branches
}

// PullRequests returns the pull request client.
func (r *userRepository) PullRequests() gitprovider.PullRequestClient {
	return r.pullRequests
}

// Files returns the file client.
func (r *userRepository) Files() gitprovider.FileClient {
	return r.files
}

// Trees returns the tree client.
func (r *userRepository) Trees() gitprovider.TreeClient {
	return r.trees
}

// Update will apply the desired state in this object to the server.
// Only set fields will be respected (i.e. PATCH behaviour).
// In order to apply changes to this object, use the .Set({Resource}Info) error
// function, or cast .APIObject() to a pointer to the provider-specific type
// and set custom fields there.
//
// ErrNotFound is returned if the resource does not exist.
//
// The internal API object will be overridden with the received server data.
func (r *userRepository) Update(ctx context.Context) error {
	// PATCH /repos/{owner}/{repo}
	opts := gitea.EditRepoOption{
		Name:                      &r.r.Name,
		Description:               &r.r.Description,
		Website:                   &r.r.Website,
		Private:                   &r.r.Private,
		Template:                  &r.r.Template,
		HasIssues:                 &r.r.HasIssues,
		InternalTracker:           r.r.InternalTracker,
		ExternalTracker:           r.r.ExternalTracker,
		HasWiki:                   &r.r.HasWiki,
		ExternalWiki:              r.r.ExternalWiki,
		DefaultBranch:             &r.r.DefaultBranch,
		HasPullRequests:           &r.r.HasPullRequests,
		HasProjects:               &r.r.HasProjects,
		IgnoreWhitespaceConflicts: &r.r.IgnoreWhitespaceConflicts,
		AllowMerge:                &r.r.AllowMerge,
		AllowRebase:               &r.r.AllowRebase,
		AllowRebaseMerge:          &r.r.AllowRebaseMerge,
		AllowSquash:               &r.r.AllowSquash,
		Archived:                  &r.r.Archived,
		DefaultMergeStyle:         &r.r.DefaultMergeStyle,
	}
	if r.r.Mirror {
		opts.MirrorInterval = &r.r.MirrorInterval
	}
	apiObj, err := updateRepo(r.c, r.ref.GetIdentity(), r.ref.GetRepository(), &opts)
	if err != nil {
		return err
	}
	r.r = *apiObj
	return nil
}

// Reconcile makes sure the desired state in this object (called "req" here) becomes
// the actual state in the backing Git provider.
//
// If req doesn't exist under the hood, it is created (actionTaken == true).
// If req doesn't equal the actual state, the resource will be updated (actionTaken == true).
// If req is already the actual state, this is a no-op (actionTaken == false).
//
// The internal API object will be overridden with the received server data if actionTaken == true.
func (r *userRepository) Reconcile(ctx context.Context) (bool, error) {
	opts := gitea.CreateRepoOption{
		Name:          r.r.Name,
		Description:   r.r.Description,
		Private:       r.r.Private,
		Template:      r.r.Template,
		DefaultBranch: r.r.DefaultBranch,
	}
	apiObj, err := getRepo(r.c, r.ref.GetIdentity(), r.ref.GetRepository())
	if err != nil {
		// Create if not found
		if errors.Is(err, gitprovider.ErrNotFound) {
			orgName := ""
			if orgRef, ok := r.ref.(gitprovider.OrgRepositoryRef); ok {
				orgName = orgRef.Organization
			}
			repo, err := createRepo(r.c, orgName, opts)
			if err != nil {
				return true, err
			}
			r.r = *repo
			return true, nil
		}

		return false, err
	}

	// Use wrappers here to extract the "spec" part of the object for comparison
	desiredSpec := newGiteaRepositorySpec(&r.r)
	actualSpec := newGiteaRepositorySpec(apiObj)

	// If desired state already is the actual state, do nothing
	if desiredSpec.Equals(actualSpec) {
		return false, nil
	}
	// Otherwise, make the desired state the actual state
	return true, r.Update(ctx)
}

// Delete deletes the current resource irreversibly.
//
// ErrNotFound is returned if the resource doesn't exist anymore.
func (r *userRepository) Delete(ctx context.Context) error {
	return deleteRepo(r.c, r.ref.GetIdentity(), r.ref.GetRepository(), r.destructiveActions)
}

func newOrgRepository(ctx *clientContext, apiObj *gitea.Repository, ref gitprovider.RepositoryRef) *orgRepository {
	return &orgRepository{
		userRepository: *newUserRepository(ctx, apiObj, ref),
		teamAccess: &TeamAccessClient{
			clientContext: ctx,
			ref:           ref,
		},
	}
}

var _ gitprovider.OrgRepository = &orgRepository{}

type orgRepository struct {
	userRepository

	teamAccess *TeamAccessClient
}

// TeamAccess returns the team access client.
func (r *orgRepository) TeamAccess() gitprovider.TeamAccessClient {
	return r.teamAccess
}

// validateRepositoryAPI validates the apiObj received from the server, to make sure that it is
// valid for our use.
func validateRepositoryAPI(apiObj *gitea.Repository) error {
	return validateAPIObject("Gitea.Repository", func(validator validation.Validator) {
		// Make sure name is set
		if apiObj.Name == "" {
			validator.Required("Name")
		}
		// Make sure visibility is valid if set
		if !apiObj.Private {
			v := gitprovider.RepositoryVisibility("public")
			validator.Append(gitprovider.ValidateRepositoryVisibility(v), v, "Visibility")
		} else {
			v := gitprovider.RepositoryVisibility("private")
			validator.Append(gitprovider.ValidateRepositoryVisibility(v), v, "Visibility")
		}
	})
}

func repositoryFromAPI(apiObj *gitea.Repository) gitprovider.RepositoryInfo {
	repo := gitprovider.RepositoryInfo{
		Description:   &apiObj.Description,
		DefaultBranch: &apiObj.DefaultBranch,
	}
	if !apiObj.Private {
		repo.Visibility = gitprovider.RepositoryVisibilityVar(gitprovider.RepositoryVisibility("public"))
	} else {
		repo.Visibility = gitprovider.RepositoryVisibilityVar(gitprovider.RepositoryVisibility("private"))
	}
	return repo
}

func repositoryToAPI(repo *gitprovider.RepositoryInfo, ref gitprovider.RepositoryRef) gitea.CreateRepoOption {
	apiObj := gitea.CreateRepoOption{
		Name: *gitprovider.StringVar(ref.GetRepository()),
	}
	repositoryInfoToCreateOption(repo, &apiObj)
	return apiObj
}

func repositoryInfoToCreateOption(repo *gitprovider.RepositoryInfo, apiObj *gitea.CreateRepoOption) {
	if repo.Description != nil {
		apiObj.Description = *repo.Description
	}
	if repo.DefaultBranch != nil {
		apiObj.DefaultBranch = *repo.DefaultBranch
	}
	if repo.Visibility != nil {
		apiObj.Private = *gitprovider.BoolVar(string(*repo.Visibility) == "private")
	}
}

func repositoryInfoToAPIObj(repo *gitprovider.RepositoryInfo, apiObj *gitea.Repository) {
	if repo.Description != nil {
		apiObj.Description = *repo.Description
	}
	if repo.DefaultBranch != nil {
		apiObj.DefaultBranch = *repo.DefaultBranch
	}
	if repo.Visibility != nil {
		apiObj.Private = *gitprovider.BoolVar(string(*repo.Visibility) == "private")
	}
}

// This function copies over the fields that are part of create/update requests of a repository
// i.e. the desired spec of the repository. This allows us to separate "spec" from "status" fields.
// See also: https://gitea.com/api/swagger#/repository/createCurrentUserRepo
func newGiteaRepositorySpec(repo *gitea.Repository) *giteaRepositorySpec {
	return &giteaRepositorySpec{
		&gitea.Repository{
			// Generic
			Name:        repo.Name,
			Description: repo.Description,
			Website:     repo.Website,
			Private:     repo.Private,
			HasIssues:   repo.HasIssues,
			HasProjects: repo.HasProjects,
			HasWiki:     repo.HasWiki,
			Internal:    repo.Internal,

			// Update-specific parameters
			DefaultBranch: repo.DefaultBranch,

			// Create-specific parameters

			// Generic
			AllowSquash: repo.AllowSquash,
			AllowMerge:  repo.AllowMerge,
			AllowRebase: repo.AllowRebase,
		},
	}
}

type giteaRepositorySpec struct {
	*gitea.Repository
}

// Equals compares two giteaRepositorySpec objects for equality.
func (s *giteaRepositorySpec) Equals(other *giteaRepositorySpec) bool {
	return reflect.DeepEqual(s, other)
}
