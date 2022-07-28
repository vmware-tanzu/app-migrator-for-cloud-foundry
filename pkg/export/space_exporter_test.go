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
	"bytes"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	contextfakes "github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/context/fakes"
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/metadata"
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/report"

	"github.com/cloudfoundry-community/go-cfclient"

	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/cache"
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/cf/fakes"
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/context"
)

func TestNewConcurrentSpaceExporter(t *testing.T) {
	type args struct {
		resultsPerPage int
		processor      *contextfakes.FakeQueryResultsProcessor
	}
	processor := &contextfakes.FakeQueryResultsProcessor{
		ExecutePageQueryStub: func(ctx *context.Context, collector context.QueryResultsCollector, f func(page int, collector context.QueryResultsCollector) func() (int, error), processFunc context.ProcessFunc) (<-chan context.ProcessResult, error) {
			results := make(chan context.ProcessResult, 1)
			defer close(results)
			results <- context.ProcessResult{Value: "my_app"}
			return results, nil
		},
	}
	tests := []struct {
		name string
		args args
		want *ConcurrentSpaceExporter
	}{
		{
			name: "new concurrent space exporter",
			args: args{resultsPerPage: 1, processor: processor},
			want: &ConcurrentSpaceExporter{queryResultsProcessor: processor},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Cleanup(func() {
				cache.Cache = nil
			})
			if got := NewConcurrentSpaceExporter(tt.args.processor); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewConcurrentSpaceExporter() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_exportSpace(t *testing.T) {
	type fields struct {
		processor *contextfakes.FakeQueryResultsProcessor
	}
	type args struct {
		ctx       *context.Context
		space     cfclient.Space
		processor func(ctx *context.Context, r context.QueryResult) context.ProcessResult
	}
	pwd, _ := os.Getwd()
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    context.ProcessResult
		wantErr bool
		handler http.Handler
	}{
		{
			name: "returns an app",
			fields: fields{
				processor: &contextfakes.FakeQueryResultsProcessor{
					ExecutePageQueryStub: func(ctx *context.Context, collector context.QueryResultsCollector, f func(page int, collector context.QueryResultsCollector) func() (int, error), processFunc context.ProcessFunc) (<-chan context.ProcessResult, error) {
						results := make(chan context.ProcessResult, 1)
						defer close(results)
						results <- context.ProcessResult{Value: "my_app"}
						return results, nil
					},
				},
			},
			args: args{
				ctx: &context.Context{
					ExportDir:          filepath.Join(pwd, "testdata/apps"),
					Metadata:           metadata.NewMetadata(),
					Summary:            report.NewSummary(&bytes.Buffer{}),
					DropletExporter:    NewDropletExporter(),
					ManifestExporter:   NewManifestExporter(),
					AutoScalerExporter: NewAutoScalerExporter(),
					ExportCFClient: &fakes.FakeClient{
						ListAppsByQueryStub: func(values url.Values) ([]cfclient.App, error) {
							return []cfclient.App{{Name: "my_app"}}, nil
						},
					},
				},
				space: cfclient.Space{
					Name: "my_space",
				},
				processor: func(ctx *context.Context, r context.QueryResult) context.ProcessResult {
					return context.ProcessResult{Value: r.Value}
				},
			},
			want: context.ProcessResult{
				Value: "my_app",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Cleanup(func() {
				cache.Cache = nil
			})
			p := &ConcurrentSpaceExporter{
				queryResultsProcessor: tt.fields.processor,
			}

			got, err := p.ExportSpace(tt.args.ctx, tt.args.space, tt.args.processor)
			if (err != nil) != tt.wantErr {
				t.Errorf("exportSpace() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			result := <-got
			if !reflect.DeepEqual(result, tt.want) {
				t.Errorf("exportSpace() got = %v, want %v", result, tt.want)
			}
		})
	}
}
