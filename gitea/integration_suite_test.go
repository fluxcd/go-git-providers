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

package gitea

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"code.gitea.io/sdk/gitea"
	"github.com/gregjones/httpcache"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/fluxcd/go-git-providers/gitprovider"
)

const (
	ghTokenFile = "/tmp/gitea-token"
	giteaDomain = "http://gitea.example.com"

	defaultDescription = "Foo description"
	// TODO: This will change
	defaultBranch = "main"
)

var (
	// customTransportImpl is a shared instance of a customTransport, allowing counting of cache hits.
	customTransportImpl *customTransport
	ctx                 context.Context = context.Background()
	c                   gitprovider.Client

	testOrgRepoName  string
	testUserRepoName string
	testOrgName      string = "fluxcd-testing"
	testUser         string = "fluxcd-gitprovider-bot"
)

func init() {
	// Call testing.Init() prior to tests.NewParams(), as otherwise -test.* will not be recognised. See also: https://golang.org/doc/go1.13#testing
	testing.Init()
	rand.Seed(time.Now().UnixNano())
}

func TestProvider(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Gitea Provider Suite")
}

func customTransportFactory(transport http.RoundTripper) http.RoundTripper {
	if customTransportImpl != nil {
		panic("didn't expect this function to be called twice")
	}
	customTransportImpl = &customTransport{
		transport:      transport,
		countCacheHits: false,
		cacheHits:      0,
		mux:            &sync.Mutex{},
	}
	return customTransportImpl
}

type customTransport struct {
	transport      http.RoundTripper
	countCacheHits bool
	cacheHits      int
	mux            *sync.Mutex
}

func (t *customTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	t.mux.Lock()
	defer t.mux.Unlock()

	resp, err := t.transport.RoundTrip(req)
	// If we should count, count all cache hits whenever found
	if t.countCacheHits {
		if _, ok := resp.Header[httpcache.XFromCache]; ok {
			t.cacheHits++
		}
	}
	return resp, err
}

func (t *customTransport) resetCounter() {
	t.mux.Lock()
	defer t.mux.Unlock()

	t.cacheHits = 0
}

func (t *customTransport) setCounter(state bool) {
	t.mux.Lock()
	defer t.mux.Unlock()

	t.countCacheHits = state
}

func (t *customTransport) getCacheHits() int {
	t.mux.Lock()
	defer t.mux.Unlock()

	return t.cacheHits
}

func (t *customTransport) countCacheHitsForFunc(fn func()) int {
	t.setCounter(true)
	t.resetCounter()
	fn()
	t.setCounter(false)
	return t.getCacheHits()
}

var _ = Describe("Gitea Provider", func() {
	BeforeSuite(func() {
		giteaToken := os.Getenv("GITEA_TOKEN")
		if len(giteaToken) == 0 {
			b, err := os.ReadFile(ghTokenFile)
			if token := string(b); err == nil && len(token) != 0 {
				giteaToken = token
			} else {
				Fail("couldn't acquire GITEA_TOKEN env variable")
			}
		}

		if orgName := os.Getenv("GIT_PROVIDER_ORGANIZATION"); len(orgName) != 0 {
			testOrgName = orgName
		}

		if gitProviderUser := os.Getenv("GIT_PROVIDER_USER"); len(gitProviderUser) != 0 {
			testUser = gitProviderUser
		}

		var err error
		c, err = NewClient(
			gitprovider.WithOAuth2Token(giteaToken),
			gitprovider.WithDomain(giteaDomain),
			gitprovider.WithDestructiveAPICalls(true),
			gitprovider.WithConditionalRequests(true),
			gitprovider.WithPreChainTransportHook(customTransportFactory),
		)
		Expect(err).ToNot(HaveOccurred())
	})

	cleanupOrgRepos := func(prefix string) {
		fmt.Fprintf(os.Stderr, "Deleting repos starting with %s in org: %s\n", prefix, testOrgName)
		repos, err := c.OrgRepositories().List(ctx, newOrgRef(testOrgName))
		Expect(err).ToNot(HaveOccurred())
		for _, repo := range repos {
			// Delete the test org repo used
			name := repo.Repository().GetRepository()
			if !strings.HasPrefix(name, prefix) {
				continue
			}
			fmt.Fprintf(os.Stderr, "Deleting the org repo: %s\n", name)
			repo.Delete(ctx)
			Expect(err).ToNot(HaveOccurred())
		}
	}

	cleanupUserRepos := func(prefix string) {
		fmt.Fprintf(os.Stderr, "Deleting repos starting with %s for user: %s\n", prefix, testUser)
		repos, err := c.UserRepositories().List(ctx, newUserRef(testUser))
		Expect(err).ToNot(HaveOccurred())
		fmt.Fprintf(os.Stderr, "repos, len: %d\n", len(repos))
		for _, repo := range repos {
			fmt.Fprintf(os.Stderr, "repo: %s\n", repo.Repository().GetRepository())
			name := repo.Repository().GetRepository()
			if !strings.HasPrefix(name, prefix) {
				fmt.Fprintf(os.Stderr, "Skipping the org repo: %s\n", name)
				continue
			}
			fmt.Fprintf(os.Stderr, "Deleting the org repo: %s\n", name)
			repo.Delete(ctx)
			Expect(err).ToNot(HaveOccurred())
		}
	}

	AfterSuite(func() {
		if os.Getenv("SKIP_CLEANUP") == "1" {
			return
		}
		// Don't do anything more if c wasn't created
		if c == nil {
			return
		}

		if len(os.Getenv("CLEANUP_ALL")) > 0 {
			defer cleanupOrgRepos("test-repo")
			defer cleanupUserRepos("test-repo")
		}

		// Delete the org test repo used
		orgRepo, err := c.OrgRepositories().Get(ctx, newOrgRepoRef(testOrgName, testOrgRepoName))
		if err != nil && len(os.Getenv("CLEANUP_ALL")) > 0 {
			fmt.Fprintf(os.Stderr, "failed to get repo: %s in org: %s, error: %s\n", testOrgRepoName, testOrgName, err)
			fmt.Fprintf(os.Stderr, "CLEANUP_ALL set so continuing\n")
		} else {
			Expect(err).ToNot(HaveOccurred())
			Expect(orgRepo.Delete(ctx)).ToNot(HaveOccurred())
		}
		// Delete the user test repo used
		userRepo, err := c.UserRepositories().Get(ctx, newUserRepoRef(testUser, testUserRepoName))
		Expect(err).ToNot(HaveOccurred())
		Expect(userRepo.Delete(ctx)).ToNot(HaveOccurred())
	})
})

