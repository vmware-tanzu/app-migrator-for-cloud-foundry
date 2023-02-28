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

package commands

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	ctxfakes "github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/context/fakes"

	"github.com/cloudfoundry-community/go-cfclient"
	"github.com/stretchr/testify/assert"
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/cache"
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/cf/fakes"
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/metadata"
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/report"
	. "github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/testsupport"

	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/context"
)

func TestImportOrg_Run(t *testing.T) {
	type fields struct {
		Org string
	}
	type args struct {
		ctx *context.Context
	}
	pwd, _ := os.Getwd()
	tests := []struct {
		name               string
		fields             fields
		args               args
		wantErr            bool
		failedAppCount     int
		successfulAppCount int
	}{
		{
			name:   "returns no error",
			fields: fields{Org: "my_org"},
			args: args{
				ctx: &context.Context{
					ExportDir: filepath.Join(pwd, "testdata/apps"),
					Metadata:  metadata.NewMetadata(),
					Summary:   report.NewSummary(&bytes.Buffer{}),
					SpaceImporter: &ctxfakes.FakeSpaceImporter{
						ImportSpaceStub: func(ctx *context.Context, processFunc context.ProcessFunc, strings []string) (<-chan context.ProcessResult, error) {
							results := make(chan context.ProcessResult, 1)
							defer close(results)
							results <- context.ProcessResult{Value: "my_app"}
							ctx.Summary.AddSuccessfulApp("my_org", "my_space", "my_app")
							return results, nil
						},
					},
					ImportCFClient: StubClient{
						FakeClient: &fakes.FakeClient{
							GetOrgByNameStub: func(s string) (cfclient.Org, error) {
								return cfclient.Org{
									Name: "my_org",
								}, nil
							},
							GetSpaceByNameStub: func(string, string) (cfclient.Space, error) {
								return cfclient.Space{
									Name: "my_space",
								}, nil
							},
							ListUserProvidedServiceInstancesByQueryStub: func(url.Values) ([]cfclient.UserProvidedServiceInstance, error) {
								return []cfclient.UserProvidedServiceInstance{{
									Name: "name-1508",
								}}, nil
							},
							DoRequestStub: func(*cfclient.Request) (*http.Response, error) {
								data, err := ioutil.ReadFile("testdata/v3droplets.json")
								assert.NoError(t, err)
								stringReader := strings.NewReader(string(data))
								resp := &http.Response{
									Body: io.NopCloser(stringReader),
								}
								return resp, nil
							},
						},
						TargetFunc: func() string {
							return "api.example.org"
						},
						DoFunc: func(req *http.Request) (*http.Response, error) {
							stringReader := strings.NewReader("")
							resp := &http.Response{
								Body: io.NopCloser(stringReader),
							}
							return resp, nil
						},
						DoWithRetryFunc: func(f func() error) error {
							return f()
						},
					},
				},
			},
			wantErr:            false,
			successfulAppCount: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Cleanup(func() {
				cache.Cache = nil
			})
			i := &ImportOrg{
				Org: tt.fields.Org,
			}
			err := i.Run(tt.args.ctx)
			if tt.failedAppCount > 0 {
				assert.True(t, tt.args.ctx.Summary.AppFailureCount() == tt.failedAppCount, fmt.Sprintf("failed app count is %d, expected %d", tt.args.ctx.Summary.AppFailureCount(), tt.failedAppCount))
				return
			}
			if tt.wantErr && err != nil {
				assert.EqualError(t, err, "list of apps is empty")
				return
			}
			if (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
			}
			assert.True(t,
				tt.args.ctx.Summary.AppSuccessCount() == tt.successfulAppCount,
				fmt.Sprintf("successful app count is %d, expected %d", tt.args.ctx.Summary.AppSuccessCount(), tt.successfulAppCount),
			)
		})
	}
}
