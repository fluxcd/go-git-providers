//go:build e2e

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

package gitea

import (
	"net/url"
	"os"
	"regexp"
	"testing"

	"github.com/fluxcd/go-git-providers/gitprovider"
)

func Test_DomainVariations(t *testing.T) {

	giteaBaseUrl = "http://try.gitea.io"
	if giteaBaseUrlVar := os.Getenv("GITEA_BASE_URL"); giteaBaseUrlVar != "" {
		giteaBaseUrl = giteaBaseUrlVar
	}

	u, err := url.Parse(giteaBaseUrl)
	if err != nil {
		t.Fatalf("failed parsing base URL %q: %s", giteaBaseUrl, err)
	}

	tests := []struct {
		name               string
		opts               gitprovider.ClientOption
		want               string
		expectedErrPattern string
	}{
		{
			name:               "custom domain without protocol uses HTTPS by default",
			opts:               gitprovider.WithDomain(u.Host),
			expectedErrPattern: "http: server gave HTTP response to HTTPS client",
		},
		{
			name: "custom domain with scheme",
			opts: gitprovider.WithDomain(giteaBaseUrl),
			want: giteaBaseUrl,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c1, err := NewClient("token", tt.opts)
			if err != nil {
				if tt.expectedErrPattern == "" {
					t.Fatalf("unexpected error: %s", err)
				}
				m, mErr := regexp.MatchString(tt.expectedErrPattern, err.Error())
				if mErr != nil {
					t.Fatalf("unexpected error from matching error: %s", mErr)
				}
				if !m {
					t.Fatalf("unexpected error %q; expected %q", err, tt.expectedErrPattern)
				}
				return // all assertions passed
			} else if tt.expectedErrPattern != "" {
				t.Fatalf("expected error %q but got none", tt.expectedErrPattern)
			}

			assertEqual(t, tt.want, c1.SupportedDomain())
		})
	}
}

func assertEqual(t *testing.T, a interface{}, b interface{}) {
	if a != b {
		t.Fatalf("%s != %s", a, b)
	}
}
