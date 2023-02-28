/*
 *  Copyright 2022 VMware, Inc.
 *
 *  Licensed under the Apache License, Version 2.0 (the "License");
 *  you may not use this file except in compliance with the License.
 *  You may obtain a copy of the License at
 *  http://www.apache.org/licenses/LICENSE-2.0
 *  Unless required by applicable law or agreed to in writing, software
 *  distributed under the License is distributed on an "AS IS" BASIS,
 *  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *  See the License for the specific language governing permissions and
 *  limitations under the License.
 */

package testsupport

import (
	"net/http"

	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/cf/fakes"
)

type StubClient struct {
	*fakes.FakeClient
	DoFunc          func(req *http.Request) (*http.Response, error)
	DoWithRetryFunc func(f func() error) error
	HTTPClientFunc  func() *http.Client
	TargetFunc      func() string
	GetFunc         func(url string) ([]byte, error)
}

func (s StubClient) Get(url string) ([]byte, error) {
	return s.GetFunc(url)
}

func (s StubClient) Do(req *http.Request) (*http.Response, error) {
	return s.DoFunc(req)
}

func (s StubClient) DoWithRetry(f func() error) error {
	return s.DoWithRetryFunc(f)
}

func (s StubClient) HTTPClient() *http.Client {
	return s.HTTPClientFunc()
}

func (s StubClient) Target() string {
	return s.TargetFunc()
}
