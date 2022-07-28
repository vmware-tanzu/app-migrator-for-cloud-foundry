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

package _import

import (
	"bytes"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/cf/fakes"
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/process"

	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/context"
	contextfakes "github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/context/fakes"

	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/cache"
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/metadata"
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/report"
)

func TestNewConcurrentSpaceImporter(t *testing.T) {
	type args struct {
		resultsPerPage int
		worker         process.DefaultQueryResultsProcessor
	}
	worker := process.NewQueryResultsProcessor(new(fakes.FakeClient))
	tests := []struct {
		name string
		args args
		want *ConcurrentSpaceImporter
	}{
		{
			name: "new concurrent space exporter",
			args: args{resultsPerPage: 1, worker: worker},
			want: &ConcurrentSpaceImporter{queryResultsProcessor: worker},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Cleanup(func() {
				cache.Cache = nil
			})
			if got := NewConcurrentSpaceImporter(tt.args.worker); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewConcurrentSpaceImporter() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_importSpace(t *testing.T) {
	type fields struct {
		processor *contextfakes.FakeQueryResultsProcessor
	}
	type args struct {
		ctx       *context.Context
		processor context.ProcessFunc
		apps      []string
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
					ExecuteQueryStub: func(ctx *context.Context, collector context.QueryResultsCollector, f func(collector context.QueryResultsCollector) (int, error), f2 func(ctx *context.Context, value context.QueryResult) context.ProcessResult) (<-chan context.ProcessResult, error) {
						results := make(chan context.ProcessResult, 1)
						defer close(results)
						results <- context.ProcessResult{Value: "my_app"}
						return results, nil
					},
				},
			},
			args: args{
				ctx: &context.Context{
					ExportDir: filepath.Join(pwd, "../commands/testdata/apps"),
					Metadata:  metadata.NewMetadata(),
					Summary:   report.NewSummary(&bytes.Buffer{}),
					ImportCFClient: &fakes.FakeClient{
						DoWithRetryStub: func(f func() error) error {
							return f()
						},
					},
				},
				processor: func(ctx *context.Context, r context.QueryResult) context.ProcessResult {
					return context.ProcessResult{Value: r.Value}
				},
				apps: []string{"my_app"},
			},
			want: context.ProcessResult{
				Value: "my_app",
			},
			wantErr: false,
		},
		{
			name: "returns error when no apps exist",
			fields: fields{
				processor: &contextfakes.FakeQueryResultsProcessor{
					ExecuteQueryStub: func(ctx *context.Context, collector context.QueryResultsCollector, f func(collector context.QueryResultsCollector) (int, error), f2 func(ctx *context.Context, value context.QueryResult) context.ProcessResult) (<-chan context.ProcessResult, error) {
						_, err := f(collector)
						return nil, err
					},
				},
			},
			args: args{
				ctx: &context.Context{
					ExportDir: filepath.Join(pwd, "testdata/apps"),
					Metadata:  metadata.NewMetadata(),
					Summary:   report.NewSummary(&bytes.Buffer{}),
				},
				apps: []string{},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Cleanup(func() {
				cache.Cache = nil
			})
			p := &ConcurrentSpaceImporter{
				queryResultsProcessor: tt.fields.processor,
			}

			got, err := p.ImportSpace(tt.args.ctx, tt.args.processor, tt.args.apps)
			if tt.wantErr && err != nil {
				assert.EqualError(t, err, "list of apps is empty")
				return
			}
			if (err != nil) != tt.wantErr {
				t.Errorf("exportSpace() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got == nil {
				t.Fatalf("no results returned")
			}
			result := <-got
			if !reflect.DeepEqual(result, tt.want) {
				t.Errorf("exportSpace() got = %v, want %v", result, tt.want)
			}
		})
	}
}
