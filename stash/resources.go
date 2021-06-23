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
	"net/http"
)

const (
	contextKey = "context"
	filterKey  = "filter"
)

var (
	stashURIprefix = "/rest/api/1.0"
	stashURIkeys   = "/rest/keys/1.0"
)

type SessionInfo struct {
	UserID    string `json:"userID,omitempty"`
	UserName  string `json:"userName,omitempty"`
	SessionID string `json:"sessionID,omitempty"`
	RequestID string `json:"requestID,omitempty"`
}

func (s *SessionInfo) setSessionInfo(resp *http.Response) {
	s.UserID = resp.Header.Get("X-Auserid")
	s.UserName = resp.Header.Get("X-Ausername")
	s.SessionID = resp.Header.Get("X-Asessionid")
	s.RequestID = resp.Header.Get("X-Arequestid")
}

func (s *SessionInfo) copySessionInfo(p *SessionInfo) {
	s.UserID = p.UserID
	s.UserName = p.UserName
	s.SessionID = p.SessionID
	s.RequestID = p.RequestID
}

type StashPaging interface {
	IsLast() bool
	SetNext(next int)
	SetStart(start int)
	SetLimit(limit int)
	Next() int
	Start() int
	Limit() int
}

// Paging is the paging information.
type Paging struct {
	StashPaging
	IsLastPage    bool  `json:"isLastPage,omitempty"`
	Limit         int64 `json:"limit,omitempty"`
	Size          int64 `json:"size,omitempty"`
	Start         int64 `json:"start,omitempty"`
	NextPageStart int64 `json:"nextPageStart,omitempty"`
}

func (p *Paging) IsLast() bool {
	return p.IsLastPage
}

type ListOptions struct {
	Start int64
	Limit int64
}

type Self struct {
	Href string `json:"href,omitempty"`
}

type Clone struct {
	Href string `json:"href,omitempty"`
	Name string `json:"name,omitempty"`
}

type Links struct {
	Self  []Self  `json:"self,omitempty"`
	Clone []Clone `json:"clone,omitempty"`
}
