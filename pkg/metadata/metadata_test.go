//go:build !integration || all
// +build !integration all

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

package metadata

import (
	"bytes"
	"io/ioutil"
	"reflect"
	"strings"
	"testing"

	"github.com/cloudfoundry-community/go-cfclient"
)

func TestLoadMetadata(t *testing.T) {

	exampleBasicJSON := strings.NewReader(`{
		"my-organization": {
			"my-space": {
				"my-favorite-app1": "2021-06-21T19:16:18Z",
				"my-favorite-app2": "2021-06-21T16:12:05Z"
			}
		}
	}`)

	exampleMultipleOrgsSpacesJSON := strings.NewReader(`{
		"my-organization": {
			"my-space": {
				"my-favorite-app1": "2021-06-21T19:16:18Z",
				"my-favorite-app2": "2021-06-21T16:12:05Z"
			},
			"face-book": {
				"my-favorite-app1": "2021-06-21T19:16:18Z",
				"my-favorite-app2": "2021-06-21T16:12:05Z"
			}
		},
		"my-disorganization": {
			"my-space": {
				"my-favorite-app1": "2021-06-21T19:16:18Z"
			}
		}
	}`)

	exampleBadJSON := strings.NewReader(`{
		"my-organization": {
			"my-space": {
				"my-favor19:16:18Z",
				"my-favorite-app2": "2021-06-21T16:12:05Z"
			}
		}
	}`)

	exampleNoJSON := strings.NewReader(``)

	m := Metadata{}

	// test corrupt JSON
	err := m.LoadMetadata(exampleBadJSON)

	if err == nil {
		t.Fatalf("We fed LoadMetadata() bad JSON data and it didn't error out.")
	}

	// test missing file
	err = m.LoadMetadata(nil)

	if err != nil {
		t.Fatalf("We fed LoadMetadata() nil and it errored out, it should instead initialize an empty Metadata array.")
	}

	// test empty JSON
	err = m.LoadMetadata(exampleNoJSON)

	if err != nil {
		t.Fatalf("We fed LoadMetadata() no JSON data and it errored out. It should initialize an empty Metadata array.")
	}

	// test basic JSON
	err = m.LoadMetadata(exampleBasicJSON)

	if err != nil {
		t.Fatalf("Method LoadMetadata() execution failed with error %s", err)
	}

	expectedResults := map[string]map[string]map[string]string{
		"my-organization": {
			"my-space": {
				"my-favorite-app1": "2021-06-21T19:16:18Z",
				"my-favorite-app2": "2021-06-21T16:12:05Z",
			},
		},
	}

	matches := reflect.DeepEqual(m.appLastLocallySeen, expectedResults)

	if !matches {
		t.Fatalf("We expected LoadMetadata() to properly parse JSON. We expected the data structure to look like this: \n %v\n\n but instead we got:\n\n %v", expectedResults, m.appLastLocallySeen)
	}

	// test complex JSON
	m = Metadata{}
	err = m.LoadMetadata(exampleMultipleOrgsSpacesJSON)

	if err != nil {
		t.Fatalf("Method LoadMetadata() execution failed with error %s", err)
	}

	expectedResults = map[string]map[string]map[string]string{
		"my-organization": {
			"my-space": {
				"my-favorite-app1": "2021-06-21T19:16:18Z",
				"my-favorite-app2": "2021-06-21T16:12:05Z",
			},
			"face-book": {
				"my-favorite-app1": "2021-06-21T19:16:18Z",
				"my-favorite-app2": "2021-06-21T16:12:05Z",
			},
		},
		"my-disorganization": {
			"my-space": {
				"my-favorite-app1": "2021-06-21T19:16:18Z",
			},
		},
	}

	matches = reflect.DeepEqual(m.appLastLocallySeen, expectedResults)

	if !matches {
		t.Fatalf("We expected LoadMetadata() to properly parse JSON. We expected the data structure to look like this: \n %v\n\n but instead we got:\n\n %v", expectedResults, m.appLastLocallySeen)
	}
}

func TestNewerThanBaseComparison(t *testing.T) {

	m := Metadata{}

	m.appLastLocallySeen = map[string]map[string]map[string]string{
		"my-organization": {
			"my-space": {
				"my-favorite-app1": "2021-06-21T19:16:18Z",
				"my-favorite-app2": "2021-06-21T16:12:05Z",
			},
		},
	}

	exampleSpace := &cfclient.Space{
		Name: "my-space",
	}

	exampleOrg := &cfclient.Org{
		Name: "my-organization",
	}

	exampleUpdatedApp := &cfclient.App{
		Name:      "my-favorite-app1",
		UpdatedAt: "2021-06-21T20:16:18Z",
	}

	if m.HasBeenUpdated(*exampleUpdatedApp, *exampleSpace, *exampleOrg) != true {
		t.Fatalf("Expected IsNewest to report false, as %s is different than %s", m.appLastLocallySeen["1ccb1a38-8ce1-48be-9340-569671217f6d"], exampleUpdatedApp.UpdatedAt)
	}

	exampleNonExistantApp := &cfclient.App{
		Name:      "my-favorite-app-non-existant",
		UpdatedAt: "2021-06-21T20:16:18Z",
	}

	if m.HasBeenUpdated(*exampleNonExistantApp, *exampleSpace, *exampleOrg) != true {
		t.Fatalf("Expected IsNewest to report false when app does not exist in database.")
	}

	exampleNonUpdatedApp := &cfclient.App{
		Name:      "my-favorite-app2",
		UpdatedAt: "2021-06-21T16:12:05Z",
	}

	if m.HasBeenUpdated(*exampleNonUpdatedApp, *exampleSpace, *exampleOrg) != false {
		t.Fatalf("Expected IsNewest to report true, as %s isn't different than %s", m.appLastLocallySeen["1ccb1a38-8ce1-48be-9340-569671217f6d"], exampleNonUpdatedApp.UpdatedAt)
	}

}

