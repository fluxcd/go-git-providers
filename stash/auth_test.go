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
	"testing"

	"github.com/fluxcd/go-git-providers/gitprovider"
)

func Test_DomainVariations(t *testing.T) {
	tests := []struct {
		name         string
		opts         gitprovider.ClientOption
		want         string
		expectedErrs []error
	}{
		{
			name: "custom domain without protocol",
			opts: gitprovider.WithDomain("stash.stashtestserver.link"),
			want: "https://stash.stashtestserver.link",
		},
		{
			name: "custom domain with port",
			opts: gitprovider.WithDomain("stash.stashtestserver.link:7990"),
			want: "https://stash.stashtestserver.link:7990",
		},
		{
			name: "custom domain with https protocol",
			opts: gitprovider.WithDomain("https://stash.stashtestserver.link:7990"),
			want: "https://stash.stashtestserver.link:7990",
		},
		{
			name: "custom domain with http protocol",
			opts: gitprovider.WithDomain("http://stash.stashtestserver.link:7990"),
			want: "http://stash.stashtestserver.link:7990",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c2, _ := NewStashClient("user1", "token", tt.opts)
			assertEqual(t, tt.want, c2.SupportedDomain())
		})
	}
}

func assertEqual(t *testing.T, a interface{}, b interface{}) {
	if a != b {
		t.Fatalf("%s != %s", a, b)
	}
}
