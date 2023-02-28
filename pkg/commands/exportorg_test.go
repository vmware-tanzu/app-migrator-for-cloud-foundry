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
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/context/fakes"

	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/cache"
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/cf"
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/context"
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/metadata"
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/report"
	. "github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/testsupport"

	"github.com/stretchr/testify/assert"
)

func TestExportOrg_Run(t *testing.T) {
	type args struct {
		ctx *context.Context
		org string
	}
	pwd, _ := os.Getwd()

	tests := []struct {
		name    string
		args    args
		wantErr bool
		err     error
		handler http.Handler
	}{
		{
			name: "returns no error",
			args: args{
				ctx: &context.Context{
					ExportDir: filepath.Join(pwd, "testdata/apps"),
					DirWriter: &fakes.FakeDirWriter{
						MkdirStub: func(s string) error {
							return nil
						},
					},
					Metadata: metadata.NewMetadata(),
					Summary:  report.NewSummary(&bytes.Buffer{}),
					SpaceExporter: stubSpaceExporter{
						err:             nil,
						failedAppCount:  0,
						successAppCount: 0,
					},
				},
				org: "my_org",
			},
			wantErr: false,
			handler: TestMux(
				WithTestHandler(t, "/v2/info", InfoTestHandler),
				WithTestHandler(t, "/v2/organizations", OrgsTestHandler),
				WithTestHandler(t, "/v2/spaces", SpacesTestHandler),
				WithTestHandler(t, "/v2/organizations/a7aff246-5f5b-4cf8-87d8-f316053e4a20/spaces", SpacesTestHandler),
				WithTestHandler(t, "/v2/spaces/5489e195-c42b-4e61-bf30-323c331ecc01/apps", AppsTestHandler),
				WithTestHandler(t, "/v2/apps", AppsTestHandler),
			),
		},
		{
			name: "returns retry error",
			args: args{
				ctx: &context.Context{
					ExportDir: filepath.Join(pwd, "testdata/apps"),
					DirWriter: &fakes.FakeDirWriter{
						MkdirStub: func(s string) error {
							return nil
						},
					},
					Metadata: metadata.NewMetadata(),
					Summary:  report.NewSummary(&bytes.Buffer{}),
				},
				org: "my_org",
			},
			wantErr: true,
			err:     fmt.Errorf("timed out retrying operation, %s", cf.ErrRetry),
			handler: TestMux(
				WithTestHandler(t, "/v2/info", InfoTestHandler),
				WithTestHandler(t, "/v2/organizations", OrgsTestHandler),
				WithTestHandler(t, "/v2/spaces", func(t *testing.T) http.HandlerFunc {
					return func(w http.ResponseWriter, req *http.Request) {
						w.Header().Set("Content-Type", "application/json")
						http.Error(w, http.StatusText(http.StatusBadGateway), http.StatusBadGateway)
					}
				}),
			),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Cleanup(func() {
				cache.Cache = nil
			})
			e := &ExportOrg{}
			s := httptest.NewServer(tt.handler)
			defer s.Close()
			tt.args.ctx.ExportCFClient = NewTestCFClient(t, s)
			err := e.Run(tt.args.ctx, tt.args.org)
			if tt.wantErr && err != nil {
				assert.EqualError(t, err, tt.err.Error())
				return
			}
			if (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
				assert.EqualError(t, err, tt.err.Error())
			}
		})
	}
}
