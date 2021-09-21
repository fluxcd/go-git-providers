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
	"net/http"
)

const (
	contextKey     = "context"
	filterKey      = "filter"
	stashURIprefix = "/rest/api/1.0"
	stashURIkeys   = "/rest/keys/1.0"
)

// Session keeps a record of a request for a given user.
type Session struct {
	// UserID is the ID of the user making the request.
	UserID string `json:"userID,omitempty"`
	// UserName is the name of the user making the request.
	UserName string `json:"userName,omitempty"`
	// SessionID is the ID of the session.
	SessionID string `json:"sessionID,omitempty"`
	// RequestID is the ID of the request.
	RequestID string `json:"requestID,omitempty"`
}

func (s *Session) set(resp *http.Response) {
	s.UserID = resp.Header.Get("X-Auserid")
	s.UserName = resp.Header.Get("X-Ausername")
	s.SessionID = resp.Header.Get("X-Asessionid")
	s.RequestID = resp.Header.Get("X-Arequestid")
}

func (s *Session) copy(p *Session) {
	s.UserID = p.UserID
	s.UserName = p.UserName
	s.SessionID = p.SessionID
	s.RequestID = p.RequestID
}

// Paging is the paging information.
type Paging struct {
	// IsLastPage indicates whether another page of items exists.
	IsLastPage bool `json:"isLastPage,omitempty"`
	// Limit indicates how many results to return per page.
	Limit int64 `json:"limit,omitempty"`
	// Size indicates the total number of results..
	Size int64 `json:"size,omitempty"`
	// Start indicates which item should be used as the first item in the page of results.
	Start int64 `json:"start,omitempty"`
	// NexPageStart must be used by the client as the start parameter on the next request.
	// Identifiers of adjacent objects in a page may not be contiguous,
	// so the start of the next page is not necessarily the start of the last page plus the last page's size.
	// Always use nextPageStart to avoid unexpected results from a paged API.
	NextPageStart int64 `json:"nextPageStart,omitempty"`
}

// IsLast returns true if the paging information indicates that there are no more pages.
func (p *Paging) IsLast() bool {
	return p.IsLastPage
}

// PagingOptions is the options for paging.
type PagingOptions struct {
	// Start indicates which item should be used as the first item in the page of results.
	Start int64
	// Limit indicates how many results to return per page.
	Limit int64
}

// Self indicates the hyperlink to a REST resource.
type Self struct {
	Href string `json:"href,omitempty"`
}

// Clone is a hyperlink to another REST resource.
type Clone struct {
	// Href is the hyperlink to the resource.
	Href string `json:"href,omitempty"`
	// Name is the name of the resource.
	Name string `json:"name,omitempty"`
}

// Links is a set of hyperlinks that link to other related resources.
type Links struct {
	// Self is the hyperlink to the resource.
	Self []Self `json:"self,omitempty"`
	// Clone is a set of hyperlinks to other REST resources.
	Clone []Clone `json:"clone,omitempty"`
}
