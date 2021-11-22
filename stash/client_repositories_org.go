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
	"errors"
	"fmt"

	"github.com/fluxcd/go-git-providers/gitprovider"
	"github.com/fluxcd/go-git-providers/validation"
	"github.com/hashicorp/go-multierror"
)

const legacyBranch = "master"

// OrgRepositoriesClient implements the gitprovider.OrgRepositoriesClient interface.
var _ gitprovider.OrgRepositoriesClient = &OrgRepositoriesClient{}

// OrgRepositoriesClient operates on repositories the user has access to.
type OrgRepositoriesClient struct {
	*clientContext
}

// Get returns the repository at the given path.
// ErrNotFound is returned if the resource does not exist.
func (c *OrgRepositoriesClient) Get(ctx context.Context, ref gitprovider.OrgRepositoryRef) (gitprovider.OrgRepository, error) {
	// Make sure the OrgRepositoryRef is valid
	if err := validateOrgRepositoryRef(ref, c.host); err != nil {
		return nil, err
	}

	slug := ref.Slug()
	if slug == "" {
		// try with name
		slug = ref.GetRepository()
	}

	apiObj, err := c.client.Repositories.Get(ctx, ref.Key(), slug)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, gitprovider.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get repository %s/%s: %w", ref.Key(), slug, err)
	}

	// Validate the API objects
	if err := validateRepositoryAPI(apiObj); err != nil {
		return nil, err
	}

	ref.SetSlug(apiObj.Slug)

	// Get the default branch
	branch, err := c.client.Branches.Default(ctx, ref.Key(), slug)
	if err != nil {
		return nil, fmt.Errorf("failed to get default branch for repository %s/%s: %w", ref.Key(), slug, err)
	}

	apiObj.DefaultBranch = branch.DisplayID

	return newOrgRepository(c.clientContext, apiObj, ref), nil
}

// List all repositories in the given organization.
// List returns all available repositories, using multiple paginated requests if needed.
func (c *OrgRepositoriesClient) List(ctx context.Context, ref gitprovider.OrganizationRef) ([]gitprovider.OrgRepository, error) {
	// Make sure the OrganizationRef is valid
	if err := validateOrganizationRef(ref, c.host); err != nil {
		return nil, err
	}

	apiObjs, err := c.client.Repositories.All(ctx, ref.Key())
	if err != nil {
		return nil, fmt.Errorf("failed to list repositories: %w", err)
	}

	var errs error
	for _, apiObj := range apiObjs {
		if err := validateRepositoryAPI(apiObj); err != nil {
			errs = multierror.Append(errs, err)
		}
	}

	if errs != nil {
		return nil, errs
	}

	// Traverse the list, and return a list of OrgRepository objects
	repos := make([]gitprovider.OrgRepository, 0, len(apiObjs))
	for _, apiObj := range apiObjs {
		repoRef := gitprovider.OrgRepositoryRef{
			OrganizationRef: ref,
			RepositoryName:  apiObj.Name,
		}
		repoRef.SetSlug(apiObj.Slug)

		repos = append(repos, newOrgRepository(c.clientContext, apiObj, repoRef))
	}
	return repos, nil
}

// Create creates a repository for the given organization, with the data and options.
// ErrAlreadyExists will be returned if the resource already exists.
func (c *OrgRepositoriesClient) Create(ctx context.Context,
	ref gitprovider.OrgRepositoryRef,
	req gitprovider.RepositoryInfo,
	opts ...gitprovider.RepositoryCreateOption) (gitprovider.OrgRepository, error) {
	// Make sure the RepositoryRef is valid
	if err := validateOrgRepositoryRef(ref, c.host); err != nil {
		return nil, err
	}

	apiObj, err := createRepository(ctx, c.client, ref.Key(), ref, req, opts...)
	if err != nil {
		if errors.Is(err, ErrAlreadyExists) {
			return nil, gitprovider.ErrAlreadyExists
		}
		return nil, fmt.Errorf("failed to create repository %s/%s: %w", ref.Key(), ref.Slug(), err)
	}

	ref.SetSlug(apiObj.Slug)

	return newOrgRepository(c.clientContext, apiObj, ref), nil
}

// Reconcile makes sure the given desired state (req) becomes the actual state in the backing Git provider.
// If req doesn't exist under the hood, it is created (actionTaken == true).
// If req doesn't equal the actual state, the resource will be updated (actionTaken == true).
// If req is already the actual state, this is a no-op (actionTaken == false).
func (c *OrgRepositoriesClient) Reconcile(ctx context.Context, ref gitprovider.OrgRepositoryRef, req gitprovider.RepositoryInfo, opts ...gitprovider.RepositoryReconcileOption) (gitprovider.OrgRepository, bool, error) {
	actual, err := c.Get(ctx, ref)
	if err != nil {
		// Create if not found
		if errors.Is(err, gitprovider.ErrNotFound) {
			resp, err := c.Create(ctx, ref, req, toCreateOpts(opts...)...)
			return resp, true, err
		}

		// Unexpected path, Get should succeed or return NotFound
		return nil, false, fmt.Errorf("unexpected error when reconciling repository: %w", err)
	}

	actionTaken, err := c.reconcileRepository(ctx, actual, req)

	return actual, actionTaken, err
}

