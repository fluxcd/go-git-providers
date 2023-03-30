/*
Copyright 2023 The Flux authors

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

package azuredevops

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

	"encoding/json"

	"github.com/gregjones/httpcache"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"go.uber.org/zap"

	"github.com/fluxcd/go-git-providers/gitprovider"
)

const (
	azureTokenFile     = "/tmp/azure.token"
	defaultDescription = "Foo description"
)

var (
	// customTransportImpl is a shared instance of a customTransport, allowing counting of cache hits.
	customTransportImpl *customTransport
	azureDomain         = "dev.azure.com"
	defaultBranch       = "main"
	testOrgName         = "souleb"
	testTeamName        = "fluxcd-test-team"
	// placeholders, will be randomized and created.
	testSharedOrgRepoName string
	testOrgRepoName       string
	testRepoName          string
	client                gitprovider.Client
)

func init() {
	// Call testing.Init() prior to tests.NewParams(), as otherwise -test.* will not be recognised. See also: https://golang.org/doc/go1.13#testing
	testing.Init()
	rand.Seed(time.Now().UnixNano())
}

func setupLogr() logr.Logger {
	zapLog, err := setupLog()
	if err != nil {
		panic(fmt.Sprintf("who watches the watchmen (%v)?", err))
	}
	return zapr.NewLogger(zapLog)
}

func setupLog() (*zap.Logger, error) {
	rawJSON := []byte(`{
	  "level": "info",
	  "encoding": "json",
	  "outputPaths": ["stdout"],
	  "errorOutputPaths": ["stderr"],
	  "initialFields": {"name": "azure integration test"},
	  "encoderConfig": {
	    "messageKey": "message",
	    "levelKey": "level",
	    "levelEncoder": "lowercase"
	  }
	}`)

	var cfg zap.Config
	if err := json.Unmarshal(rawJSON, &cfg); err != nil {
		return nil, err
	}
	logger, err := cfg.Build()
	if err != nil {
		return nil, err
	}
	defer logger.Sync()
	return logger, nil
}

func TestProvider(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "azure Provider Suite")
}

func customTransportFactory(transport http.RoundTripper) http.RoundTripper {
	if customTransportImpl != nil {
		panic("didn't expect this function to be called twice")
	}
	customTransportImpl = &customTransport{
		transport:      transport,
		countCacheHits: false,
		cacheHits:      0,
		mu:             &sync.Mutex{},
	}
	return customTransportImpl
}

type customTransport struct {
	transport      http.RoundTripper
	countCacheHits bool
	cacheHits      int
	mu             *sync.Mutex
}

func (t *customTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

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
	t.mu.Lock()
	defer t.mu.Unlock()

	t.cacheHits = 0
}

func (t *customTransport) setCounter(state bool) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.countCacheHits = state
}

func (t *customTransport) getCacheHits() int {
	t.mu.Lock()
	defer t.mu.Unlock()

	return t.cacheHits
}

func (t *customTransport) countCacheHitsForFunc(fn func()) int {
	t.setCounter(true)
	t.resetCounter()
	fn()
	t.setCounter(false)
	return t.getCacheHits()
}

var _ = Describe("azure Provider", func() {
	var (
		ctx context.Context = context.Background()
	)

	BeforeSuite(func() {

		log := setupLogr()
		log.V(1).Info("logger construction succeeded")

		azureToken := os.Getenv("AZURE_TOKEN")
		if azureToken == "" {
			b, err := os.ReadFile(azureTokenFile)
			if token := string(b); err == nil && token != "" {
				azureToken = token
			} else {
				Fail("couldn't acquire AZURE_TOKEN env variable")
			}
		}

		if azureDomainVar := os.Getenv("AZURE_DOMAIN"); azureDomainVar != "" {
			azureDomain = azureDomainVar
		}

		if orgName := os.Getenv("GIT_PROVIDER_ORGANIZATION"); orgName != "" {
			testOrgName = orgName
		}

		if teamName := os.Getenv("AZURE_TEST_TEAM_NAME"); teamName != "" {
			testTeamName = teamName
		}

		var err error
		client, err = NewClient(
			azureToken,
			ctx,
			gitprovider.WithDomain(azureDomain),
			gitprovider.WithDestructiveAPICalls(true),
			gitprovider.WithConditionalRequests(true),
			// It seems not to be possible with azure devops to provide our own http Client
			gitprovider.WithPreChainTransportHook(customTransportFactory),
			gitprovider.WithLogger(&log),
		)
		Expect(err).ToNot(HaveOccurred())
	})

	AfterSuite(func() {
		if os.Getenv("SKIP_CLEANUP") == "1" {
			return
		}
		// Don't do anything more if client wasn't created
		if client == nil {
			return
		}

		defer cleanupOrgRepos(ctx, "test-org-repo")
		defer cleanupOrgRepos(ctx, "test-shared-org-repo")
	})
})

func newOrgRef(organizationName string) gitprovider.OrganizationRef {
	return gitprovider.OrganizationRef{
		Domain:       azureDomain,
		Organization: organizationName,
	}
}

func cleanupOrgRepos(ctx context.Context, prefix string) {
	fmt.Printf("Deleting repos starting with %s in org: %s\n", prefix, testOrgName)
	// Get the test organization
	orgRef := newOrgRef(testOrgName)
	testOrg, err := client.Organizations().Get(ctx, orgRef)
	Expect(err).ToNot(HaveOccurred())
	repos, err := client.OrgRepositories().List(ctx, testOrg.Organization())
	Expect(err).ToNot(HaveOccurred())
	for _, repo := range repos {
		// Delete the test org repo used
		name := repo.Repository().GetRepository()
		slug := repo.Repository().(gitprovider.Slugger).Slug()
		key := repo.Repository().(gitprovider.Keyer).Key()
		if !strings.HasPrefix(name, prefix) {
			continue
		}
		fmt.Printf("Deleting the %s organization's repository: %s with slug %s\n", key, name, slug)
		Expect(repo.Delete(ctx)).To(Succeed())
	}
}
