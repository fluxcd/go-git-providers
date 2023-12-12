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
	"errors"
	"reflect"

	"github.com/google/go-github/v57/github"

	"github.com/fluxcd/go-git-providers/gitprovider"
	"github.com/fluxcd/go-git-providers/validation"
)

var githubRepositoryKnownFields = map[string]struct{}{
	"Name":        {},
	"Description": {},
	"Homepage":    {},
	"Private":     {},
	"Visibility":  {},
	"HasIssues":   {},
	"HasProjects": {},
	"HasWiki":     {},
	"IsTemplate":  {},
	// Update-specific parameters
	// See: https://docs.github.com/en/rest/reference/repos#update-a-repository
	"DefaultBranch": {},
	// Create-specific parameters
	// See: https://docs.github.com/en/rest/reference/repos#create-an-organization-repository
	"TeamID":            {},
	"AutoInit":          {},
	"GitignoreTemplate": {},
	"LicenseTemplate":   {},
	// Generic
	"AllowSquashMerge":    {},
	"AllowMergeCommit":    {},
	"AllowRebaseMerge":    {},
	"DeleteBranchOnMerge": {},
}

func newUserRepository(ctx *clientContext, apiObj *github.Repository, ref gitprovider.RepositoryRef) *userRepository {
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
		trees: &TreeClient{
			clientContext: ctx,
			ref:           ref,
		},
	}
}

var _ gitprovider.UserRepository = &userRepository{}

type userRepository struct {
	*clientContext

	r         github.Repository // go-github
	topUpdate *github.Repository
	ref       gitprovider.RepositoryRef

	deployKeys   *DeployKeyClient
	commits      *CommitClient
	branches     *BranchClient
	pullRequests *PullRequestClient
	files        *FileClient
	trees        *TreeClient
}

func (r *userRepository) Get() gitprovider.RepositoryInfo {
	return repositoryFromAPI(&r.r)
}

// Set sets the desired state of this object.
// User have to call Update() to apply the changes to the server.
// The changes will then be reflected in the internal API object.
func (r *userRepository) Set(info gitprovider.RepositoryInfo) error {
	if err := info.ValidateInfo(); err != nil {
		return err
	}
	r.topUpdate = updateApiObjWithRepositoryInfo(&info, &r.r)
	return nil
}

func (r *userRepository) APIObject() interface{} {
	return &r.r
}

func (r *userRepository) Repository() gitprovider.RepositoryRef {
	return r.ref
}

func (r *userRepository) DeployKeys() gitprovider.DeployKeyClient {
	return r.deployKeys
}

func (r *userRepository) DeployTokens() (gitprovider.DeployTokenClient, error) {
	return nil, gitprovider.ErrNoProviderSupport
}

func (r *userRepository) Commits() gitprovider.CommitClient {
	return r.commits
}

func (r *userRepository) Branches() gitprovider.BranchClient {
	return r.branches
}

func (r *userRepository) PullRequests() gitprovider.PullRequestClient {
	return r.pullRequests
}

func (r *userRepository) Files() gitprovider.FileClient {
	return r.files
}

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
	apiObj, err := r.c.UpdateRepo(ctx, r.ref.GetIdentity(), r.ref.GetRepository(), r.topUpdate)
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
	apiObj, err := r.c.GetRepo(ctx, r.ref.GetIdentity(), r.ref.GetRepository())
	if err != nil {
		// Create if not found
		if errors.Is(err, gitprovider.ErrNotFound) {
			orgName := ""
			if orgRef, ok := r.ref.(gitprovider.OrgRepositoryRef); ok {
				orgName = orgRef.Organization
			}
			repo, err := r.c.CreateRepo(ctx, orgName, &r.r)
			if err != nil {
				return true, err
			}
			r.r = *repo
			return true, nil
		}

		return false, err
	}

	// Use wrappers here to extract the "spec" part of the object for comparison
	desiredSpec := newGithubRepositorySpec(&r.r)
	actualSpec := newGithubRepositorySpec(apiObj)

	// If desired state already is the actual state, do nothing
	if desiredSpec.Equals(actualSpec) {
		return false, nil
	}
	// Otherwise, make the desired state the actual state
	// create the update repository
	r.topUpdate = updateGithubRepository(desiredSpec.Repository, actualSpec.Repository)

	return true, r.Update(ctx)
}

// Delete deletes the current resource irreversibly.
//
// ErrNotFound is returned if the resource doesn't exist anymore.
func (r *userRepository) Delete(ctx context.Context) error {
	return r.c.DeleteRepo(ctx, r.ref.GetIdentity(), r.ref.GetRepository())
}

