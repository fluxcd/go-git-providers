/*
Copyright 2026 The Flux CD contributors.

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

package github

import (
	"reflect"
	"testing"
)

func Test_gitRefCandidates(t *testing.T) {
	tests := []struct {
		name string
		ref  string
		want []string
	}{
		{
			name: "branch name",
			ref:  "main",
			want: []string{"heads/main", "tags/main"},
		},
		{
			name: "heads prefix",
			ref:  "heads/feature",
			want: []string{"heads/feature"},
		},
		{
			name: "tags prefix",
			ref:  "tags/v1.0.0",
			want: []string{"tags/v1.0.0"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := gitRefCandidates(tt.ref)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("gitRefCandidates(%q) = %v, want %v", tt.ref, got, tt.want)
			}
		})
	}
}
