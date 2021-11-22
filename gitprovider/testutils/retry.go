/*
Copyright 2020 The Flux authors

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

package testutils

import (
	"fmt"
	"os"
	"time"
)

// RetryI is an interface for retry operations
type RetryI interface {
	IsRetryable(err error, opDesc string) bool
	SetTimeout(timeout time.Duration)
	SetInterval(interval time.Duration)
	SetBackoff(backoff time.Duration)
	SetRetries(retries int)
	Timeout() time.Duration
	Interval() time.Duration
	Backoff() time.Duration
	Retries() int
}

// RetryOp is a retry operation
type RetryOp struct {
	RetryI
	timeout  time.Duration
	interval time.Duration
	backoff  time.Duration
	retries  int
	counter  int
}

// Timeout returns the timeout for the retry operation
func (r RetryOp) Timeout() time.Duration {
	return r.timeout
}

// Interval returns the interval for the retry operation
func (r RetryOp) Interval() time.Duration {
	return r.interval
}

// Backoff returns the backoff for the retry operation
func (r RetryOp) Backoff() time.Duration {
	return r.backoff
}

// Retries returns the number of retries for the retry operation
func (r RetryOp) Retries() int {
	return r.retries
}

// SetTimeout sets the timeout for the retry operation
func (r RetryOp) SetTimeout(timeout time.Duration) {
	r.timeout = timeout
}

// SetInterval sets the interval for the retry operation
func (r RetryOp) SetInterval(interval time.Duration) {
	r.interval = interval
}

// SetBackoff sets the backoff for the retry operation
func (r RetryOp) SetBackoff(backoff time.Duration) {
	r.backoff = backoff
}

// SetRetries sets the number of retries for the retry operation
func (r RetryOp) SetRetries(retries int) {
	r.retries = retries
}

// IsRetryable returns true if the operation is retryable
func (r RetryOp) IsRetryable(err error, opDesc string) bool {
	if err == nil {
		return true
	}

	fmt.Fprintf(os.Stderr, "%s, failed, error: %s\n", opDesc, err)
	if r.counter >= r.retries {
		time.Sleep(r.backoff)
		r.counter = 0
	}
	r.counter++
	return false
}

// NewRetry returns a new retry operation
func NewRetry() RetryI {
	r := RetryOp{
		retries:  10,
		counter:  0,
		timeout:  time.Second * 60,
		interval: time.Second,
		backoff:  time.Second * 10,
	}
	return r
}
