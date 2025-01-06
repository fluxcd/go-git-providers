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

package gitea_test

import (
	"context"
	"fmt"
	"os"

	gogitea "code.gitea.io/sdk/gitea"
	"github.com/fluxcd/go-git-providers/gitea"
	"github.com/fluxcd/go-git-providers/gitprovider"
)

func ExampleOrgRepositoriesClient_Get() {
	// Create a new client
	ctx := context.Background()
	c, err := gitea.NewClient(os.Getenv("GITEA_ACCESS_TOKEN"), gitprovider.WithDomain(gitea.DefaultDomain))
	checkErr(err)

	// Parse the URL into an OrgRepositoryRef
	ref, err := gitprovider.ParseOrgRepositoryURL("https://gitea.com/gitea/go-sdk")
	checkErr(err)

	// Get public information about the flux repository.
	repo, err := c.OrgRepositories().Get(ctx, *ref)
	checkErr(err)

	// Use .Get() to aquire a high-level gitprovider.OrganizationInfo struct
	repoInfo := repo.Get()
	// Cast the internal object to a *gogitea.Repository to access custom data
	internalRepo := repo.APIObject().(*gogitea.Repository)

	fmt.Printf("Description: %s. Homepage: %s", *repoInfo.Description, internalRepo.HTMLURL)
}
