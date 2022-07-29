//go:build !integration || all
// +build !integration all

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
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/context/fakes"

	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/cache"
	cffakes "github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/cf/fakes"
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/context"
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/export"
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/metadata"
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/report"
	. "github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/testsupport"

	"github.com/cloudfoundry-community/go-cfclient"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
)

func TestExportApp_Run(t *testing.T) {
	logger := log.New()
	type args struct {
		ctx        *context.Context
		org, space string
	}
	type fields struct {
		out *bytes.Buffer
	}
	tests := []struct {
		name    string
		args    args
		fields  fields
		app     ExportApp
		wantErr bool
		errMsg  string
	}{
		{
			name: "invalid directory",
			app: ExportApp{
				ExportSpace: ExportSpace{},
				AppName:     "",
			},
			fields: fields{
				out: &bytes.Buffer{},
			},
			args: args{
				ctx: &context.Context{
					Logger:    logger,
					ExportDir: "/doesnotexist",
					DirWriter: &fakes.FakeDirWriter{
						MkdirStub: func(s string) error {
							return fmt.Errorf("dir does not exist")
						},
					},
					Metadata: metadata.NewMetadata(),
					ExportCFClient: StubClient{
						FakeClient: &cffakes.FakeClient{
							ListAppsByQueryStub: func(values url.Values) ([]cfclient.App, error) {
								return []cfclient.App{{Name: "my_app"}}, nil
							},
						},
						DoWithRetryFunc: func(f func() error) error {
							return f()
						},
					},
				},
				org:   "my_org",
				space: "my_space",
			},
			wantErr: true,
			errMsg:  "cannot create target directory: dir does not exist",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Cleanup(func() {
				cache.Cache = nil
			})
			logger.SetOutput(tt.fields.out)
			err := tt.app.Run(tt.args.ctx, tt.args.org, tt.args.space)
			if tt.wantErr {
				if expected, actual := tt.errMsg, err.Error(); !strings.Contains(actual, expected) {
					t.Errorf("expected %s, actual %s", expected, actual)
				}
				return
			}
			assert.NoError(t, err)
		})
	}
}

func TestExportApp_RunWithHTTPServer(t *testing.T) {
	pwd, _ := os.Getwd()
	logger := log.New()
	tests := []struct {
		name             string
		Context          *context.Context
		Org              string
		Space            string
		Want             string
		WantErr          bool
		ShouldContain    string
		ShouldNotContain string
		Assertion        AssertionFunc
		Handler          http.Handler
	}{
		{
			name:    "success",
			Handler: ExportTestHandler(t),
			Context: &context.Context{
				Logger:    logger,
				ExportDir: filepath.Join(pwd, "testdata/apps"),
				DirWriter: &fakes.FakeDirWriter{
					MkdirStub: func(s string) error {
						return nil
					},
				},
				Metadata:           metadata.NewMetadata(),
				Summary:            report.NewSummary(&bytes.Buffer{}),
				AutoScalerExporter: export.NewAutoScalerExporter(),
				DropletExporter:    export.NewDropletExporter(),
				ManifestExporter:   export.NewManifestExporter(),
			},
			Org:              "my_org",
			Space:            "my_space",
			ShouldNotContain: "Error occurred",
			Assertion:        SuccessfulAppCount(t, 1),
		},
		{
			name:    "droplet exporter returns err",
			Handler: ExportTestHandler(t),
			Context: &context.Context{
				Logger:    logger,
				ExportDir: filepath.Join(pwd, "testdata/apps"),
				DirWriter: &fakes.FakeDirWriter{
					MkdirStub: func(s string) error {
						return nil
					},
				},
				Metadata: metadata.NewMetadata(),
				Summary:  report.NewSummary(&bytes.Buffer{}),
				DropletExporter: stubDropletExporter{
					numberOfPackages: func(ctx *context.Context, app cfclient.App) (float64, error) {
						var count float64
						return count, errors.New("error getting packages")
					},
				},
			},
			Org:           "my_org",
			Space:         "my_space",
			ShouldContain: "error getting packages",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Cleanup(func() {
				cache.Cache = nil
			})
			ts := httptest.NewServer(tt.Handler)
			defer ts.Close()
			cf := NewTestCFClient(t, ts)
			tt.Context.ExportCFClient = cf
			tc := TestCase{
				Context:          tt.Context,
				Org:              tt.Org,
				Space:            tt.Space,
				Want:             tt.Want,
				WantErr:          tt.WantErr,
				ShouldContain:    tt.ShouldContain,
				ShouldNotContain: tt.ShouldNotContain,
				Assertion:        tt.Assertion,
				Handler:          tt.Handler,
			}
			executeTests(t, tc)
		})
	}
}

