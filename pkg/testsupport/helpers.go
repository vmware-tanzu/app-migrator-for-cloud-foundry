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

package testsupport

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/cf"
)

// NewTestCFClient creates a cf client for use in tests
func NewTestCFClient(t *testing.T, s *httptest.Server) cf.Client {
	t.Helper()
	client, err := cf.NewClient(
		&cf.Config{
			Target:      s.URL,
			SSLDisabled: true,
			AccessToken: "some-access-token",
		},
		cf.WithHTTPClient(s.Client()),
		cf.WithRetryPause(3*time.Millisecond),
		cf.WithRetryTimeout(100*time.Millisecond),
	)
	assert.NoError(t, err)
	return client
}

// Convenient HTTP Handlers for Integration Tests

type TestHandlerFunc func(t *testing.T) http.HandlerFunc

func WithHTTPHandler(pattern string, f http.HandlerFunc) func() (string, http.HandlerFunc) {
	return func() (string, http.HandlerFunc) {
		return pattern, f
	}
}

func WithTestHandler(t *testing.T, pattern string, f TestHandlerFunc) func() (string, http.HandlerFunc) {
	return func() (string, http.HandlerFunc) {
		return pattern, f(t)
	}
}

func TestMux(handlers ...func() (string, http.HandlerFunc)) http.Handler {
	mux := http.NewServeMux()

	for _, h := range handlers {
		mux.HandleFunc(h())
	}

	return mux
}

func JSONTestHandler(t *testing.T, data string) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		data, err := ioutil.ReadFile(data)
		assert.NoError(t, err)
		w.Header().Set("Content-Type", "application/json")
		_, err = io.WriteString(w, string(data))
		assert.NoError(t, err)
	}
}

