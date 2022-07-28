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

package process

import (
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/vbauerster/mpb/v7"
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/cf"
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/context"
)

type DefaultQueryResultsProcessor struct {
	cf cf.Client
}

func NewQueryResultsProcessor(cf cf.Client) DefaultQueryResultsProcessor {
	return DefaultQueryResultsProcessor{
		cf: cf,
	}
}

func (w DefaultQueryResultsProcessor) ExecuteQuery(ctx *context.Context, queryResultsCollector context.QueryResultsCollector, query func(collector context.QueryResultsCollector) (int, error), processor func(ctx *context.Context, value context.QueryResult) context.ProcessResult) (<-chan context.ProcessResult, error) {
	var wg sync.WaitGroup
	collectedResults := make(chan context.ProcessResult)

	numOfApps, err := query(queryResultsCollector)
	if err != nil {
		return nil, err
	}

	if numOfApps == 0 {
		queryResultsCollector.Close()
		return nil, errors.New("list of apps is empty")
	}

	results := w.process(ctx, queryResultsCollector, processor, numOfApps)
	if results == nil {
		return collectedResults, nil
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		for r := range results {
			collectedResults <- r
		}
	}()

	if numOfApps == queryResultsCollector.ResultCount() {
		queryResultsCollector.Close()
	}

	go func() {
		wg.Wait()
		close(collectedResults)
	}()

	return collectedResults, nil
}

func (w DefaultQueryResultsProcessor) ExecutePageQuery(ctx *context.Context, queryResultsCollector context.QueryResultsCollector, query func(page int, collector context.QueryResultsCollector) func() (int, error), processor context.ProcessFunc) (<-chan context.ProcessResult, error) {
	var wg sync.WaitGroup
	var page = 1
	collectedResults := make(chan context.ProcessResult)

	start := time.Now()

	for {
		numOfApps, err := query(page, queryResultsCollector)()
		if err != nil {
			return nil, err
		}

		if numOfApps == 0 {
			queryResultsCollector.Close()
			break
		}

		results := w.process(ctx, queryResultsCollector, processor, numOfApps)
		if results == nil {
			break
		}

		wg.Add(1)
		go func(processResults <-chan context.ProcessResult, page int) {
			defer wg.Done()
			resultCount := 0
			for processResult := range processResults {
				collectedResults <- processResult
				resultCount = resultCount + 1
			}
		}(results, page)

		page++
	}

	go func() {
		wg.Wait()
		close(collectedResults)
		ctx.Summary.SetDuration(time.Since(start))
	}()

	return collectedResults, nil
}

func (w DefaultQueryResultsProcessor) process(ctx *context.Context, queryResultsCollector context.QueryResultsCollector, processor context.ProcessFunc, numOfApps int) <-chan context.ProcessResult {
	resultsPerPage := queryResultsCollector.ResultsPerPage()
	if resultsPerPage == 0 {
		return nil
	}
	batchSize := resultsPerPage
	workerChan := make(chan context.QueryResult, batchSize)

	go func() {
		defer close(workerChan)
		for result := range queryResultsCollector.GetResults() {
			workerChan <- result
		}
	}()

	return w.processResults(ctx, workerChan, processor, numOfApps, batchSize)
}

func (w DefaultQueryResultsProcessor) processResults(ctx *context.Context, in <-chan context.QueryResult, processor context.ProcessFunc, numOfApps, batchSize int) <-chan context.ProcessResult {
	results := make(chan context.ProcessResult, batchSize)
	var wg sync.WaitGroup
	p := ctx.Progress
	if p == nil {
		p = mpb.New(
			mpb.WithWidth(64),
		)
	}
	ctx.Progress = p

	total := batchSize
	if numOfApps < batchSize {
		total = numOfApps
	}
	wg.Add(total)

	for i := 0; i < total; i++ {
		go func() {
			defer wg.Done()
			for v := range in {
				results <- processor(ctx, v)
			}
		}()
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	return results
}

type PageQueryResultsCollector struct {
	Results chan context.QueryResult
	Count   uint32
	PerPage int
}

func NewAppsQueryResultsCollector(resultsPerPage int) *PageQueryResultsCollector {
	return &PageQueryResultsCollector{
		Results: make(chan context.QueryResult, resultsPerPage),
		Count:   0,
		PerPage: resultsPerPage,
	}
}

func (a *PageQueryResultsCollector) ResultsPerPage() int {
	return a.PerPage
}

func (a *PageQueryResultsCollector) AddResult(result context.QueryResult) {
	atomic.AddUint32(&a.Count, 1)
	a.Results <- result
}

func (a *PageQueryResultsCollector) ResultCount() int {
	return int(a.Count)
}

func (a *PageQueryResultsCollector) GetResults() <-chan context.QueryResult {
	return a.Results
}

func (a *PageQueryResultsCollector) Close() {
	close(a.Results)
}
