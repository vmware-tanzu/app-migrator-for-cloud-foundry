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
	"fmt"
	"github.com/stretchr/testify/assert"
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
		collector      *contextfakes.FakeQueryResultsCollector
	}
	processor := &contextfakes.FakeQueryResultsProcessor{
		ExecutePageQueryStub: func(ctx *context.Context, collector context.QueryResultsCollector, f func(page int, collector context.QueryResultsCollector) func() (int, error), processFunc context.ProcessFunc) (<-chan context.ProcessResult, error) {
			results := make(chan context.ProcessResult, 1)
			defer close(results)
			results <- context.ProcessResult{Value: "my_app"}
			return results, nil
		},
	}
	collector := &contextfakes.FakeQueryResultsCollector{
		GetResultsStub: func() <-chan context.QueryResult {
			results := make(chan context.QueryResult, 1)
			return results
		},
	}
	tests := []struct {
		name string
		args args
		want *ConcurrentSpaceExporter
	}{
		{
			name: "new concurrent space exporter",
			args: args{resultsPerPage: 1, processor: processor, collector: collector},
			want: &ConcurrentSpaceExporter{queryResultsProcessor: processor, queryResultsCollector: collector},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Cleanup(func() {
				cache.Cache = nil
			})
			if got := NewConcurrentSpaceExporter(tt.args.processor, tt.args.collector); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewConcurrentSpaceExporter() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_exportSpace(t *testing.T) {
	numOfApps := int32(5)
	type fields struct {
		processor *contextfakes.FakeQueryResultsProcessor
	}
	type args struct {
		ctx     *context.Context
		space   cfclient.Space
		process context.ProcessFunc
	}
	pwd, _ := os.Getwd()
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []context.ProcessResult
		wantErr bool
		handler http.Handler
	}{
		{
			name: "exports an app",
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
							return createApps(numOfApps), nil
						},
					},
				},
				space: cfclient.Space{
					Name: "my_space",
				},
				process: func(ctx *context.Context, r context.QueryResult) context.ProcessResult {
					return context.ProcessResult{Value: r.Value}
				},
			},
			want:    []context.ProcessResult{{Value: "my_app"}},
			wantErr: false,
		},
		{
			name: "exports multiple apps",
			fields: fields{
				processor: &contextfakes.FakeQueryResultsProcessor{
					ExecutePageQueryStub: func(ctx *context.Context, collector context.QueryResultsCollector, f func(page int, collector context.QueryResultsCollector) func() (int, error), proc context.ProcessFunc) (<-chan context.ProcessResult, error) {
						results := make(chan context.ProcessResult, numOfApps)
						defer close(results)
						for n := 0; n < int(numOfApps); n++ {
							results <- context.ProcessResult{Value: fmt.Sprintf("%s_%d", "my_app", n)}
						}
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
							return createApps(numOfApps), nil
						},
					},
				},
				space: cfclient.Space{
					Name: "my_space",
				},
				process: func(ctx *context.Context, r context.QueryResult) context.ProcessResult {
					return context.ProcessResult{Value: r.Value}
				},
			},
			want:    createResults(numOfApps),
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
			got, err := p.ExportSpace(tt.args.ctx, tt.args.space, tt.args.process)
			if (err != nil) != tt.wantErr {
				t.Errorf("exportSpace() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.Equal(t, len(tt.want), len(got))
			i := 0
			for r := range got {
				if !reflect.DeepEqual(r, tt.want[i]) {
					t.Errorf("exportSpace() got = %v, want %v", r, tt.want[i])
				}
				i++
			}
		})
	}
}

func createResults(count int32) []context.ProcessResult {
	var results []context.ProcessResult
	for i := 0; i < int(count); i++ {
		results = append(results, context.ProcessResult{Value: newAppName(i)})
	}
	return results
}

func createApps(count int32) []cfclient.App {
	var apps []cfclient.App
	for i := 0; i < int(count); i++ {
		apps = append(apps, cfclient.App{Name: newAppName(i)})
	}
	return apps
}

func newAppName(i int) string {
	return fmt.Sprintf("%s_%d", "my_app", i)
}
