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
	"encoding/json"
	"io"
	"io/ioutil"
	"sync"
	"time"

	"github.com/cloudfoundry-community/go-cfclient"
)

// Metadata defines a struct for storing app metadata
// Example of what ^^^ map looks like:
//		 {
//			"pivot-dlohle": { // org
//				"playground": { // space
//					"HOSSEINS-APP": "2021-06-22T20:18:36Z", // app_name : last_updated_timestamp
//					"java-app-test": "2021-06-22T21:16:44Z",
//					"my-favorite-app": "2021-06-22T20:18:36Z",
//					"pm": "2021-06-22T20:18:36Z",
//					"test-nopush": "2021-06-22T20:18:36Z",
//					"timeout-app-test": "2021-06-22T20:18:37Z"
//				}
//			}
//		}
type Metadata struct {
	appLastLocallySeen map[string]map[string]map[string]string
	mutex              sync.RWMutex
}

// NewMetadata creates a new initialized metadata instance
func NewMetadata() *Metadata {
	return &Metadata{
		appLastLocallySeen: make(map[string]map[string]map[string]string),
	}
}

func (m *Metadata) LoadMetadata(data io.Reader) error {
	if m.appLastLocallySeen == nil {
		m.appLastLocallySeen = make(map[string]map[string]map[string]string)
	}

	if data == nil {
		return nil
	}

	fileData, err := ioutil.ReadAll(data)

	if err != nil {
		return err
	}

	if len(fileData) == 0 {
		return nil
	}

	err = json.Unmarshal(fileData, &m.appLastLocallySeen)

	if err != nil {
		return err
	}

	return nil
}

func (m *Metadata) SaveMetadata(data io.Writer) error {

	// pretty-print JSON to file with 4 space identendation
	byteArray, err := json.MarshalIndent(&m.appLastLocallySeen, "", "\t")

	if err != nil {
		return err
	}

	_, err = data.Write(byteArray)

	if err != nil {
		return err
	}

	return nil
}

func (m *Metadata) RecordUpdate(app cfclient.App, space cfclient.Space, org cfclient.Org) error {
	// parse to ensure we're given a proper timestamp
	unixTime, err := time.Parse(time.RFC3339, app.UpdatedAt)

	if err != nil {
		return err
	}

	m.getMapDataSafely(app, space, org)

	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.appLastLocallySeen[org.Name][space.Name][app.Name] = unixTime.UTC().Format(time.RFC3339)

	return nil
}

func (m *Metadata) HasBeenUpdated(app cfclient.App, space cfclient.Space, org cfclient.Org) bool {
	// return TRUE when app is newer than in DB or has same date
	// return FALSE when app is older than in DB
	remotelyUpdatedAt, _ := time.Parse(time.RFC3339, app.UpdatedAt)

	rawLocalTimestamp := m.getMapDataSafely(app, space, org)

	locallyLastSeen, _ := time.Parse(time.RFC3339, rawLocalTimestamp)

	if remotelyUpdatedAt.Equal(locallyLastSeen) {
		return false
	}

	return remotelyUpdatedAt.After(locallyLastSeen)

}

func (m *Metadata) HasNewerLocally(app cfclient.App, space cfclient.Space, org cfclient.Org) bool {
	// return FALSE when remote app is newer than in DB
	// return TRUE when remote app is older than in DB or has same date
	remotelyUpdatedAt, _ := time.Parse(time.RFC3339, app.UpdatedAt)

	rawLocalTimestamp := m.getMapDataSafely(app, space, org)

	locallyLastSeen, _ := time.Parse(time.RFC3339, rawLocalTimestamp)

	if remotelyUpdatedAt.Equal(locallyLastSeen) {
		return false
	}

	return remotelyUpdatedAt.Before(locallyLastSeen)

}

func (m *Metadata) getMapDataSafely(app cfclient.App, space cfclient.Space, org cfclient.Org) string {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	_, exists := m.appLastLocallySeen[org.Name]
	if !exists {
		m.appLastLocallySeen[org.Name] = make(map[string]map[string]string)
	}

	_, exists = m.appLastLocallySeen[org.Name][space.Name]
	if !exists {
		m.appLastLocallySeen[org.Name][space.Name] = make(map[string]string)
	}

	timestamp, exists := m.appLastLocallySeen[org.Name][space.Name][app.Name]
	if !exists {
		return ""
	}

	return timestamp
}
