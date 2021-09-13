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

package stash

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
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
	stashTokenFile     = "/tmp/stash.token"
	defaultDescription = "Foo description"
)

var (
	// customTransportImpl is a shared instance of a customTransport, allowing counting of cache hits.
	customTransportImpl *customTransport
	stashDomain         = "stash.example.com"
	defaultBranch       = "main"
	testOrgName         string
	testUserName        string
	testTeamName        string
	client              gitprovider.Client
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
	  "initialFields": {"name": "stash integration test"},
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
	RunSpecs(t, "Stash Provider Suite")
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

var _ = Describe("Stash Provider", func() {

	BeforeSuite(func() {

		log := setupLogr()
		log.V(1).Info("logger construction succeeded")

		stashToken := os.Getenv("STASH_TOKEN")
		if stashToken == "" {
			b, err := ioutil.ReadFile(stashTokenFile)
			if token := string(b); err == nil && token != "" {
				stashToken = token
			} else {
				Fail("couldn't acquire STASH_TOKEN env variable")
			}
		}

		if stashDomainVar := os.Getenv("STASH_DOMAIN"); stashDomainVar != "" {
			stashDomain = stashDomainVar
		}

		if orgName := os.Getenv("GIT_PROVIDER_ORGANIZATION"); orgName != "" {
			testOrgName = orgName
		}

		if gitProviderUser := os.Getenv("GIT_PROVIDER_USER"); gitProviderUser != "" {
			testUserName = gitProviderUser
		}

		if teamName := os.Getenv("STASH_TEST_TEAM_NAME"); teamName != "" {
			testTeamName = teamName
		}

		var err error
		client, err = NewClient(
			stashToken,
			gitprovider.WithDomain(stashDomain),
			gitprovider.WithDestructiveAPICalls(true),
			gitprovider.WithConditionalRequests(true),
			gitprovider.WithPreChainTransportHook(customTransportFactory),
			gitprovider.WithLogger(&log),
		)
		Expect(err).ToNot(HaveOccurred())
	})
})

func newOrgRef(organizationName string) gitprovider.OrganizationRef {
	return gitprovider.OrganizationRef{
		Domain:       stashDomain,
		Organization: organizationName,
	}
}

func newUserRef(userLogin string) gitprovider.UserRef {
	return gitprovider.UserRef{
		Domain:    stashDomain,
		UserLogin: userLogin,
	}
}
