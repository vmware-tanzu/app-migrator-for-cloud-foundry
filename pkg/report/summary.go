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

package report

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"
	"text/tabwriter"
	"time"
)

const (
	keyFormat string = "%s.%s.%s"
)

// Result is a specific app migration result: success or failure
type Result struct {
	OrgName   string
	SpaceName string
	AppName   string
	Message   string
}

// Summary is a thread safe sink of execution results for app migrations
type Summary struct {
	results      map[string]string
	successCount int
	failureCount int
	resMutex     sync.RWMutex
	sucMutex     sync.RWMutex
	errMutex     sync.RWMutex
	TableWriter  io.Writer
	duration     time.Duration
}

// NewSummary creates a new initialized summary instance
func NewSummary(w io.Writer) *Summary {
	return &Summary{
		results:     make(map[string]string),
		TableWriter: w,
	}
}

// AppFailureCount is the number of total app failures that have occurred
func (s *Summary) AppFailureCount() int {
	s.errMutex.Lock()
	defer s.errMutex.Unlock()
	return s.failureCount
}

// AppSuccessCount is the number of total app successes that have occurred
func (s *Summary) AppSuccessCount() int {
	s.sucMutex.Lock()
	defer s.sucMutex.Unlock()
	return s.successCount
}

// Results returns a copy of all the app migrations that have occurred
func (s *Summary) Results() []Result {
	s.resMutex.Lock()
	defer s.resMutex.Unlock()

	var r []Result
	for a, m := range s.results {
		app := strings.Split(a, ".")
		r = append(r, Result{
			OrgName:   app[0],
			SpaceName: app[1],
			AppName:   app[2],
			Message:   m,
		})
	}
	sort.Slice(r, func(i, j int) bool {
		var sortedByOrgName, sortedBySpaceName bool

		sortedByOrgName = r[i].OrgName < r[j].OrgName
		if r[i].OrgName == r[j].OrgName {
			sortedBySpaceName = r[i].SpaceName < r[j].SpaceName
			if r[i].SpaceName == r[j].SpaceName {
				return r[i].AppName < r[j].AppName
			}
			return sortedBySpaceName
		}

		return sortedByOrgName
	})

	return r
}

func (s *Summary) Duration() time.Duration {
	return s.duration
}

func (s *Summary) SetDuration(duration time.Duration) {
	s.duration = duration
}

// AddFailedApp adds a failed app along with its error to the summary result
func (s *Summary) AddFailedApp(org, space, app string, err error) {
	if len(app) == 0 {
		return
	}
	s.errMutex.Lock()
	defer s.errMutex.Unlock()
	s.failureCount++

	s.resMutex.Lock()
	defer s.resMutex.Unlock()

	s.results[fmt.Sprintf(keyFormat, org, space, app)] = err.Error()
}

// AddSuccessfulApp adds a successful app and increments the count of successful apps
func (s *Summary) AddSuccessfulApp(org, space, app string) {
	s.sucMutex.Lock()
	defer s.sucMutex.Unlock()
	s.successCount++

	s.resMutex.Lock()
	defer s.resMutex.Unlock()

	s.results[fmt.Sprintf(keyFormat, org, space, app)] = "successful"
}

func (s *Summary) Display() {
	tw := tabwriter.NewWriter(s.TableWriter, 10, 2, 2, ' ', 0)

	// Summary
	_, _ = fmt.Fprintf(tw, "Migration took %v\nSummary: %d successes, %d errors.\n", s.Duration(), s.AppSuccessCount(), s.AppFailureCount())
	fmt.Println()

	// Header
	_, _ = fmt.Fprintln(tw, "Org\tSpace\tApp\tResult")
	fmt.Println()

	for _, f := range s.Results() {
		row := []string{
			f.OrgName,
			f.SpaceName,
			f.AppName,
			strings.Split(f.Message, ":")[0],
		}
		_, _ = fmt.Fprintln(tw, strings.Join(row, "\t"))
	}
	_ = tw.Flush()
	fmt.Println()
}