func newOrgRepository(ctx *clientContext, apiObj *github.Repository, ref gitprovider.RepositoryRef) *orgRepository {
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

func (r *orgRepository) TeamAccess() gitprovider.TeamAccessClient {
	return r.teamAccess
}

// validateRepositoryAPI validates the apiObj received from the server, to make sure that it is
// valid for our use.
func validateRepositoryAPI(apiObj *github.Repository) error {
	return validateAPIObject("GitHub.Repository", func(validator validation.Validator) {
		// Make sure name is set
		if apiObj.Name == nil {
			validator.Required("Name")
		}
		// Make sure visibility is valid if set
		if apiObj.Visibility != nil {
			v := gitprovider.RepositoryVisibility(*apiObj.Visibility)
			validator.Append(gitprovider.ValidateRepositoryVisibility(v), v, "Visibility")
		}
	})
}

func repositoryFromAPI(apiObj *github.Repository) gitprovider.RepositoryInfo {
	repo := gitprovider.RepositoryInfo{
		Description:   apiObj.Description,
		DefaultBranch: apiObj.DefaultBranch,
	}
	if apiObj.Visibility != nil {
		repo.Visibility = gitprovider.RepositoryVisibilityVar(gitprovider.RepositoryVisibility(*apiObj.Visibility))
	}
	return repo
}

func repositoryToAPI(repo *gitprovider.RepositoryInfo, ref gitprovider.RepositoryRef) github.Repository {
	apiObj := github.Repository{
		Name: gitprovider.StringVar(ref.GetRepository()),
	}
	repositoryInfoToAPIObj(repo, &apiObj)
	return apiObj
}

func repositoryInfoToAPIObj(repo *gitprovider.RepositoryInfo, apiObj *github.Repository) {
	if repo.Description != nil {
		apiObj.Description = repo.Description
	}
	if repo.DefaultBranch != nil {
		apiObj.DefaultBranch = repo.DefaultBranch
	}
	if repo.Visibility != nil {
		apiObj.Visibility = gitprovider.StringVar(string(*repo.Visibility))
	}
}

func updateApiObjWithRepositoryInfo(repo *gitprovider.RepositoryInfo, apiObj *github.Repository) *github.Repository {
	actual := newGithubRepositorySpec(apiObj).Repository
	desired := newGithubRepositorySpec(apiObj).Repository

	if repo.Description != nil {
		desired.Description = repo.Description
	}
	if repo.DefaultBranch != nil {
		desired.DefaultBranch = repo.DefaultBranch
	}
	if repo.Visibility != nil {
		desired.Visibility = gitprovider.StringVar(string(*repo.Visibility))
	}

	// create the update repository
	return updateGithubRepository(desired, actual)
}

func applyRepoCreateOptions(apiObj *github.Repository, opts gitprovider.RepositoryCreateOptions) {
	apiObj.AutoInit = opts.AutoInit
	if opts.LicenseTemplate != nil {
		apiObj.LicenseTemplate = gitprovider.StringVar(string(*opts.LicenseTemplate))
	}
}

// This function copies over the fields that are part of create/update requests of a repository
// i.e. the desired spec of the repository. This allows us to separate "spec" from "status" fields.
// See also: https://github.com/google/go-github/blob/master/github/repos.go#L340-L358
func newGithubRepositorySpec(repo *github.Repository) *githubRepositorySpec {
	rep := &github.Repository{}
	v := reflect.ValueOf(rep).Elem()
	t := reflect.TypeOf(rep).Elem()
	// loop over all fields in the struct
	// set the value of the field in the struct to the value of the field in the repo obj
	// if the field is a known field
	for i := 0; i < v.NumField(); i++ {
		f := t.Field(i)
		_, ok := githubRepositoryKnownFields[f.Name]
		if ok {
			val := reflect.ValueOf(repo).Elem().FieldByName(f.Name)
			if v.Field(i).CanSet() {
				v.Field(i).Set(val)
			}
		}
	}

	return &githubRepositorySpec{rep}
}

type githubRepositorySpec struct {
	*github.Repository
}

func (s *githubRepositorySpec) Equals(other *githubRepositorySpec) bool {
	return reflect.DeepEqual(s, other)
}

func updateGithubRepository(desired, actual *github.Repository) *github.Repository {
	// create the updated repository
	u := &github.Repository{
		Name: desired.Name,
	}

	uVal := reflect.ValueOf(u).Elem()
	desiredVal := reflect.ValueOf(desired).Elem()
	actualVal := reflect.ValueOf(actual).Elem()
	t := reflect.TypeOf(actual).Elem()
	// loop over all fields in the struct
	// and set the fields in the update repository if they are set in the desired state
	// and not the actual state
	for i := 0; i < desiredVal.NumField(); i++ {
		f := t.Field(i)
		if f.Name != "Name" && (desiredVal.FieldByName(f.Name) != actualVal.FieldByName(f.Name)) {
			if uVal.FieldByName(f.Name).CanSet() {
				uVal.FieldByName(f.Name).Set(desiredVal.FieldByName(f.Name))
			}
		}
	}

	return u
}
