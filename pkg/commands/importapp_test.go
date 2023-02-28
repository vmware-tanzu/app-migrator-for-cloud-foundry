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

	log "github.com/sirupsen/logrus"

	"github.com/cloudfoundry-community/go-cfclient"
	"github.com/stretchr/testify/assert"
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/cf/fakes"
	ctxfakes "github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/context/fakes"

	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/cache"
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/context"
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/metadata"
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/report"
	. "github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/testsupport"
)

func TestImportApp_Run(t *testing.T) {
	type fields struct {
		ImportSpace ImportSpace
		AppName     string
		appGUID     string
		out         *bytes.Buffer
		AppCount    int
	}
	type args struct {
		ctx *context.Context
	}
	pwd, _ := os.Getwd()
	logger := log.New()
	tests := []struct {
		name               string
		fields             fields
		args               args
		wantErr            bool
		err                error
		successfulAppCount int
	}{
		{
			name: "imports app",
			fields: fields{
				ImportSpace: ImportSpace{
					ImportOrg: ImportOrg{Org: "my_org"},
					Space:     "my_space",
				},
				AppName:  "my_app",
				appGUID:  "6064d98a-95e6-400b-bc03-be65e6d59622",
				out:      &bytes.Buffer{},
				AppCount: 1,
			},
			args: args{
				ctx: &context.Context{
					Logger:    logger,
					ExportDir: filepath.Join(pwd, "testdata/apps"),
					Metadata:  metadata.NewMetadata(),
					Summary:   report.NewSummary(&bytes.Buffer{}),
					ImportCFClient: &fakes.FakeClient{
						DoWithRetryStub: func(f func() error) error {
							return f()
						},
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
						ListAppsByQueryStub: func(url.Values) ([]cfclient.App, error) {
							return []cfclient.App{
								{
									Name: "my_app",
								},
							}, nil
						},
						GetAppByGuidNoInlineCallStub: func(string) (cfclient.App, error) {
							return cfclient.App{
								Name: "my_app",
							}, nil
						},
						ListUserProvidedServiceInstancesByQueryStub: func(url.Values) ([]cfclient.UserProvidedServiceInstance, error) {
							return []cfclient.UserProvidedServiceInstance{
								{
									Name: "my_ups",
								},
							}, nil
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
						DoStub: func(req *http.Request) (*http.Response, error) {
							stringReader := strings.NewReader("")
							resp := &http.Response{
								Body:       io.NopCloser(stringReader),
								StatusCode: http.StatusOK,
							}
							return resp, nil
						},
					},
					SpaceImporter: &ctxfakes.FakeSpaceImporter{
						ImportSpaceStub: func(ctx *context.Context, processFunc context.ProcessFunc, strings []string) (<-chan context.ProcessResult, error) {
							results := make(chan context.ProcessResult, 1)
							defer close(results)
							results <- context.ProcessResult{Value: "my_app"}
							ctx.Summary.AddSuccessfulApp("my_org", "my_space", "my_app")
							return results, nil
						},
					},
				},
			},
			successfulAppCount: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Cleanup(func() {
				cache.Cache = nil
			})
			i := &ImportApp{
				ImportSpace: tt.fields.ImportSpace,
				AppName:     tt.fields.AppName,
				appGUID:     tt.fields.appGUID,
				AppCount:    tt.fields.AppCount,
			}
			err := i.Run(tt.args.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err != nil {
				assert.EqualError(t, err, tt.err.Error())
			}
			assert.True(t,
				tt.args.ctx.Summary.AppSuccessCount() == tt.successfulAppCount,
				fmt.Sprintf("successful app count is %d, expected %d", tt.args.ctx.Summary.AppSuccessCount(), tt.successfulAppCount),
			)
		})
	}
}

func TestImportApp_applyAutoscalerInstances(t *testing.T) {
	type fields struct {
		ImportSpace ImportSpace
		AppName     string
		appGUID     string
		out         *bytes.Buffer
		AppCount    int
	}
	type args struct {
		ctx *context.Context
	}
	pwd, _ := os.Getwd()
	logger := log.New()
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "applies autoscaler instances",
			fields: fields{
				ImportSpace: ImportSpace{
					ImportOrg: ImportOrg{Org: "my_org"},
					Space:     "my_space",
				},
				AppName:  "my_app",
				appGUID:  "6064d98a-95e6-400b-bc03-be65e6d59622",
				out:      &bytes.Buffer{},
				AppCount: 1,
			},
			args: args{
				ctx: &context.Context{
					Logger:    logger,
					ExportDir: filepath.Join(pwd, "testdata/apps"),
					Metadata:  metadata.NewMetadata(),
					Summary:   report.NewSummary(&bytes.Buffer{}),
					ImportCFClient: StubClient{
						FakeClient: &fakes.FakeClient{},
						TargetFunc: func() string {
							return "api.example.org"
						},
						DoFunc: func(req *http.Request) (*http.Response, error) {
							stringReader := strings.NewReader("")
							resp := &http.Response{
								Body:       io.NopCloser(stringReader),
								StatusCode: http.StatusOK,
							}
							return resp, nil
						},
						DoWithRetryFunc: func(f func() error) error {
							return f()
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Cleanup(func() {
				cache.Cache = nil
			})
			i := &ImportApp{
				ImportSpace: tt.fields.ImportSpace,
				AppName:     tt.fields.AppName,
				appGUID:     tt.fields.appGUID,
				AppCount:    tt.fields.AppCount,
			}
			if err := i.applyAutoscalerInstances(tt.args.ctx); (err != nil) != tt.wantErr {
				t.Errorf("applyAutoscalerInstances() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestImportApp_applyAutoscalerRules(t *testing.T) {
	type fields struct {
		ImportSpace ImportSpace
		AppName     string
		appGUID     string
		out         *bytes.Buffer
		AppCount    int
	}
	type args struct {
		ctx *context.Context
	}
	pwd, _ := os.Getwd()
	logger := log.New()
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "applies autoscaler rules",
			fields: fields{
				ImportSpace: ImportSpace{
					ImportOrg: ImportOrg{Org: "my_org"},
					Space:     "my_space",
				},
				AppName:  "my_app",
				appGUID:  "6064d98a-95e6-400b-bc03-be65e6d59622",
				out:      &bytes.Buffer{},
				AppCount: 1,
			},
			args: args{
				ctx: &context.Context{
					Logger:    logger,
					ExportDir: filepath.Join(pwd, "testdata/apps"),
					Metadata:  metadata.NewMetadata(),
					Summary:   report.NewSummary(&bytes.Buffer{}),
					ImportCFClient: StubClient{
						FakeClient: &fakes.FakeClient{},
						TargetFunc: func() string {
							return "api.example.org"
						},
						DoFunc: func(req *http.Request) (*http.Response, error) {
							stringReader := strings.NewReader("")
							resp := &http.Response{
								Body:       io.NopCloser(stringReader),
								StatusCode: http.StatusOK,
							}
							return resp, nil
						},
						DoWithRetryFunc: func(f func() error) error {
							return f()
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Cleanup(func() {
				cache.Cache = nil
			})
			i := &ImportApp{
				ImportSpace: tt.fields.ImportSpace,
				AppName:     tt.fields.AppName,
				appGUID:     tt.fields.appGUID,
				AppCount:    tt.fields.AppCount,
			}
			if err := i.applyAutoscalerRules(tt.args.ctx); (err != nil) != tt.wantErr {
				t.Errorf("applyAutoscalerRules() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestImportApp_applyAutoscalerSchedules(t *testing.T) {
	type fields struct {
		ImportSpace ImportSpace
		AppName     string
		appGUID     string
		out         *bytes.Buffer
		AppCount    int
	}
	type args struct {
		ctx *context.Context
	}
	pwd, _ := os.Getwd()
	logger := log.New()
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "applies autoscaler schedules",
			fields: fields{
				ImportSpace: ImportSpace{
					ImportOrg: ImportOrg{Org: "my_org"},
					Space:     "my_space",
				},
				AppName:  "my_app",
				appGUID:  "6064d98a-95e6-400b-bc03-be65e6d59622",
				out:      &bytes.Buffer{},
				AppCount: 1,
			},
			args: args{
				ctx: &context.Context{
					Logger:    logger,
					ExportDir: filepath.Join(pwd, "testdata/apps"),
					Metadata:  metadata.NewMetadata(),
					Summary:   report.NewSummary(&bytes.Buffer{}),
					ImportCFClient: StubClient{
						FakeClient: &fakes.FakeClient{},
						TargetFunc: func() string {
							return "api.example.org"
						},
						DoFunc: func(req *http.Request) (*http.Response, error) {
							stringReader := strings.NewReader("")
							resp := &http.Response{
								Body:       io.NopCloser(stringReader),
								StatusCode: http.StatusOK,
							}
							return resp, nil
						},
						DoWithRetryFunc: func(f func() error) error {
							return f()
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Cleanup(func() {
				cache.Cache = nil
			})
			i := &ImportApp{
				ImportSpace: tt.fields.ImportSpace,
				AppName:     tt.fields.AppName,
				appGUID:     tt.fields.appGUID,
				AppCount:    tt.fields.AppCount,
			}
			if err := i.applyAutoscalerSchedules(tt.args.ctx); (err != nil) != tt.wantErr {
				t.Errorf("applyAutoscalerSchedules() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestImportApp_bindRoutes(t *testing.T) {
	type fields struct {
		ImportSpace ImportSpace
		AppName     string
		appGUID     string
		out         *bytes.Buffer
		AppCount    int
	}
	type args struct {
		ctx    *context.Context
		routes []string
	}
	pwd, _ := os.Getwd()
	logger := log.New()
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "binds route to app",
			fields: fields{
				ImportSpace: ImportSpace{
					ImportOrg: ImportOrg{Org: "my_org"},
					Space:     "my_space",
				},
				AppName:  "my_app",
				appGUID:  "6064d98a-95e6-400b-bc03-be65e6d59622",
				out:      &bytes.Buffer{},
				AppCount: 1,
			},
			args: args{
				ctx: &context.Context{
					Logger:    logger,
					ExportDir: filepath.Join(pwd, "testdata/apps"),
					ImportCFClient: StubClient{
						FakeClient: &fakes.FakeClient{
							BindRouteStub: func(string, string) error {
								return nil
							},
							CreateRouteStub: func(cfclient.RouteRequest) (cfclient.Route, error) {
								return cfclient.Route{}, nil
							},
						},
						TargetFunc: func() string {
							return "api.example.org"
						},
						DoFunc: func(req *http.Request) (*http.Response, error) {
							stringReader := strings.NewReader("")
							resp := &http.Response{
								Body:       io.NopCloser(stringReader),
								StatusCode: http.StatusOK,
							}
							return resp, nil
						},
						DoWithRetryFunc: func(f func() error) error {
							return f()
						},
					},
				},
				routes: []string{"a-hostname.a-domain.com/some_path"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Cleanup(func() {
				cache.Cache = nil
			})
			i := &ImportApp{
				ImportSpace: tt.fields.ImportSpace,
				AppName:     tt.fields.AppName,
				appGUID:     tt.fields.appGUID,
				AppCount:    tt.fields.AppCount,
			}
			if err := i.bindRoutes(tt.args.ctx, tt.args.routes); (err != nil) != tt.wantErr {
				t.Errorf("bindRoutes() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestImportApp_bindServices(t *testing.T) {
	type fields struct {
		ImportSpace ImportSpace
		AppName     string
		appGUID     string
		out         *bytes.Buffer
		AppCount    int
	}
	type args struct {
		ctx          *context.Context
		serviceNames []string
	}
	pwd, _ := os.Getwd()
	logger := log.New()
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "creates service instance",
			fields: fields{
				ImportSpace: ImportSpace{
					ImportOrg: ImportOrg{Org: "my_org"},
					Space:     "my_space",
				},
				AppName:  "my_app",
				appGUID:  "6064d98a-95e6-400b-bc03-be65e6d59622",
				out:      &bytes.Buffer{},
				AppCount: 1,
			},
			args: args{
				ctx: &context.Context{
					Logger:    logger,
					ExportDir: filepath.Join(pwd, "testdata/apps"),
					ImportCFClient: StubClient{
						FakeClient: &fakes.FakeClient{
							ListServiceInstancesByQueryStub: func(url.Values) ([]cfclient.ServiceInstance, error) {
								return []cfclient.ServiceInstance{{
									Guid: "a14baddf-1ccc-5299-0152-ab9s49de4422",
									Name: "my-service",
								}}, nil
							},
							CreateServiceBindingStub: func(string, string) (*cfclient.ServiceBinding, error) {
								return &cfclient.ServiceBinding{
									Guid:                "0b6e8fe9-b173-4845-a7aa-e093f1081c94",
									Name:                "my-database",
									ServiceInstanceGuid: "a14baddf-1ccc-5299-0152-ab9s49de4422",
								}, nil
							},
						},
						TargetFunc: func() string {
							return "api.example.org"
						},
						DoFunc: func(req *http.Request) (*http.Response, error) {
							stringReader := strings.NewReader("")
							resp := &http.Response{
								Body:       io.NopCloser(stringReader),
								StatusCode: http.StatusOK,
							}
							return resp, nil
						},
						DoWithRetryFunc: func(f func() error) error {
							return f()
						},
					},
				},
				serviceNames: []string{"my-service"},
			},
		},
		{
			name: "creates user provided service instance",
			fields: fields{
				ImportSpace: ImportSpace{
					ImportOrg: ImportOrg{Org: "my_org"},
					Space:     "my_space",
				},
				AppName:  "my_app",
				appGUID:  "6064d98a-95e6-400b-bc03-be65e6d59622",
				out:      &bytes.Buffer{},
				AppCount: 1,
			},
			args: args{
				ctx: &context.Context{
					Logger:    logger,
					ExportDir: filepath.Join(pwd, "testdata/apps"),
					ImportCFClient: StubClient{
						FakeClient: &fakes.FakeClient{
							ListServiceInstancesByQueryStub: func(url.Values) ([]cfclient.ServiceInstance, error) {
								return []cfclient.ServiceInstance{}, nil
							},
							ListUserProvidedServiceInstancesByQueryStub: func(url.Values) ([]cfclient.UserProvidedServiceInstance, error) {
								return []cfclient.UserProvidedServiceInstance{{
									Name: "service-provided-instance",
								}}, nil
							},
						},
						TargetFunc: func() string {
							return "api.example.org"
						},
						DoFunc: func(req *http.Request) (*http.Response, error) {
							stringReader := strings.NewReader("")
							resp := &http.Response{
								Body:       io.NopCloser(stringReader),
								StatusCode: http.StatusOK,
							}
							return resp, nil
						},
						DoWithRetryFunc: func(f func() error) error {
							return f()
						},
					},
				},
				serviceNames: []string{"my-service"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Cleanup(func() {
				cache.Cache = nil
			})
			i := &ImportApp{
				ImportSpace: tt.fields.ImportSpace,
				AppName:     tt.fields.AppName,
				appGUID:     tt.fields.appGUID,
				AppCount:    tt.fields.AppCount,
			}
			if err := i.bindServices(tt.args.ctx, tt.args.serviceNames); (err != nil) != tt.wantErr {
				t.Errorf("bindServices() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestImportApp_createApp(t *testing.T) {
	type fields struct {
		ImportSpace ImportSpace
		AppName     string
		appGUID     string
		out         *bytes.Buffer
		AppCount    int
	}
	type args struct {
		ctx *context.Context
	}
	myApp := cfclient.App{
		Name:      "my_app",
		StackGuid: "f6c960cc-98ba-4fd1-b197-ecbf39108aa2",
		Buildpack: "my_buildpack",
	}
	mySpace := cfclient.Space{Name: "my_space"}
	myOrg := cfclient.Org{Name: "my_org"}
	pwd, _ := os.Getwd()
	logger := log.New()
	tests := []struct {
		name               string
		fields             fields
		args               args
		wantErr            bool
		successfulAppCount int
	}{
		{
			name: "creates app",
			fields: fields{
				ImportSpace: ImportSpace{
					ImportOrg: ImportOrg{Org: "my_org"},
					Space:     "my_space",
				},
				AppName:  "my_app",
				appGUID:  "6064d98a-95e6-400b-bc03-be65e6d59622",
				out:      &bytes.Buffer{},
				AppCount: 1,
			},
			args: args{
				&context.Context{
					Logger:    logger,
					ExportDir: filepath.Join(pwd, "testdata/apps"),
					Metadata:  metadata.NewMetadata(),
					Summary:   report.NewSummary(&bytes.Buffer{}),
					ImportCFClient: StubClient{
						FakeClient: &fakes.FakeClient{
							ListAppsByQueryStub: func(values url.Values) ([]cfclient.App, error) {
								return []cfclient.App{myApp}, nil
							},
							GetOrgByNameStub: func(s string) (cfclient.Org, error) {
								return myOrg, nil
							},
							GetSpaceByNameStub: func(string, string) (cfclient.Space, error) {
								return mySpace, nil
							},
							ListStacksByQueryStub: func(url.Values) ([]cfclient.Stack, error) {
								return []cfclient.Stack{{
									Guid: "f6c960cc-98ba-4fd1-b197-ecbf39108aa2",
									Name: "cflinuxfs3",
								}}, nil
							},
							BindRouteStub: func(string, string) error {
								return nil
							},
							CreateRouteStub: func(cfclient.RouteRequest) (cfclient.Route, error) {
								return cfclient.Route{}, nil
							},
							ListServiceInstancesByQueryStub: func(url.Values) ([]cfclient.ServiceInstance, error) {
								return []cfclient.ServiceInstance{}, nil
							},
							ListUserProvidedServiceInstancesByQueryStub: func(url.Values) ([]cfclient.UserProvidedServiceInstance, error) {
								return []cfclient.UserProvidedServiceInstance{{
									Name: "service-provided-instance",
								}}, nil
							},
						},
						DoWithRetryFunc: func(f func() error) error {
							return f()
						},
					},
				},
			},
			successfulAppCount: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Cleanup(func() {
				cache.Cache = nil
			})
			i := &ImportApp{
				ImportSpace: tt.fields.ImportSpace,
				AppName:     tt.fields.AppName,
				appGUID:     tt.fields.appGUID,
				AppCount:    tt.fields.AppCount,
			}
			if err := i.createApp(tt.args.ctx); (err != nil) != tt.wantErr {
				t.Errorf("createApp() error = %v, wantErr %v", err, tt.wantErr)
			}
			assert.True(t,
				tt.args.ctx.Summary.AppSuccessCount() == tt.successfulAppCount,
				fmt.Sprintf("successful app count is %d, expected %d", tt.args.ctx.Summary.AppSuccessCount(), tt.successfulAppCount),
			)
		})
	}
}

func TestImportApp_getAppNameFromManifest(t *testing.T) {
	type fields struct {
		ImportSpace ImportSpace
		AppName     string
		appGUID     string
		AppCount    int
	}
	type args struct {
		ctx *context.Context
	}
	pwd, _ := os.Getwd()
	logger := log.New()
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "gets app name",
			fields: fields{
				ImportSpace: ImportSpace{
					ImportOrg: ImportOrg{Org: "my_org"},
					Space:     "my_space",
				},
				AppName: "my_app",
			},
			args: args{
				ctx: &context.Context{
					Logger:    logger,
					ExportDir: filepath.Join(pwd, "testdata/apps"),
				},
			},
			want: "my_app",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Cleanup(func() {
				cache.Cache = nil
			})
			i := &ImportApp{
				ImportSpace: tt.fields.ImportSpace,
				AppName:     tt.fields.AppName,
				appGUID:     tt.fields.appGUID,
				AppCount:    tt.fields.AppCount,
			}
			got, err := i.getAppNameFromManifest(tt.args.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("getAppNameFromManifest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("getAppNameFromManifest() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestImportApp_uploadAppBits(t *testing.T) {
	type fields struct {
		ImportSpace ImportSpace
		AppName     string
		appGUID     string
		out         *bytes.Buffer
		AppCount    int
	}
	type args struct {
		ctx *context.Context
	}
	pwd, _ := os.Getwd()
	logger := log.New()
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "uploads app bits",
			fields: fields{
				ImportSpace: ImportSpace{
					ImportOrg: ImportOrg{Org: "my_org"},
					Space:     "my_space",
				},
				AppName:  "my_app",
				appGUID:  "6064d98a-95e6-400b-bc03-be65e6d59622",
				out:      &bytes.Buffer{},
				AppCount: 1,
			},
			args: args{
				ctx: &context.Context{
					Logger:    logger,
					ExportDir: filepath.Join(pwd, "testdata/apps"),
					ImportCFClient: StubClient{
						FakeClient: &fakes.FakeClient{},
						DoWithRetryFunc: func(f func() error) error {
							return f()
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Cleanup(func() {
				cache.Cache = nil
			})
			i := &ImportApp{
				ImportSpace: tt.fields.ImportSpace,
				AppName:     tt.fields.AppName,
				appGUID:     tt.fields.appGUID,
				AppCount:    tt.fields.AppCount,
			}
			if err := i.uploadAppBits(tt.args.ctx); (err != nil) != tt.wantErr {
				t.Errorf("uploadAppBits() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestImportApp_uploadBlob(t *testing.T) {
	type fields struct {
		ImportSpace ImportSpace
		AppName     string
		appGUID     string
		AppCount    int
	}
	type args struct {
		ctx *context.Context
	}
	pwd, _ := os.Getwd()
	logger := log.New()
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "uploads blob",
			fields: fields{
				AppName: "my_app",
				ImportSpace: ImportSpace{
					ImportOrg: ImportOrg{Org: "my_org"},
					Space:     "my_space",
				},
			},
			args: args{
				ctx: &context.Context{
					Logger:    logger,
					ExportDir: filepath.Join(pwd, "testdata/apps"),
					ImportCFClient: StubClient{
						FakeClient: &fakes.FakeClient{
							UploadDropletBitsStub: func(reader io.Reader, s string) (string, error) {
								return "", nil
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
						DoWithRetryFunc: func(f func() error) error {
							return f()
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Cleanup(func() {
				cache.Cache = nil
			})
			i := &ImportApp{
				ImportSpace: tt.fields.ImportSpace,
				AppName:     tt.fields.AppName,
				appGUID:     tt.fields.appGUID,
				AppCount:    tt.fields.AppCount,
			}
			if err := i.uploadBlob(tt.args.ctx); (err != nil) != tt.wantErr {
				t.Errorf("uploadBlob() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestImportApp_uploadDroplet(t *testing.T) {
	type fields struct {
		ImportSpace ImportSpace
		AppName     string
		appGUID     string
		AppCount    int
	}
	type args struct {
		ctx *context.Context
	}
	pwd, _ := os.Getwd()
	logger := log.New()
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "uploads droplet",
			fields: fields{
				AppName: "my_app",
				ImportSpace: ImportSpace{
					ImportOrg: ImportOrg{Org: "my_org"},
					Space:     "my_space",
				},
			},
			args: args{
				ctx: &context.Context{
					Logger:    logger,
					ExportDir: filepath.Join(pwd, "testdata/apps"),
					ImportCFClient: StubClient{
						FakeClient: &fakes.FakeClient{
							UploadDropletBitsStub: func(reader io.Reader, s string) (string, error) {
								return "", nil
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
						DoWithRetryFunc: func(f func() error) error {
							return f()
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Cleanup(func() {
				cache.Cache = nil
			})
			i := &ImportApp{
				ImportSpace: tt.fields.ImportSpace,
				AppName:     tt.fields.AppName,
				appGUID:     tt.fields.appGUID,
				AppCount:    tt.fields.AppCount,
			}
			if err := i.uploadDroplet(tt.args.ctx); (err != nil) != tt.wantErr {
				t.Errorf("uploadDroplet() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_getSizeFromString(t *testing.T) {
	type args struct {
		sizeStr string
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{
			name: "size of 1G",
			args: args{
				sizeStr: "1G",
			},
			want: 1024,
		},
		{
			name: "size of 256M",
			args: args{
				sizeStr: "256M",
			},
			want: 256,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Cleanup(func() {
				cache.Cache = nil
			})
			if got := getSizeFromString(tt.args.sizeStr); got != tt.want {
				t.Errorf("getSizeFromString() = %v, want %v", got, tt.want)
			}
		})
	}
}
