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

//go:build !integration || all
// +build !integration all

package commands

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/cf"
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/context/fakes"

	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/context"

	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/cache"
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/export"
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/metadata"
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/report"
	. "github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/testsupport"

	"github.com/stretchr/testify/assert"
)

func TestExportAll_Run(t *testing.T) {
	type args struct {
		ctx *context.Context
	}

	pwd, err := os.Getwd()
	assert.NoErrorf(t, err, "error should not occur")

	tests := []struct {
		name               string
		args               args
		err                error
		handler            http.Handler
		successfulAppCount int
	}{
		{
			name: "returns no export error",
			args: args{
				&context.Context{
					ExportDir: filepath.Join(pwd, "testdata/apps"),
					DirWriter: &fakes.FakeDirWriter{
						MkdirStub: func(s string) error {
							return nil
						},
					},
					Metadata:           metadata.NewMetadata(),
					Summary:            report.NewSummary(&bytes.Buffer{}),
					DropletExporter:    export.NewDropletExporter(),
					ManifestExporter:   export.NewManifestExporter(),
					AutoScalerExporter: export.NewAutoScalerExporter(),
					SpaceExporter:      stubSpaceExporter{successAppCount: 1},
				},
			},
			handler:            ExportTestHandler(t),
			successfulAppCount: 1,
		},
		{
			name: "returns no export error on retry",
			args: args{
				&context.Context{
					ExportDir: filepath.Join(pwd, "testdata/apps"),
					DirWriter: &fakes.FakeDirWriter{
						MkdirStub: func(s string) error {
							return nil
						},
					},
					Metadata:           metadata.NewMetadata(),
					Summary:            report.NewSummary(&bytes.Buffer{}),
					DropletExporter:    export.NewDropletExporter(),
					ManifestExporter:   export.NewManifestExporter(),
					AutoScalerExporter: export.NewAutoScalerExporter(),
					SpaceExporter:      stubSpaceExporter{},
				},
			},
			err: fmt.Errorf("timed out retrying operation, %s", cf.ErrRetry),
			handler: TestMux(
				WithTestHandler(t, "/v2/info", InfoTestHandler),
				WithTestHandler(t, "/v2/organizations", func(t *testing.T) http.HandlerFunc {
					return func(w http.ResponseWriter, req *http.Request) {
						w.Header().Set("Content-Type", "application/json")
						http.Error(w, http.StatusText(http.StatusBadGateway), http.StatusBadGateway)
					}
				}),
			),
		},
		{
			name: "returns no error when orgs excluded",
			args: args{
				&context.Context{
					ExportDir: filepath.Join(pwd, "testdata/apps"),
					DirWriter: &fakes.FakeDirWriter{
						MkdirStub: func(s string) error {
							return nil
						},
					},
					Metadata:           metadata.NewMetadata(),
					Summary:            report.NewSummary(&bytes.Buffer{}),
					DropletExporter:    export.NewDropletExporter(),
					ManifestExporter:   export.NewManifestExporter(),
					AutoScalerExporter: export.NewAutoScalerExporter(),
					ExcludedOrgs:       []string{"my_org"},
					SpaceExporter:      stubSpaceExporter{},
				},
			},
			handler: ExportTestHandler(t),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Cleanup(func() {
				cache.Cache = nil
			})
			e := &ExportAll{}
			s := httptest.NewServer(tt.handler)
			defer s.Close()
			tt.args.ctx.ExportCFClient = NewTestCFClient(t, s)

			err := e.Run(tt.args.ctx)
			if tt.err != nil {
				assert.EqualError(t, err, tt.err.Error())
			} else {
				require.NoError(t, err)
			}
			assert.True(t,
				tt.args.ctx.Summary.AppSuccessCount() == tt.successfulAppCount,
				fmt.Sprintf("successful app count is %d, expected %d", tt.args.ctx.Summary.AppSuccessCount(), tt.successfulAppCount),
			)
		})
	}
}