// update will apply the desired state in this object to the server.
// ErrNotFound is returned if the resource does not exist.
func update(ctx context.Context, c *Client, orgKey, repoSlug string, repository *Repository, branchID string) (*Repository, error) {
	apiObj, err := c.Repositories.Update(ctx, orgKey, repoSlug, repository)
	if err != nil {
		return nil, fmt.Errorf("failed to update repository: %w", err)
	}

	apiObj.DefaultBranch = repository.DefaultBranch

	// Update default branch
	if branchID != "" {
		// update default branch
		if err := c.Branches.SetDefault(ctx, orgKey, repoSlug, fmt.Sprintf("refs/heads/%s", branchID)); err != nil {
			return nil, fmt.Errorf("failed to update default branch: %w", err)
		}

		apiObj.DefaultBranch = branchID
	}

	return apiObj, nil
}

func deleteRepository(ctx context.Context, c *Client, orgKey, repoSlug string) error {
	if err := c.Repositories.Delete(ctx, orgKey, repoSlug); err != nil {
		return fmt.Errorf("failed to delete repository: %w", err)
	}

	return nil
}

func createRepository(ctx context.Context, c *Client, orgKey string, ref gitprovider.RepositoryRef, req gitprovider.RepositoryInfo, opts ...gitprovider.RepositoryCreateOption) (*Repository, error) {
	// First thing, validate and default the request to ensure a valid and fully-populated object
	// (to minimize any possible diffs between desired and actual state)
	if err := gitprovider.ValidateAndDefaultInfo(&req); err != nil {
		return nil, err
	}

	// Assemble the options struct based on the given options
	opt, err := gitprovider.MakeRepositoryCreateOptions(opts...)
	if err != nil {
		return nil, err
	}

	// Convert to the API object and apply the options
	data := repositoryToAPI(&req, ref)
	if err != nil {
		return nil, err
	}

	repo, err := c.Repositories.Create(ctx, orgKey, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create repository: %w", err)
	}

	user, err := c.Users.Get(ctx, repo.Session.UserName)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	var initCommit *CreateCommit

	if opt.AutoInit != nil && *(opt.AutoInit) {
		readmeContents := fmt.Sprintf("# %s\n%s", repo.Name, repo.Description)
		readmePath, licensePath := "README.md", "LICENSE.md"
		files := []CommitFile{
			{
				Path:    &readmePath,
				Content: &readmeContents,
			},
		}
		var licenseContent string
		if opt.LicenseTemplate != nil {
			licenseContent, err = getLicense(*opt.LicenseTemplate)
			// If the license template is invalid, we'll just skip the license
			if err == nil {
				files = append(files, CommitFile{
					Path:    &licensePath,
					Content: &licenseContent,
				})
			}
		}

		initCommit, err = NewCommit(
			WithAuthor(&CommitAuthor{
				Name:  user.Name,
				Email: user.EmailAddress,
			}),
			WithMessage("initial commit"),
			WithURL(getRepoHTTPref(repo.Links.Clone)),
			WithFiles(files))

		if err != nil {
			return nil, fmt.Errorf("failed to create initial commit: %w", err)
		}

		err = initRepo(ctx, c, initCommit, repo)

		if data.DefaultBranch != "" && data.DefaultBranch != legacyBranch {
			//create default branch
			br, err := setDefaultBranch(ctx, c, orgKey, data.DefaultBranch, repo)
			if err != nil {
				return nil, fmt.Errorf("failed to create default branch: %w", err)
			}
			// save the default branch after setting it
			repo.DefaultBranch = br.DisplayID
		}
	} else if data.DefaultBranch != "" && data.DefaultBranch != legacyBranch {
		// Init repo anyway because we need to set the default branch and for now we have an empty repo.
		// Stash set it by default to master so we use that to branch from.
		initCommit, err = NewCommit(
			WithAuthor(&CommitAuthor{
				Name:  user.Name,
				Email: user.EmailAddress,
			}),
			WithMessage("initial commit"),
			WithURL(getRepoHTTPref(repo.Links.Clone)))

		if err != nil {
			return nil, fmt.Errorf("failed to create initial commit: %w", err)
		}

		err = initRepo(ctx, c, initCommit, repo)
		br, err := setDefaultBranch(ctx, c, orgKey, data.DefaultBranch, repo)
		if err != nil {
			return nil, fmt.Errorf("failed to create default branch: %w", err)
		}

		// save the default branch after setting it
		repo.DefaultBranch = br.DisplayID
	}

	return repo, nil
}