func TestRecordingTimestampUpdates(t *testing.T) {

	m := &Metadata{}

	m.appLastLocallySeen = map[string]map[string]map[string]string{
		"my-organization": {
			"my-space": {
				"my-favorite-app1": "2021-06-21T19:16:18Z",
				"my-favorite-app2": "2021-06-21T16:12:05Z",
			},
		},
	}

	exampleSpace := &cfclient.Space{
		Name: "my-space",
	}

	exampleOrg := &cfclient.Org{
		Name: "my-organization",
	}

	exampleUpdatedApp := &cfclient.App{
		Name:      "my-favorite-app1",
		UpdatedAt: "2021-06-22T20:16:18Z",
	}

	// test existing app in local DB getting updated timestamp
	err := m.RecordUpdate(*exampleUpdatedApp, *exampleSpace, *exampleOrg)

	if err != nil {
		t.Fatalf("Method RecordUpdate() execution failed with error %s", err)
	}

	results := m.appLastLocallySeen[exampleOrg.Name][exampleSpace.Name][exampleUpdatedApp.Name]

	if results != exampleUpdatedApp.UpdatedAt {
		t.Fatalf("After calling RecordUpdate() the timestamp has not changed. Expected %s and it's %s instead.", exampleUpdatedApp.UpdatedAt, m.appLastLocallySeen["my-organization"]["my-space"]["my-favorite-app1"])
	}

	// test trying to update app with a bad timestamp
	exampleBadInputApp := &cfclient.App{
		Name:      "my-favorite-app1",
		UpdatedAt: "2021-0616:18Z",
	}

	err = m.RecordUpdate(*exampleBadInputApp, *exampleSpace, *exampleOrg)

	if err == nil {
		t.Fatalf("We fed RecordUpdate() with bad timestamp format and it didn't error out.")
	}

	// test trying to update app not yet seen before
	exampleFirstSeenApp := &cfclient.App{
		Name:      "my-favorite-app-ive-never-seen",
		UpdatedAt: "2021-06-22T20:16:18Z",
	}

	err = m.RecordUpdate(*exampleFirstSeenApp, *exampleSpace, *exampleOrg)

	if err != nil {
		t.Fatalf("We fed RecordUpdate() with a new application and it errored out with: %s", err)
	}

	results = m.appLastLocallySeen[exampleOrg.Name][exampleSpace.Name][exampleFirstSeenApp.Name]

	if results != exampleFirstSeenApp.UpdatedAt {
		t.Fatalf("We fed RecordUpdate() a new application and it didn't save the timestamp properly. We got %s but we expected %s", m.appLastLocallySeen[exampleFirstSeenApp.Guid], exampleFirstSeenApp.UpdatedAt)
	}

	// test trying to update an app not yet seen before in an org and space not yet seen before either
	exampleFirstSeenSpace := &cfclient.Space{
		Name: "my-book",
	}

	exampleFirstSeenOrg := &cfclient.Org{
		Name: "my-uwuanization",
	}
	err = m.RecordUpdate(*exampleFirstSeenApp, *exampleFirstSeenSpace, *exampleFirstSeenOrg)

	if err != nil {
		t.Fatalf("We fed RecordUpdate() with a new application and it errored out with: %s", err)
	}

	results = m.appLastLocallySeen[exampleFirstSeenOrg.Name][exampleFirstSeenSpace.Name][exampleFirstSeenApp.Name]

	if results != exampleFirstSeenApp.UpdatedAt {
		t.Fatalf("We fed RecordUpdate() a new application and it didn't save the timestamp properly. We got %s but we expected %s", m.appLastLocallySeen[exampleFirstSeenApp.Guid], exampleFirstSeenApp.UpdatedAt)
	}

}

func TestSaveMetadata(t *testing.T) {

	m := Metadata{}

	m.appLastLocallySeen = map[string]map[string]map[string]string{
		"my-organization": {
			"my-space": {
				"my-favorite-app1": "2021-06-21T19:16:18Z",
				"my-favorite-app2": "2021-06-21T16:12:05Z",
			},
		},
	}

	var saveOutput bytes.Buffer

	err := m.SaveMetadata(&saveOutput)

	if err != nil {
		t.Fatalf("Method LoadMetadata() execution failed with error %s", err)
	}

	fileData, _ := ioutil.ReadAll(&saveOutput)

	expectedResults := []byte(`{
	"my-organization": {
		"my-space": {
			"my-favorite-app1": "2021-06-21T19:16:18Z",
			"my-favorite-app2": "2021-06-21T16:12:05Z"
		}
	}
}`)

	if !bytes.Equal(fileData, expectedResults) {
		t.Fatalf("Expected the output of SaveMetadata: %s to match the expected result %s", fileData, expectedResults)
	}

}
