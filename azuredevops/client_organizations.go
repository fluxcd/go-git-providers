package azuredevops

import (
	"context"
	"github.com/fluxcd/go-git-providers/gitprovider"
)

// OrganizationsClient implements the gitprovider.OrganizationsClient interface.
var _ gitprovider.OrganizationsClient = &OrganizationsClient{}

// OrganizationsClient operates on the groups the user has access to.
type OrganizationsClient struct {
	*clientContext
}

func (c *OrganizationsClient) Get(ctx context.Context, o gitprovider.OrganizationRef) (gitprovider.Organization, error) {
	//TODO implement me
	panic("implement me")
}

// List all the projects the specific user has access to.
// List returns all available projects, using multiple paginated requests if needed.

func (c *OrganizationsClient) List(ctx context.Context) ([]gitprovider.Organization, error) {
	apiObjs, err := c.c.ListProjects(ctx)
	if err != nil {
		return nil, err
	}

	projects := make([]gitprovider.Organization, len(apiObjs.Value))
	for i, apiObj := range apiObjs.Value {
		ref := gitprovider.OrganizationRef{
			Domain:       *apiObj.Url,
			Organization: *apiObj.Name,
		}

		//ref.SetKey(base64.RawURLEncoding.EncodeToString(apiObj.Id))
		projects[i] = newOrganization(c.clientContext, apiObj, ref)
	}
	return projects, nil

}

// Children returns the immediate child-organizations for the specific OrganizationRef o.
// The OrganizationRef may point to any existing sub-organization.
// Children returns all available organizations, using multiple paginated requests if needed.
func (c *OrganizationsClient) Children(_ context.Context, _ gitprovider.OrganizationRef) ([]gitprovider.Organization, error) {
	return nil, gitprovider.ErrNoProviderSupport
}