func ImportTestHandler(t *testing.T) http.Handler {
	return TestMux(
		WithTestHandler(t, "/v2/info", InfoTestHandler),
		WithTestHandler(t, "/v2/organizations", OrgsTestHandler),
		WithTestHandler(t, "/v2/organizations/1c0e6074-777f-450e-9abc-c42f39d9b75b", OrgTestHandler),
		WithTestHandler(t, "/v2/spaces", SpacesTestHandler),
		WithTestHandler(t, "/v2/apps", AppsTestHandler),
		WithTestHandler(t, "/v3/apps/6064d98a-95e6-400b-bc03-be65e6d59622", AppsTestHandler),
		WithTestHandler(t, "/v2/stacks", StacksTestHandler),
		WithHTTPHandler("/v2/apps/6064d98a-95e6-400b-bc03-be65e6d59622", func(w http.ResponseWriter, req *http.Request) {
			w.WriteHeader(http.StatusCreated)
			data, err := ioutil.ReadFile("testdata/app.json")
			assert.NoError(t, err)
			_, err = io.WriteString(w, string(data))
			assert.NoError(t, err)
		}),
		WithTestHandler(t, "/v2/private_domains", DomainsTestHandler),
		WithHTTPHandler("/v2/routes", func(w http.ResponseWriter, req *http.Request) {
			data, err := ioutil.ReadFile("testdata/routes.json")
			assert.NoError(t, err)
			w.Header().Set("Content-Type", "application/json")
			_, err = io.WriteString(w, string(data))
			assert.NoError(t, err)
		}),
		WithHTTPHandler("/v2/routes/311d34d1-c045-4853-845f-05132377ad7d/apps/6064d98a-95e6-400b-bc03-be65e6d59622", func(w http.ResponseWriter, req *http.Request) {
			w.WriteHeader(http.StatusCreated)
			data, err := ioutil.ReadFile("testdata/routes.json")
			assert.NoError(t, err)
			_, err = io.WriteString(w, string(data))
			assert.NoError(t, err)
		}),
		WithTestHandler(t, "/v2/service_instances", ServiceInstancesTestHandler),
		WithHTTPHandler("/v2/user_provided_service_instances", func(w http.ResponseWriter, req *http.Request) {
			data, err := ioutil.ReadFile("testdata/user_provided_service_instances.json")
			assert.NoError(t, err)
			_, err = io.WriteString(w, string(data))
			assert.NoError(t, err)
		}),
		WithTestHandler(t, "/v2/apps/6064d98a-95e6-400b-bc03-be65e6d59622/bits", func(t *testing.T) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusCreated)
				data, err := ioutil.ReadFile("testdata/apps/my_org/my_space/my_app.tgz")
				assert.NoError(t, err)
				_, err = io.WriteString(w, string(data))
				assert.NoError(t, err)
			}
		}),
		WithTestHandler(t, "/v2/apps/6064d98a-95e6-400b-bc03-be65e6d59622/droplet/upload", func(t *testing.T) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusCreated)
				var respObj struct {
					Metadata struct {
						GUID string `json:"guid"`
						URL  string `json:"url"`
					} `json:"metadata"`
				}
				bytes, err := json.Marshal(respObj)
				assert.NoError(t, err)
				_, err = io.WriteString(w, string(bytes))
				assert.NoError(t, err)
			}
		}),
		WithTestHandler(t, "/v3/apps/6064d98a-95e6-400b-bc03-be65e6d59622/droplets", func(t *testing.T) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				data, err := ioutil.ReadFile("testdata/v3droplets.json")
				assert.NoError(t, err)
				_, err = io.WriteString(w, string(data))
				assert.NoError(t, err)
			}
		}),
		WithTestHandler(t, "/v3/droplets/585bc3c1-3743-497d-88b0-403ad6b56d16", func(t *testing.T) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusAccepted)
			}
		}),
		WithHTTPHandler("/api/v2/apps/6064d98a-95e6-400b-bc03-be65e6d59622/rules", func(w http.ResponseWriter, req *http.Request) {
			w.WriteHeader(http.StatusCreated)
			data, err := ioutil.ReadFile("testdata/apps/my_org/my_space/my_app_autoscale_rules.json")
			assert.NoError(t, err)
			_, err = io.WriteString(w, string(data))
			assert.NoError(t, err)
		}),
		WithHTTPHandler("/api/v2/apps/6064d98a-95e6-400b-bc03-be65e6d59622", func(w http.ResponseWriter, req *http.Request) {
			w.WriteHeader(http.StatusCreated)
			data, err := ioutil.ReadFile("testdata/apps/my_org/my_space/my_app_autoscale_instances.json")
			assert.NoError(t, err)
			_, err = io.WriteString(w, string(data))
			assert.NoError(t, err)
		}),
		WithHTTPHandler("/api/v2/apps/6064d98a-95e6-400b-bc03-be65e6d59622/scheduled_limit_changes", func(w http.ResponseWriter, req *http.Request) {
			w.WriteHeader(http.StatusCreated)
			data, err := ioutil.ReadFile("testdata/apps/my_org/my_space/my_app_autoscale_schedules.json")
			assert.NoError(t, err)
			_, err = io.WriteString(w, string(data))
			assert.NoError(t, err)
		}),
	)
}