func setDefaultBranch(ctx context.Context, c *Client, orgKey, branch string, repo *Repository) (*Branch, error) {
	//create default branch
	br, err := c.Branches.Create(ctx, orgKey, repo.Slug, fmt.Sprintf("refs/heads/%s", branch), fmt.Sprintf("refs/heads/%s", legacyBranch))
	if err != nil {
		return nil, fmt.Errorf("failed to create default branch: %w", err)
	}

	if err := c.Branches.SetDefault(ctx, orgKey, repo.Slug, fmt.Sprintf("refs/heads/%s", branch)); err != nil {
		return nil, fmt.Errorf("failed to set default branch: %w", err)
	}

	return br, nil
}

func initRepo(ctx context.Context, c *Client, initCommit *CreateCommit, repo *Repository) error {
	r, dir, err := c.Git.InitRepository(initCommit, true)
	if err != nil {
		if err := c.Repositories.Delete(ctx, repo.Project.Key, repo.Slug); err != nil {
			return fmt.Errorf("failed to delete repository: %w", err)
		}
		return fmt.Errorf("failed to init repository: %w", err)
	}

	err = c.Git.Push(ctx, r)
	if err != nil {
		return fmt.Errorf("failed to push initial commit: %w", err)
	}

	err = c.Git.Cleanup(dir)
	if err != nil {
		return fmt.Errorf("failed to cleanup repository: %w", err)
	}

	return nil
}

func getRepoHTTPref(clones []Clone) string {
	for _, clone := range clones {
		if clone.Name == "http" {
			return clone.Href
		}
	}
	return "no http ref found"
}

func (c *OrgRepositoriesClient) reconcileRepository(ctx context.Context, actual gitprovider.UserRepository, req gitprovider.RepositoryInfo) (bool, error) {
	actionTaken := false

	// If the desired matches the actual state, just return the actual state
	new := actual.Get()
	if req.Equals(new) {
		return actionTaken, nil
	}
	// Populate the desired state to the current-actual object
	err := actual.Set(req)
	if err != nil {
		return actionTaken, err
	}

	projectKey, repoSlug := getStashRefs(actual.Repository())
	// Apply the desired state by running Update
	repo := actual.APIObject().(*Repository)
	if *req.DefaultBranch != "" && repo.DefaultBranch != *req.DefaultBranch {
		_, err = update(ctx, c.client, projectKey, repoSlug, repo, *req.DefaultBranch)
	} else {
		_, err = update(ctx, c.client, projectKey, repoSlug, repo, "")
	}

	if err != nil {
		return actionTaken, err
	}

	actionTaken = true
	return actionTaken, nil
}

func toCreateOpts(opts ...gitprovider.RepositoryReconcileOption) []gitprovider.RepositoryCreateOption {
	// Convert RepositoryReconcileOption => RepositoryCreateOption
	createOpts := make([]gitprovider.RepositoryCreateOption, 0, len(opts))
	for _, opt := range opts {
		createOpts = append(createOpts, opt)
	}
	return createOpts
}

// validateOrgRepositoryRef makes sure the OrgRepositoryRef is valid for GitHub's usage.
func validateOrgRepositoryRef(ref gitprovider.OrgRepositoryRef, expectedDomain string) error {
	// Make sure the RepositoryRef fields are valid
	if err := validation.ValidateTargets("OrgRepositoryRef", ref); err != nil {
		return err
	}
	// Make sure the type is valid, and domain is expected
	return validateIdentityFields(ref, expectedDomain)
}

func getStashRefs(ref gitprovider.RepositoryRef) (string, string) {
	var repoSlug string
	if slugger, ok := ref.(gitprovider.Slugger); ok {
		repoSlug = slugger.Slug()
	} else {
		repoSlug = ref.GetRepository()
	}

	var projectKey string
	if keyer, ok := ref.(gitprovider.Keyer); ok {
		projectKey = keyer.Key()
	} else {
		projectKey = ref.GetIdentity()
	}

	return projectKey, repoSlug
}

// validateRepositoryAPI validates the apiObj received from the server, to make sure that it is
// valid for our use.
func validateRepositoryAPI(apiObj *Repository) error {
	return validateAPIObject("Stash.Repository", func(validator validation.Validator) {
		// Make sure name is set
		if apiObj.Name == "" {
			validator.Required("Name")
		}
		// Make sure slug is set
		if apiObj.Slug == "" {
			validator.Required("Slug")
		}
	})
}
