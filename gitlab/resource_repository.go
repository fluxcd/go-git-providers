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
	"context"
	"errors"

	"github.com/google/go-cmp/cmp"
	gogitlab "github.com/xanzy/go-gitlab"

	"github.com/fluxcd/go-git-providers/gitprovider"
)

func newUserProject(ctx *clientContext, apiObj *gogitlab.Project, ref gitprovider.RepositoryRef) *userProject {
	return &userProject{
		clientContext: ctx,
		p:             *apiObj,
		ref:           ref,
		deployKeys: &DeployKeyClient{
			clientContext: ctx,
			ref:           ref,
		},
		commits: &CommitClient{
			clientContext:ctx,
			ref:ref,
		},
		branches: &BranchClient{
			clientContext:ctx,
			ref:ref,
		},
		pullRequests: &PullRequestClient{
			clientContext:ctx,
			ref:ref,
		},
	}
}

var _ gitprovider.UserRepository = &userProject{}

type userProject struct {
	*clientContext

	p   gogitlab.Project
	ref gitprovider.RepositoryRef

	deployKeys *DeployKeyClient
	commits *CommitClient
	branches *BranchClient
	pullRequests *PullRequestClient
}

func (p *userProject) Get() gitprovider.RepositoryInfo {
	return repositoryFromAPI(&p.p)
}

func (p *userProject) Set(info gitprovider.RepositoryInfo) error {
	if err := info.ValidateInfo(); err != nil {
		return err
	}
	repositoryInfoToAPIObj(&info, &p.p)
	return nil
}

func (p *userProject) APIObject() interface{} {
	return &p.p
}

func (p *userProject) Repository() gitprovider.RepositoryRef {
	return p.ref
}

func (p *userProject) DeployKeys() gitprovider.DeployKeyClient {
	return p.deployKeys
}

func (p *userProject) Commits() gitprovider.CommitClient {
	return p.commits
}

func (p *userProject) Branches() gitprovider.BranchClient {
	return p.branches
}

func (p *userProject) PullRequests() gitprovider.PullRequestClient {
	return p.pullRequests
}

