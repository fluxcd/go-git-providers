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
	"fmt"

	"github.com/fluxcd/go-git-providers/gitprovider"
	"github.com/fluxcd/go-git-providers/validation"
)

// OrganizationsClient implements the gitprovider.OrganizationsClient interface.
var _ gitprovider.OrganizationsClient = &OrganizationsClient{}

// OrganizationsClient operates on the projects the user has access to.
type OrganizationsClient struct {
	*clientContext
}

// Get a specific organization the user has access to.
// ErrNotFound is returned if the resource does not exist.
func (c *OrganizationsClient) Get(ctx context.Context, ref gitprovider.OrganizationRef) (gitprovider.Organization, error) {
	// Make sure the OrganizationRef is valid
	if err := validateOrganizationRef(ref, c.host); err != nil {
		return nil, err
	}
	apiObj, err := c.client.Projects.Get(ctx, ref.Organization)
	if err != nil {
		return nil, fmt.Errorf("failed to get organization %q: %w", ref.Organization, err)
	}

	// Validate the API objects

	if err := validateProjectAPI(apiObj); err != nil {
		return nil, err
	}

	ref.Key = apiObj.Key

	return newOrganization(c.clientContext, apiObj, ref), nil
}

// List all the organizations the specific user has access to.
// List returns all available organizations, using multiple paginated requests if needed.
func (c *OrganizationsClient) List(ctx context.Context) ([]gitprovider.Organization, error) {
	// Retrieve all projects
	apiObjs, err := c.client.Projects.All(ctx, c.maxPages)
	if err != nil {
		return nil, fmt.Errorf("failed to list organizations: %w", err)
	}

	// Validate the API objects
	for _, apiObj := range apiObjs {
		if err := validateProjectAPI(apiObj); err != nil {
			return nil, err
		}
	}

	projects := make([]gitprovider.Organization, len(apiObjs))
	for i, apiObj := range apiObjs {
		ref := gitprovider.OrganizationRef{
			Domain:       c.host,
			Organization: apiObj.Name,
			Key:          apiObj.Key,
		}
		projects[i] = newOrganization(c.clientContext, apiObj, ref)
	}

	return projects, nil
}

// Children returns the immediate child-organizations for the specific OrganizationRef o.
// The OrganizationRef may point to any existing sub-organization.
// Children returns all available organizations, using multiple paginated requests if needed.
func (c *OrganizationsClient) Children(ctx context.Context, ref gitprovider.OrganizationRef) ([]gitprovider.Organization, error) {
	return nil, gitprovider.ErrNoProviderSupport
}

// validateOrganizationRef makes sure the OrganizationRef is valid for stash usage.
func validateOrganizationRef(ref gitprovider.OrganizationRef, expectedDomain string) error {
	// Make sure the OrganizationRef fields are valid
	if err := validation.ValidateTargets("OrganizationRef", ref); err != nil {
		return err
	}
	// Make sure the type is valid, and domain is expected
	return validateIdentityFields(ref, expectedDomain)
}

// validateIdentityFields makes sure the type of the IdentityRef is supported, and the domain is as expected.
func validateIdentityFields(ref gitprovider.IdentityRef, expectedDomain string) error {
	// Make sure the expected domain is used
	if ref.GetDomain() != expectedDomain {
		return fmt.Errorf("domain %q not supported by this client: %w", ref.GetDomain(), gitprovider.ErrDomainUnsupported)
	}
	// Make sure the right type of identityref is used
	switch ref.GetType() {
	case gitprovider.IdentityTypeOrganization, gitprovider.IdentityTypeUser:
		return nil
	case gitprovider.IdentityTypeSuborganization:
		return fmt.Errorf("stash doesn't support sub-organizations: %w", gitprovider.ErrNoProviderSupport)
	}
	return fmt.Errorf("invalid identity type: %v: %w", ref.GetType(), gitprovider.ErrInvalidArgument)
}

func validateProjectAPI(apiObj *Project) error {
	return validateAPIObject("Stash.Project", func(validator validation.Validator) {
		// Make sure name is set
		if apiObj.Name == "" {
			validator.Required("Name")
		}

		// Make sure key is set
		if apiObj.Key == "" {
			validator.Required("Key")
		}
	})
}
