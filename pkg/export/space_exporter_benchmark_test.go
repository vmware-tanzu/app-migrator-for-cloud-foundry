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
	"github.com/stretchr/testify/assert"
	"net/url"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/cf/fakes"
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/context"
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/metadata"
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/process"
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/report"

	"github.com/cloudfoundry-community/go-cfclient"
)

func BenchmarkExportSpace1(b *testing.B)   { benchmarkExportSpace(1, b) }
func BenchmarkExportSpace20(b *testing.B)  { benchmarkExportSpace(20, b) }
func BenchmarkExportSpace100(b *testing.B) { benchmarkExportSpace(100, b) }

func benchmarkExportSpace(i int, b *testing.B) {
	var (
		ctx       *context.Context
		space     cfclient.Space
		proc      context.ProcessFunc
		numOfApps int32
	)
	pwd, _ := os.Getwd()
	for n := 0; n < b.N; n++ {
		atomic.StoreInt32(&numOfApps, int32(i))
		ctx = &context.Context{
			ExportDir:          filepath.Join(pwd, "testdata/apps"),
			Metadata:           metadata.NewMetadata(),
			Summary:            report.NewSummary(&bytes.Buffer{}),
			DropletExporter:    NewDropletExporter(),
			ManifestExporter:   NewManifestExporter(),
			AutoScalerExporter: NewAutoScalerExporter(),
			ExportCFClient: &fakes.FakeClient{
				ListAppsByQueryStub: func(values url.Values) ([]cfclient.App, error) {
					n := numOfApps
					if n > int32(ctx.ConcurrencyLimit) {
						n = int32(ctx.ConcurrencyLimit)
					}
					return createApps(n), nil
				},
			},
			ConcurrencyLimit: 5,
		}
		space = cfclient.Space{
			Name: "my_space",
		}
		proc = func(ctx *context.Context, r context.QueryResult) context.ProcessResult {
			atomic.AddInt32(&numOfApps, -1)
			return context.ProcessResult{Value: r.Value}
		}
		queryResultsCollector := process.NewAppsQueryResultsCollector(ctx.ConcurrencyLimit)
		p := &ConcurrentSpaceExporter{
			queryResultsProcessor: process.NewQueryResultsProcessor(false),
			queryResultsCollector: queryResultsCollector,
		}
		_, err := p.ExportSpace(ctx, space, proc)
		assert.NoError(b, err)
	}
}
