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

package cache

import (
	"net/http"

	"github.com/gregjones/httpcache"
)

// TODO: Implement an unit test for this package.

// NewHTTPCacheTransport is a gitprovider.ChainableRoundTripperFunc which adds
// HTTP Conditional Requests caching for the backend, if the server supports it.
func NewHTTPCacheTransport(in http.RoundTripper) http.RoundTripper {
	// Create a new httpcache high-level Transport
	t := httpcache.NewMemoryCacheTransport()
	// Configure the httpcache Transport to use in as its underlying Transport.
	// If in is nil, http.DefaultTransport will be used.
	t.Transport = in
	// Set "out" to use a slightly custom variant of the httpcache Transport
	// (with more aggressive cache invalidation)
	return &cacheRoundtripper{Transport: t}
}

// cacheRoundtripper is a slight wrapper around *httpcache.Transport that automatically
// invalidates the cache on non-GET/HEAD requests, and non-"200 OK" responses.
type cacheRoundtripper struct {
	Transport *httpcache.Transport
}

// This function follows the same logic as in github.com/gregjones/httpcache to be able
// to implement our custom roundtripper logic below.
func cacheKey(req *http.Request) string {
	if req.Method == http.MethodGet {
		return req.URL.String()
	}
	return req.Method + " " + req.URL.String()
}

// RoundTrip calls the underlying RoundTrip (using the cache), but invalidates the cache on
// non GET/HEAD requests and non-"200 OK" responses.
func (r *cacheRoundtripper) RoundTrip(req *http.Request) (*http.Response, error) {
	// These two statements are the same as in github.com/gregjones/httpcache Transport.RoundTrip
	// to be able to implement our custom roundtripper below
	cacheKey := cacheKey(req)
	cacheable := (req.Method == "GET" || req.Method == "HEAD") && req.Header.Get("range") == ""

	// If the object isn't a GET or HEAD request, also invalidate the cache of the GET URL
	// as this action will modify the underlying resource (e.g. DELETE/POST/PATCH)
	if !cacheable {
		r.Transport.Cache.Delete(req.URL.String())
	}
	// Call the underlying roundtrip
	resp, err := r.Transport.RoundTrip(req)
	// Don't cache anything but "200 OK" requests
	if resp == nil || resp.StatusCode != http.StatusOK {
		r.Transport.Cache.Delete(cacheKey)
	}
	return resp, err
}
