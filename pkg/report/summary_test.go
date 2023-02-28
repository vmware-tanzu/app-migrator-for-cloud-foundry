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
	"bytes"
	"errors"
	"fmt"
	"github.com/stretchr/testify/assert"
	"io"
	"strings"
	"sync"
	"testing"
)

var errAppError = errors.New("this is an example error")

func TestAddFailedAppWithNoName(t *testing.T) {
	s := NewSummary(&bytes.Buffer{})
	s.AddFailedApp("", "", "", errAppError)
	if s.AppFailureCount() != 0 {
		t.Fatalf("Expected there to be 3 failed apps, but instead got %d", s.AppFailureCount())
	}
}

func TestAddFailedApps(t *testing.T) {
	failedAppNames := []string{"my-org.my-space.my-favorite-app", "my-org.my-space.playground/my-favorite-app", "my-org.my-space.@#$%^&*(UI"}
	s := NewSummary(&bytes.Buffer{})

	for _, n := range failedAppNames {
		app := strings.Split(n, ".")
		s.AddFailedApp(app[0], app[1], app[2], errAppError)
	}

	if s.AppFailureCount() != 3 {
		t.Fatalf("Expected there to be 3 failed apps, but instead got %d", s.AppFailureCount())
	}

	appFailures := s.Results()
	for _, name := range failedAppNames {
		var failure *Result
		for _, f := range appFailures {
			if name == f.OrgName+"."+f.SpaceName+"."+f.AppName {
				failure = &f
				break
			}
		}

		if failure == nil {
			t.Fatalf("Could not find failed app " + name)
		}
		if failure.Message != errAppError.Error() {
			t.Fatalf(fmt.Sprintf("We expected %s error to be: %s but instead we got %s.", name, errAppError, failure.Message))
		}
	}
}

func TestAddFailedAppsConcurrently(t *testing.T) {
	var wg sync.WaitGroup
	s := NewSummary(&bytes.Buffer{})
	for i := 1; i <= 50; i++ {
		wg.Add(1)
		go func(failedAppNum int) {
			failedAppName := fmt.Sprintf("FailedApp%d", failedAppNum)
			s.AddFailedApp("", "", failedAppName, errAppError)
			wg.Done()
		}(i)
	}
	wg.Wait()

	if s.AppFailureCount() != 50 {
		t.Fatalf("Expected there to be 50 failed apps, but instead got %d", s.AppFailureCount())
	}
	if s.AppSuccessCount() != 0 {
		t.Fatalf("Expected there to be 0 successful apps, but instead got %d", s.AppSuccessCount())
	}
}

func TestAddSuccessfulAppsConcurrently(t *testing.T) {
	var wg sync.WaitGroup
	s := NewSummary(&bytes.Buffer{})
	for i := 1; i <= 50; i++ {
		wg.Add(1)
		go func() {
			s.AddSuccessfulApp("", "", "")
			wg.Done()
		}()
	}
	wg.Wait()

	if s.AppFailureCount() != 0 {
		t.Fatalf("Expected there to be 0 failed apps, but instead got %d", s.AppFailureCount())
	}
	if s.AppSuccessCount() != 50 {
		t.Fatalf("Expected there to be 50 successful apps, but instead got %d", s.AppSuccessCount())
	}
}

func TestSummary_Display(t *testing.T) {
	output := &bytes.Buffer{}
	type fields struct {
		TableWriter io.Writer
	}
	tests := []struct {
		name   string
		fields fields
	}{
		{
			name: "prints results table sorted by org, space, and app",
			fields: fields{
				TableWriter: output,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewSummary(tt.fields.TableWriter)
			s.AddSuccessfulApp("blue", "dev", "my-good-app")
			s.AddSuccessfulApp("blue", "dev", "another-good-app")
			s.AddSuccessfulApp("blue", "stage", "my-good-app")
			s.AddFailedApp("red", "dev", "my-bad-app", errAppError)
			s.Display()
			assert.Equal(t, `Migration took 0s
Summary: 3 successes, 1 errors.
Org       Space     App               Result
blue      dev       another-good-app  successful
blue      dev       my-good-app       successful
blue      stage     my-good-app       successful
red       dev       my-bad-app        this is an example error
`, output.String())
		})
	}
}