func executeTests(t *testing.T, tc TestCase) {
	exportApp := ExportApp{
		ExportSpace: ExportSpace{
			Space: "my_space",
			ExportOrg: ExportOrg{
				Org: "my_org",
			},
		},
		AppName: "my_app",
	}
	out := &bytes.Buffer{}
	logger := log.New()
	logger.SetOutput(out)
	tc.Context.Logger = logger
	err := exportApp.Run(tc.Context, tc.Org, tc.Space)
	if (err != nil) != tc.WantErr {
		t.Errorf("execute should not error: %v", err)
	}
	if tc.WantErr {
		assert.Equal(t, tc.Want, err.Error())
		return
	}

	if tc.Assertion != nil {
		tc.Assertion(tc.Context)
	}

	got := out.String()

	if len(tc.ShouldNotContain) > 0 {
		if expected, actual := tc.ShouldNotContain, got; strings.Contains(actual, expected) {
			t.Errorf("expected %s, actual %s", expected, actual)
		}
	}
	if len(tc.ShouldContain) > 0 {
		if expected, actual := tc.ShouldContain, got; !strings.Contains(actual, expected) {
			t.Errorf("expected %s, actual %s", expected, actual)
		}
	}

	if tc.Want != "" {
		diff := cmp.Diff(tc.Want, got)
		if diff != "" {
			t.Error(diff)
		}
	}
}

type TestCase struct {
	Context          *context.Context
	Org              string
	Space            string
	Want             string
	WantErr          bool
	ShouldContain    string
	ShouldNotContain string
	Assertion        AssertionFunc
	Handler          http.Handler
}

// Assertion helpers

type AssertionFunc func(ctx *context.Context) bool

func SuccessfulAppCount(t *testing.T, count int) AssertionFunc {
	return func(ctx *context.Context) bool {
		return assert.True(t,
			ctx.Summary.AppSuccessCount() == count,
			fmt.Sprintf("successful app count is %d, expected %d", ctx.Summary.AppSuccessCount(), count),
		)
	}
}

type stubDropletExporter struct {
	numberOfPackages func(ctx *context.Context, app cfclient.App) (float64, error)
	downloadDroplet  func(ctx *context.Context, org cfclient.Org, space cfclient.Space, app cfclient.App, exportDir string) error
	downloadPackages func(ctx *context.Context, org cfclient.Org, space cfclient.Space, app cfclient.App, exportDir string) error
}

func (s stubDropletExporter) NumberOfPackages(ctx *context.Context, app cfclient.App) (float64, error) {
	return s.numberOfPackages(ctx, app)
}

func (s stubDropletExporter) DownloadDroplet(ctx *context.Context, org cfclient.Org, space cfclient.Space, app cfclient.App, exportDir string) error {
	return s.downloadDroplet(ctx, org, space, app, exportDir)
}

func (s stubDropletExporter) DownloadPackages(ctx *context.Context, org cfclient.Org, space cfclient.Space, app cfclient.App, exportDir string) error {
	return s.downloadPackages(ctx, org, space, app, exportDir)
}
