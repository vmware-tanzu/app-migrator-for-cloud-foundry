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
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/cloudfoundry-community/go-cfclient"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/cache"
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/cf/fakes"
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/context"
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/metadata"
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/report"
	. "github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/testsupport"
)

func TestImportIncremental_Run(t *testing.T) {
	type fields struct {
		app   cfclient.App
		space cfclient.Space
		org   cfclient.Org
	}
	type args struct {
		ctx *context.Context
	}
	myApp := cfclient.App{Name: "my_app"}
	mySpace := cfclient.Space{Name: "my_space"}
	myOrg := cfclient.Org{Name: "my_org"}
	pwd, _ := os.Getwd()
	logger := log.New()
	tests := []struct {
		name         string
		fields       fields
		args         args
		recordUpdate bool
	}{
		{
			name: "imports app",
			fields: fields{
				app:   myApp,
				space: mySpace,
				org:   myOrg,
			},
			args: args{
				ctx: &context.Context{
					Logger:    logger,
					ExportDir: filepath.Join(pwd, "testdata/apps"),
					Metadata:  metadata.NewMetadata(),
					Summary:   report.NewSummary(&bytes.Buffer{}),
					ImportCFClient: StubClient{
						FakeClient: &fakes.FakeClient{
							DoRequestStub: func(*cfclient.Request) (*http.Response, error) {
								data, err := ioutil.ReadFile("testdata/v3droplets.json")
								assert.NoError(t, err)
								stringReader := strings.NewReader(string(data))
								resp := &http.Response{
									Body: io.NopCloser(stringReader),
								}
								return resp, nil
							},
							ListAppsByQueryStub: func(values url.Values) ([]cfclient.App, error) {
								return []cfclient.App{myApp}, nil
							},
							GetOrgByNameStub: func(s string) (cfclient.Org, error) {
								return myOrg, nil
							},
							GetSpaceByNameStub: func(string, string) (cfclient.Space, error) {
								return mySpace, nil
							},
							ListUserProvidedServiceInstancesByQueryStub: func(url.Values) ([]cfclient.UserProvidedServiceInstance, error) {
								return []cfclient.UserProvidedServiceInstance{{
									Name: "name-1508",
								}}, nil
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
			recordUpdate: true,
		},
		{
			name: "does not import app",
			fields: fields{
				app:   myApp,
				space: mySpace,
				org:   myOrg,
			},
			args: args{
				ctx: &context.Context{
					Logger:    logger,
					ExportDir: filepath.Join(pwd, "testdata/apps"),
					Metadata:  metadata.NewMetadata(),
					Summary:   report.NewSummary(&bytes.Buffer{}),
					ImportCFClient: &fakes.FakeClient{
						ListAppsByQueryStub: func(values url.Values) ([]cfclient.App, error) {
							return []cfclient.App{myApp}, nil
						},
						GetOrgByNameStub: func(s string) (cfclient.Org, error) {
							return myOrg, nil
						},
						GetSpaceByNameStub: func(string, string) (cfclient.Space, error) {
							return mySpace, nil
						},
						DoWithRetryStub: func(f func() error) error {
							return f()
						},
					},
				},
			},
			recordUpdate: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Cleanup(func() {
				cache.Cache = nil
			})
			i := &ImportIncremental{}
			if tt.recordUpdate {
				tt.fields.app.UpdatedAt = time.Now().Format(time.RFC3339)
				err := tt.args.ctx.Metadata.RecordUpdate(tt.fields.app, tt.fields.space, tt.fields.org)
				require.NoError(t, err)
			}
			err := i.Run(tt.args.ctx)
			require.NoError(t, err)
		})
	}
}
