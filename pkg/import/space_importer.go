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
	"errors"
	"path/filepath"
	"strings"
	"sync"

	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/context"
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/process"
)

type ConcurrentSpaceImporter struct {
	queryResultsProcessor context.QueryResultsProcessor
}

func NewConcurrentSpaceImporter(processor context.QueryResultsProcessor) *ConcurrentSpaceImporter {
	c := &ConcurrentSpaceImporter{
		queryResultsProcessor: processor,
	}
	return c
}

//ImportSpace imports all the apps in a given space
func (c *ConcurrentSpaceImporter) ImportSpace(ctx *context.Context, processor context.ProcessFunc, apps []string) (<-chan context.ProcessResult, error) {
	queryFunc := func(collector context.QueryResultsCollector) (int, error) {
		var wg sync.WaitGroup
		if len(apps) == 0 {
			return 0, errors.New("list of apps is empty")
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			for _, file := range apps {
				appName := strings.TrimSuffix(file, "_manifest.yml")
				collector.AddResult(context.QueryResult{Value: filepath.Base(appName)})
			}
		}()

		wg.Wait()

		return len(apps), nil
	}

	return c.queryResultsProcessor.ExecuteQuery(ctx, process.NewAppsQueryResultsCollector(len(apps)), queryFunc, processor)
}