func ExportTestHandler(t *testing.T) http.Handler {
	return TestMux(
		WithTestHandler(t, "/v2/info", InfoTestHandler),
		WithTestHandler(t, "/v2/organizations", OrgsTestHandler),
		WithTestHandler(t, "/v2/organizations/1c0e6074-777f-450e-9abc-c42f39d9b75b", OrgTestHandler),
		WithTestHandler(t, "/v2/spaces", SpacesTestHandler),
		WithTestHandler(t, "/v2/apps", AppsTestHandler),
		WithTestHandler(t, "/v3/apps/6064d98a-95e6-400b-bc03-be65e6d59622", AppsTestHandler),
		WithTestHandler(t, "/v2/apps/6064d98a-95e6-400b-bc03-be65e6d59622", AppTestHandler),
		WithTestHandler(t, "/v3/apps/6064d98a-95e6-400b-bc03-be65e6d59622/packages", PackagesTestHandler),
		WithTestHandler(t, "/v3/packages/752edab0-2147-4f58-9c25-cd72ad8c3561/download", func(t *testing.T) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/plain; charset=utf-8")
				w.Header().Set("X-Content-Type-Options", "nosniff")
				w.Header().Set("Location", "/bits")
				w.WriteHeader(http.StatusFound)
			}
		}),
		WithTestHandler(t, "/bits", func(t *testing.T) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/plain; charset=utf-8")
				_, err := io.WriteString(w, "app bits data")
				assert.NoError(t, err)
			}
		}),
		WithTestHandler(t, "/v2/apps/6064d98a-95e6-400b-bc03-be65e6d59622/droplet/download", func(t *testing.T) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/plain; charset=utf-8")
				w.Header().Set("X-Content-Type-Options", "nosniff")
				w.Header().Set("Location", "/droplet")
				w.WriteHeader(http.StatusFound)
			}
		}),
		WithTestHandler(t, "/droplet", func(t *testing.T) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/plain; charset=utf-8")
				_, err := io.WriteString(w, "droplet data")
				assert.NoError(t, err)
			}
		}),
		WithTestHandler(t, "/v2/spaces/5489e195-c42b-4e61-bf30-323c331ecc01", SpaceTestHandler),
		WithTestHandler(t, "/v2/spaces/5489e195-c42b-4e61-bf30-323c331ecc01/apps", AppsTestHandler),
		WithTestHandler(t, "/v3/apps/6064d98a-95e6-400b-bc03-be65e6d59622/routes", V3RoutesTestHandler),
		WithTestHandler(t, "/v2/apps/6064d98a-95e6-400b-bc03-be65e6d59622/service_bindings", ServiceBindingsTestHandler),
		WithTestHandler(t, "/v2/service_instances/92f0f510-dbb1-4c04-aa7c-28a8dc0797b4", ServiceInstancesTestHandler),
		WithTestHandler(t, "/api/v2/apps/6064d98a-95e6-400b-bc03-be65e6d59622/rules", AutoScalerRulesTestHandler),
		WithTestHandler(t, "/api/v2/apps/6064d98a-95e6-400b-bc03-be65e6d59622", AutoScalerAppInstancesTestHandler),
		WithTestHandler(t, "/api/v2/apps/6064d98a-95e6-400b-bc03-be65e6d59622/scheduled_limit_changes", AutoScalerSchedulesTestHandler),
	)
}

func InfoTestHandler(t *testing.T) http.HandlerFunc {
	return JSONTestHandler(t, "testdata/info.json")
}

func OrgTestHandler(t *testing.T) http.HandlerFunc {
	return JSONTestHandler(t, "testdata/org.json")
}

func OrgsTestHandler(t *testing.T) http.HandlerFunc {
	return JSONTestHandler(t, "testdata/orgs.json")
}

func SpaceTestHandler(t *testing.T) http.HandlerFunc {
	return JSONTestHandler(t, "testdata/space.json")
}

func SpacesTestHandler(t *testing.T) http.HandlerFunc {
	return JSONTestHandler(t, "testdata/spaces.json")
}

func AppTestHandler(t *testing.T) http.HandlerFunc {
	return JSONTestHandler(t, "testdata/app.json")
}

func AppsTestHandler(t *testing.T) http.HandlerFunc {
	return JSONTestHandler(t, "testdata/apps.json")
}

func PackagesTestHandler(t *testing.T) http.HandlerFunc {
	return JSONTestHandler(t, "testdata/v3packages.json")
}

func V3RoutesTestHandler(t *testing.T) http.HandlerFunc {
	return JSONTestHandler(t, "testdata/v3routes.json")
}

func ServiceBindingsTestHandler(t *testing.T) http.HandlerFunc {
	return JSONTestHandler(t, "testdata/service_bindings.json")
}

func ServiceInstancesTestHandler(t *testing.T) http.HandlerFunc {
	return JSONTestHandler(t, "testdata/service_instances.json")
}

func AutoScalerRulesTestHandler(t *testing.T) http.HandlerFunc {
	return JSONTestHandler(t, "testdata/autoscaler_rules.json")
}

func AutoScalerAppInstancesTestHandler(t *testing.T) http.HandlerFunc {
	return JSONTestHandler(t, "testdata/autoscaler_instances.json")
}

func AutoScalerSchedulesTestHandler(t *testing.T) http.HandlerFunc {
	return JSONTestHandler(t, "testdata/autoscaler_schedules.json")
}

func StacksTestHandler(t *testing.T) http.HandlerFunc {
	return JSONTestHandler(t, "testdata/stacks.json")
}

func DomainsTestHandler(t *testing.T) http.HandlerFunc {
	return JSONTestHandler(t, "testdata/stacks.json")
}
