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
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/context/fakes"

	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/context"

	"github.com/cloudfoundry-community/go-cfclient"
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/cache"
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/metadata"
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/report"
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/testsupport"

	"github.com/stretchr/testify/assert"
)

func TestExportSpace_Run(t *testing.T) {
	type fields struct {
		ExportOrg ExportOrg
		Space     string
	}
	type args struct {
		ctx        *context.Context
		org, space string
	}
	pwd, _ := os.Getwd()
	tests := []struct {
		name           string
		fields         fields
		args           args
		wantErr        bool
		failedAppCount int
	}{
		{
			name:   "returns no error",
			fields: fields{ExportOrg: ExportOrg{Org: "my_org"}, Space: "my_space"},
			args: args{
				org:   "my_org",
				space: "my_space",
				ctx: &context.Context{
					ExportDir: filepath.Join(pwd, "testdata/apps"),
					DirWriter: &fakes.FakeDirWriter{
						MkdirStub: func(s string) error {
							return nil
						},
					},
					Metadata: metadata.NewMetadata(),
					Summary:  report.NewSummary(&bytes.Buffer{}),
					ExportCFClient: testsupport.StubClient{
						DoWithRetryFunc: func(f func() error) error {
							return nil
						},
					},
					SpaceExporter: stubSpaceExporter{successAppCount: 1},
				},
			},
			wantErr: false,
		},
		{
			name:   "returns an error",
			fields: fields{ExportOrg: ExportOrg{Org: "my_org"}, Space: "my_space"},
			args: args{
				org:   "my_org",
				space: "my_space",
				ctx: &context.Context{
					ExportDir: filepath.Join(pwd, "testdata/apps"),
					DirWriter: &fakes.FakeDirWriter{
						MkdirStub: func(s string) error {
							return nil
						},
					},
					Metadata: metadata.NewMetadata(),
					Summary:  report.NewSummary(&bytes.Buffer{}),
					ExportCFClient: testsupport.StubClient{
						DoWithRetryFunc: func(f func() error) error {
							return nil
						},
					},
					SpaceExporter: stubSpaceExporter{err: errors.New("some error")},
				},
			},
			wantErr: true,
		},
		{
			name:   "returns a failed app",
			fields: fields{ExportOrg: ExportOrg{Org: "my_org"}, Space: "my_space"},
			args: args{
				org:   "my_org",
				space: "my_space",
				ctx: &context.Context{
					ExportDir: filepath.Join(pwd, "testdata/apps"),
					DirWriter: &fakes.FakeDirWriter{
						MkdirStub: func(s string) error {
							return nil
						},
					},
					Metadata: metadata.NewMetadata(),
					Summary:  report.NewSummary(&bytes.Buffer{}),
					ExportCFClient: testsupport.StubClient{
						DoWithRetryFunc: func(f func() error) error {
							return nil
						},
					},
					SpaceExporter: stubSpaceExporter{err: errors.New("some error"), failedAppCount: 1, successAppCount: 0},
				},
			},
			failedAppCount: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Cleanup(func() {
				cache.Cache = nil
			})
			e := &ExportSpace{
				ExportOrg: tt.fields.ExportOrg,
				Space:     tt.fields.Space,
			}
			err := e.Run(tt.args.ctx, tt.args.org, tt.args.space)
			if tt.failedAppCount > 0 {
				assert.True(t, tt.args.ctx.Summary.AppFailureCount() == tt.failedAppCount, fmt.Sprintf("failed app count is %d, expected %d", tt.args.ctx.Summary.AppFailureCount(), tt.failedAppCount))
				return
			}
			if tt.wantErr && err != nil {
				assert.EqualError(t, err, "some error")
				return
			}
			if (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

type stubSpaceExporter struct {
	err             error
	failedAppCount  int
	successAppCount int
}

func (s stubSpaceExporter) ExportSpace(ctx *context.Context, space cfclient.Space, processor context.ProcessFunc) (<-chan context.ProcessResult, error) {
	results := make(chan context.ProcessResult, 1)
	defer close(results)
	if s.failedAppCount > 0 {
		results <- context.ProcessResult{Value: "my_app", Err: s.err}
		return results, nil
	}
	if s.err != nil {
		return nil, s.err
	}
	results <- context.ProcessResult{Value: "my_app"}
	ctx.Summary.AddSuccessfulApp("my_org", space.Name, "my_app")
	return results, nil
}