func newOrgRef(organizationName string) gitprovider.OrganizationRef {
	return gitprovider.OrganizationRef{
		Domain:       giteaDomain,
		Organization: organizationName,
	}
}

func newOrgRepoRef(organizationName, repoName string) gitprovider.OrgRepositoryRef {
	return gitprovider.OrgRepositoryRef{
		OrganizationRef: newOrgRef(organizationName),
		RepositoryName:  repoName,
	}
}

func newUserRef(userLogin string) gitprovider.UserRef {
	return gitprovider.UserRef{
		Domain:    giteaDomain,
		UserLogin: userLogin,
	}
}

func newUserRepoRef(userLogin, repoName string) gitprovider.UserRepositoryRef {
	return gitprovider.UserRepositoryRef{
		UserRef:        newUserRef(userLogin),
		RepositoryName: repoName,
	}
}

func findUserRepo(repos []gitprovider.UserRepository, name string) gitprovider.UserRepository {
	if name == "" {
		return nil
	}
	for _, repo := range repos {
		if repo.Repository().GetRepository() == name {
			return repo
		}
	}
	return nil
}

func findOrgRepo(repos []gitprovider.OrgRepository, name string) gitprovider.OrgRepository {
	if name == "" {
		return nil
	}
	for _, repo := range repos {
		if repo.Repository().GetRepository() == name {
			return repo
		}
	}
	return nil
}

func findOrgTeam(ctx context.Context, tc gitprovider.TeamsClient, name string) gitprovider.Team {
	if name == "" {
		return nil
	}
	team, err := tc.Get(ctx, name)
	if err != nil {
		return nil
	}
	return team
}

func validateRepo(repo gitprovider.OrgRepository, expectedRepoRef gitprovider.RepositoryRef) {
	info := repo.Get()
	// Expect certain fields to be set
	Expect(repo.Repository()).To(Equal(expectedRepoRef))
	Expect(*info.Description).To(Equal(defaultDescription))
	Expect(*info.Visibility).To(Equal(gitprovider.RepositoryVisibilityPrivate))
	Expect(*info.DefaultBranch).To(Equal(defaultBranch))
	// Expect high-level fields to match their underlying data
	internal := repo.APIObject().(*gitea.Repository)
	Expect(repo.Repository().GetRepository()).To(Equal(internal.Name))
	Expect(repo.Repository().GetIdentity()).To(Equal(internal.Owner.UserName))
	Expect(*info.Description).To(Equal(internal.Description))
	internalPrivatestr := gitea.VisibleTypePublic
	if internal.Private {
		internalPrivatestr = gitea.VisibleTypePrivate
	}
	Expect(string(*info.Visibility)).To(Equal(string(internalPrivatestr)))
	Expect(*info.DefaultBranch).To(Equal(internal.DefaultBranch))
}