// The internal API object will be overridden with the received server data.
func (p *userProject) Update(ctx context.Context) error {
	// PATCH /repos/{owner}/{repo}
	apiObj, err := p.c.UpdateProject(ctx, &p.p)
	if err != nil {
		return err
	}
	p.p = *apiObj
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
func (p *userProject) Reconcile(ctx context.Context) (bool, error) {
	apiObj, err := p.c.GetUserProject(ctx, getRepoPath(p.ref))
	if err != nil {
		// Create if not found
		if errors.Is(err, gitprovider.ErrNotFound) {
			// orgName := ""
			// if orgRef, ok := p.ref.(gitprovider.OrgRepositoryRef); ok {
			// 	orgName = orgRef.Organization
			// }
			project, err := p.c.CreateProject(ctx, &p.p, nil)
			if err != nil {
				return true, err
			}
			p.p = *project
			return true, nil
		}

		return false, err
	}

	// Use wrappers here to extract the "spec" part of the object for comparison
	desiredSpec := newGitlabProjectSpec(&p.p)
	actualSpec := newGitlabProjectSpec(apiObj)

	// If desired state already is the actual state, do nothing
	if desiredSpec.Equals(actualSpec) {
		return false, nil
	}
	// Otherwise, make the desired state the actual state
	return true, p.Update(ctx)
}

// Delete deletes the current resource irreversibly.
//
// ErrNotFound is returned if the resource doesn't exist anymore.
func (p *userProject) Delete(ctx context.Context) error {
	return p.c.DeleteProject(ctx, getRepoPath(p.ref))
}

func newGroupProject(ctx *clientContext, apiObj *gogitlab.Project, ref gitprovider.RepositoryRef) *orgRepository {
	return &orgRepository{
		userProject: *newUserProject(ctx, apiObj, ref),
		teamAccess: &TeamAccessClient{
			clientContext: ctx,
			ref:           ref,
		},
	}
}

var _ gitprovider.OrgRepository = &orgRepository{}

type orgRepository struct {
	userProject

	teamAccess *TeamAccessClient
}

func (r *orgRepository) TeamAccess() gitprovider.TeamAccessClient {
	return r.teamAccess
}

func (r *orgRepository) Commits() gitprovider.CommitClient {
	return r.commits
}

func (r *orgRepository) Branches() gitprovider.BranchClient {
	return r.branches
}

// Reconcile makes sure the desired state in this object (called "req" here) becomes
// the actual state in the backing Git provider.
//
// If req doesn't exist under the hood, it is created (actionTaken == true).
// If req doesn't equal the actual state, the resource will be updated (actionTaken == true).
// If req is already the actual state, this is a no-op (actionTaken == false).
//
// The internal API object will be overridden with the received server data if actionTaken == true.
func (r *orgRepository) Reconcile(ctx context.Context) (bool, error) {
	apiObj, err := r.c.GetGroupProject(ctx, r.ref.GetIdentity(), r.ref.GetRepository())
	if err != nil {
		// Create if not found
		if errors.Is(err, gitprovider.ErrNotFound) {
			project, err := r.c.CreateProject(ctx, &r.p, nil)
			if err != nil {
				return true, err
			}
			r.p = *project
			return true, nil
		}

		return false, err
	}

	// Use wrappers here to extract the "spec" part of the object for comparison
	desiredSpec := newGitlabProjectSpec(&r.p)
	actualSpec := newGitlabProjectSpec(apiObj)

	// If desired state already is the actual state, do nothing
	if desiredSpec.Equals(actualSpec) {
		return false, nil
	}
	// Otherwise, make the desired state the actual state
	return true, r.Update(ctx)
}

func repositoryFromAPI(apiObj *gogitlab.Project) gitprovider.RepositoryInfo {
	repo := gitprovider.RepositoryInfo{
		Description:   &apiObj.Description,
		DefaultBranch: &apiObj.DefaultBranch,
	}
	repo.Visibility = gitprovider.RepositoryVisibilityVar(gitprovider.RepositoryVisibility(apiObj.Visibility))
	return repo
}

func repositoryToAPI(repo *gitprovider.RepositoryInfo, ref gitprovider.RepositoryRef) gogitlab.Project {
	apiObj := gogitlab.Project{
		Name: *gitprovider.StringVar(ref.GetRepository()),
	}
	repositoryInfoToAPIObj(repo, &apiObj)
	return apiObj
}

func repositoryInfoToAPIObj(repo *gitprovider.RepositoryInfo, apiObj *gogitlab.Project) {
	if repo.Description != nil {
		apiObj.Description = *repo.Description
	}
	if repo.DefaultBranch != nil {
		apiObj.DefaultBranch = *repo.DefaultBranch
	}
	if repo.Visibility != nil {
		apiObj.Visibility = gitlabVisibilityMap[*repo.Visibility]
	}
}

// This function copies over the fields that are part of create/update requests of a project
// i.e. the desired spec of the repository. This allows us to separate "spec" from "status" fields.
func newGitlabProjectSpec(project *gogitlab.Project) *gitlabProjectSpec {
	return &gitlabProjectSpec{
		&gogitlab.Project{
			// Generic
			Name:        project.Name,
			Namespace:   project.Namespace,
			Description: project.Description,
			Visibility:  project.Visibility,

			// Update-specific parameters
			DefaultBranch: project.DefaultBranch,
		},
	}
}

type gitlabProjectSpec struct {
	*gogitlab.Project
}

func (s *gitlabProjectSpec) Equals(other *gitlabProjectSpec) bool {
	return cmp.Equal(s, other)
}

//nolint
var gitlabVisibilityMap = map[gitprovider.RepositoryVisibility]gogitlab.VisibilityValue{
	gitprovider.RepositoryVisibilityInternal: gogitlab.InternalVisibility,
	gitprovider.RepositoryVisibilityPrivate:  gogitlab.PrivateVisibility,
	gitprovider.RepositoryVisibilityPublic:   gogitlab.PublicVisibility,
}
