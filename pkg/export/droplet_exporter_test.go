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

package export

import (
	"errors"
	"reflect"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/cache"

	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/cf/fakes"
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/context"
	. "github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/testsupport"

	"github.com/cloudfoundry-community/go-cfclient"
)

func TestDefaultDropletExporter_DownloadDroplet(t *testing.T) {
	type args struct {
		ctx       *context.Context
		org       cfclient.Org
		space     cfclient.Space
		app       cfclient.App
		exportDir string
	}
	logger := log.New()
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "download droplet returns a result",
			args: args{
				ctx: &context.Context{
					Logger: logger,
					ExportCFClient: StubClient{
						FakeClient: &fakes.FakeClient{},
						GetFunc: func(string) ([]byte, error) {
							return []byte(`{}`), nil
						},
					},
				},
				exportDir: "/tmp",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Cleanup(func() {
			cache.Cache = nil
		})
		t.Run(tt.name, func(t *testing.T) {
			d := &DefaultDropletExporter{}
			if err := d.DownloadDroplet(tt.args.ctx, tt.args.org, tt.args.space, tt.args.app, tt.args.exportDir); (err != nil) != tt.wantErr {
				t.Errorf("DownloadDroplet() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDefaultDropletExporter_DownloadPackages(t *testing.T) {
	type args struct {
		c         *context.Context
		org       cfclient.Org
		space     cfclient.Space
		app       cfclient.App
		exportDir string
	}
	logger := log.New()
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "download packages returns a result",
			args: args{
				c: &context.Context{
					Logger: logger,
					ExportCFClient: StubClient{
						FakeClient: &fakes.FakeClient{},
						GetFunc: func(string) ([]byte, error) {
							return []byte(`{}`), nil
						},
					},
				},
				org: cfclient.Org{
					Name: "my_org",
				},
				space: cfclient.Space{
					Name: "my_space",
				},
				app: cfclient.App{
					Name: "my_app",
				},
				exportDir: "/tmp",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Cleanup(func() {
				cache.Cache = nil
			})
			d := &DefaultDropletExporter{
				PackageRetriever: stubPackageRetriever{GetPackages: func(c *context.Context, appGUID string) (string, error) {
					return "some-guid", nil
				}},
				PackageDownloader: stubPackageDownloader{DownloadPackages: func(c *context.Context, packageGUID string) ([]byte, error) {
					return []byte(`{}`), nil
				}},
			}
			if err := d.DownloadPackages(tt.args.c, tt.args.org, tt.args.space, tt.args.app, tt.args.exportDir); (err != nil) != tt.wantErr {
				t.Errorf("DownloadPackages() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDefaultDropletExporter_NumberOfPackages(t *testing.T) {
	type args struct {
		ctx *context.Context
		app cfclient.App
	}
	logger := log.New()
	tests := []struct {
		name    string
		args    args
		want    float64
		wantErr bool
	}{
		{
			name: "get packages returns error",
			args: args{
				ctx: &context.Context{
					Logger: logger,
					ExportCFClient: StubClient{
						GetFunc: func(url string) ([]byte, error) {
							return nil, errors.New("some error")
						},
					},
				},
				app: cfclient.App{
					Guid: "cbad697f-cac1-48f4-9017-ac08f39dfb31",
				},
			},
			want:    float64(0),
			wantErr: true,
		},
		{
			name: "get packages returns result",
			args: args{
				ctx: &context.Context{
					Logger: logger,
					ExportCFClient: StubClient{
						FakeClient: &fakes.FakeClient{},
						GetFunc: func(string) ([]byte, error) {
							return []byte(`{"pagination": {"total_results":1,"total_pages":1}}`), nil
						},
					},
				}},
			want:    float64(1),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Cleanup(func() {
				cache.Cache = nil
			})
			d := &DefaultDropletExporter{}
			got, err := d.NumberOfPackages(tt.args.ctx, tt.args.app)
			if (err != nil) != tt.wantErr {
				t.Errorf("NumberOfPackages() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("NumberOfPackages() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewDropletExporter(t *testing.T) {
	tests := []struct {
		name string
		want *DefaultDropletExporter
	}{
		{
			name: "new concurrent space exporter",
			want: &DefaultDropletExporter{
				PackageRetriever:  &DefaultPackageRetriever{},
				PackageDownloader: &DefaultPackageDownloader{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Cleanup(func() {
				cache.Cache = nil
			})
			if got := NewDropletExporter(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewDropletExporter() = %v, want %v", got, tt.want)
			}
		})
	}
}

type stubPackageDownloader struct {
	DownloadPackages func(c *context.Context, packageGUID string) ([]byte, error)
}

func (s stubPackageDownloader) downloadPackages(ctx *context.Context, packageGUID string) ([]byte, error) {
	return s.DownloadPackages(ctx, packageGUID)
}

type stubPackageRetriever struct {
	GetPackages func(c *context.Context, appGUID string) (string, error)
}

func (s stubPackageRetriever) getPackages(ctx *context.Context, appGUID string) (string, error) {
	return s.GetPackages(ctx, appGUID)
}
