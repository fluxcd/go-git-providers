package gitlab_test

// func ExampleOrgRepositoriesClient_Get() {
// 	// Create a new client
// 	ctx := context.Background()
// 	c, err := gitlab.NewClientFromPAT(os.Getenv("GITLAB_TOKEN"))
// 	checkErr(err)

// 	// Parse the URL into an OrgRepositoryRef
// 	ref, err := gitprovider.ParseOrgRepositoryURL("https://github.com/fluxcd/flux")
// 	checkErr(err)

// 	// Get public information about the flux repository.
// 	repo, err := c.OrgRepositories().Get(ctx, *ref)
// 	checkErr(err)

// 	// Use .Get() to aquire a high-level gitprovider.OrganizationInfo struct
// 	repoInfo := repo.Get()
// 	// Cast the internal object to a *gogithub.Repository to access custom data
// 	internalRepo := repo.APIObject().(*gogitlab.Project)

// 	fmt.Printf("Description: %s. Homepage: %s", *repoInfo.Description, internalRepo.HTTPURLToRepo)
// 	// Output: Description: The GitOps Kubernetes operator. Homepage: https://docs.fluxcd.io
// }
