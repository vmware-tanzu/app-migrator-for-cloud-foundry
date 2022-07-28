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
	"net/url"
	"strconv"
	"sync"

	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/cf"
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/context"
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/process"

	"github.com/cloudfoundry-community/go-cfclient"
)

//DefaultConcurrencyLimit controls the number of apps that are multiplexed across goroutines. Setting this to a large
//number means fewer goroutines will be spun up to handle n number of apps. Setting this to 1 means that each app will
//be exported in its own goroutine. If there are 30 apps, then setting this to 5 will create 6 goroutines to process 5
//apps at a time; setting it to 1 will create 30 goroutines in that same scenario.
const DefaultConcurrencyLimit = 5

type ConcurrentSpaceExporter struct {
	queryResultsProcessor context.QueryResultsProcessor
}

func NewConcurrentSpaceExporter(processor context.QueryResultsProcessor) *ConcurrentSpaceExporter {
	c := &ConcurrentSpaceExporter{
		queryResultsProcessor: processor,
	}
	return c
}

func (c *ConcurrentSpaceExporter) ExportSpace(ctx *context.Context, space cfclient.Space, processor context.ProcessFunc) (<-chan context.ProcessResult, error) {
	listApps := func(page int, collector context.QueryResultsCollector) func() (int, error) {
		return func() (int, error) {
			params := url.Values{
				"q":                []string{"space_guid:" + space.Guid},
				"results-per-page": []string{strconv.Itoa(collector.ResultsPerPage())},
				"page":             []string{strconv.Itoa(page)},
			}
			apps, err := ctx.ExportCFClient.ListAppsByQuery(params)
			if err != nil {
				cfErr := cfclient.CloudFoundryHTTPError{}
				if errors.As(err, &cfErr) {
					if cfErr.StatusCode >= 500 && cfErr.StatusCode <= 599 {
						return 0, cf.ErrRetry
					}
				}
			}
			var wg sync.WaitGroup
			wg.Add(1)
			go func() {
				defer wg.Done()
				for _, app := range apps {
					collector.AddResult(context.QueryResult{Value: app.Name})
				}
			}()

			wg.Wait()

			return len(apps), err
		}
	}

	return c.queryResultsProcessor.ExecutePageQuery(ctx, process.NewAppsQueryResultsCollector(ctx.ConcurrencyLimit), listApps, processor)
}
