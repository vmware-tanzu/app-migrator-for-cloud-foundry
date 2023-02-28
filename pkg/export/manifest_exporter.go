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
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path"

	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/context"

	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/cf"

	"github.com/cloudfoundry-community/go-cfclient"
	"gopkg.in/yaml.v2"
)

type DefaultManifestExporter struct {
}

type AppManifest struct {
	Applications []Application `yaml:"applications"`
}

type Application struct {
	Name       string   `yaml:"name"`
	Buildpacks []string `yaml:"buildpacks"`
	Command    string   `yaml:"command"`
	DiskQuota  string   `yaml:"disk_quota"`
	Docker     struct {
		Image    string `yaml:"image,omitempty"`
		Username string `yaml:"username,omitempty"`
	} `yaml:"docker,omitempty"`
	Env                     map[string]interface{} `yaml:"env"`
	HealthCheckType         string                 `yaml:"health-check-type"`
	HealthCheckHTTPEndpoint string                 `yaml:"health-check-http-endpoint,omitempty"`
	Instances               int64                  `yaml:"instances"`
	Memory                  string                 `yaml:"memory"`
	NoRoute                 bool                   `yaml:"no-route,omitempty"`
	Routes                  []struct {
		Route string `yaml:"route,omitempty"`
	} `yaml:"routes,omitempty"`
	Services []string `yaml:"services"`
	Stack    string   `yaml:"stack"`
	Timeout  int64    `yaml:"timeout,omitempty"`
}

func NewManifestExporter() *DefaultManifestExporter {
	return &DefaultManifestExporter{}
}

func (m *DefaultManifestExporter) ExportAppManifest(ctx *context.Context, org cfclient.Org, space cfclient.Space, app cfclient.App, appExportDir string) error {
	ctx.Logger.Infof("Writing app manifest for %s/%s/%s", org.Name, space.Name, app.Name)

	manifestApp := Application{}
	manifestApp.Name = app.Name
	manifestApp.Env = app.Environment
	manifestApp.HealthCheckType = app.HealthCheckType
	manifestApp.HealthCheckHTTPEndpoint = app.HealthCheckHttpEndpoint
	manifestApp.Instances = int64(app.Instances)
	manifestApp.Command = app.Command
	manifestApp.Memory = getSizeString(int64(app.Memory))
	manifestApp.DiskQuota = getSizeString(int64(app.DiskQuota))
	manifestApp.Timeout = int64(app.HealthCheckTimeout)

	var (
		resp *http.Response
		err  error
	)
	err = ctx.ExportCFClient.DoWithRetry(func() error {
		req := ctx.ExportCFClient.NewRequest(http.MethodGet, fmt.Sprintf("/v3/apps/%s", app.Guid))
		resp, err = ctx.ExportCFClient.DoRequest(req)
		if err == nil {
			if resp.StatusCode >= 500 && resp.StatusCode <= 599 {
				defer resp.Body.Close()
				return cf.ErrRetry
			}
		}
		return err
	})
	if err != nil {
		cfErr := &cfclient.CloudFoundryHTTPError{}
		if errors.As(err, cfErr) {
			if cfErr.StatusCode == http.StatusNotFound {
				return nil
			}
		}
		return err
	}
	defer resp.Body.Close()

	var v3app cfclient.V3App
	if err = json.NewDecoder(resp.Body).Decode(&v3app); err != nil {
		return err
	}

	manifestApp.Buildpacks = v3app.Lifecycle.BuildpackData.Buildpacks
	if v3app.Lifecycle.Type == "docker" {
		manifestApp.Buildpacks = nil
		manifestApp.Docker = struct {
			Image    string "yaml:\"image,omitempty\""
			Username string "yaml:\"username,omitempty\""
		}{
			Image:    app.DockerImage,
			Username: app.DockerCredentials.Username,
		}
	}

	manifestApp.Stack = v3app.Lifecycle.BuildpackData.Stack

	var routeResp *http.Response
	err = ctx.ExportCFClient.DoWithRetry(func() error {
		var err error
		req := ctx.ExportCFClient.NewRequest(http.MethodGet, fmt.Sprintf("/v3/apps/%s/routes", app.Guid))
		routeResp, err = ctx.ExportCFClient.DoRequest(req)
		if err == nil {
			if routeResp.StatusCode >= 500 && routeResp.StatusCode <= 599 {
				defer routeResp.Body.Close()
				return cf.ErrRetry
			}
		}
		return err
	})
	if err != nil {
		return err
	}

	defer routeResp.Body.Close()

	var routesResponse = struct {
		Resources []struct {
			URL string `json:"url"`
		} `json:"resources"`
	}{}

	if err = json.NewDecoder(routeResp.Body).Decode(&routesResponse); err != nil {
		return err
	}

	manifestApp.NoRoute = len(routesResponse.Resources) == 0
	manifestApp.Routes = make([]struct {
		Route string `yaml:"route,omitempty"`
	}, 0)

	var routes []string
	for _, route := range routesResponse.Resources {
		routes = append(routes, route.URL)
	}
	routeMapper := &routeMapper{
		DomainsToAdd:     ctx.DomainsToAdd,
		DomainsToReplace: ctx.DomainsToReplace,
	}
	adjustedRoutes := routeMapper.AdjustRoutes(routes)
	for _, adjustedRoute := range adjustedRoutes {
		manifestApp.Routes = append(manifestApp.Routes, struct {
			Route string `yaml:"route,omitempty"`
		}{adjustedRoute})
	}

	var sbResp *http.Response
	err = ctx.ExportCFClient.DoWithRetry(func() error {
		req := ctx.ExportCFClient.NewRequest(http.MethodGet, fmt.Sprintf("/v2/apps/%s/service_bindings", app.Guid))
		sbResp, err = ctx.ExportCFClient.DoRequest(req)
		if err == nil {
			if sbResp.StatusCode >= 500 && sbResp.StatusCode <= 599 {
				defer sbResp.Body.Close()
				return cf.ErrRetry
			}
		}
		return err
	})
	if err != nil {
		return err
	}
	defer sbResp.Body.Close()

	var bindings cfclient.ServiceBindingsResponse
	if err = json.NewDecoder(sbResp.Body).Decode(&bindings); err != nil {
		return err
	}

	manifestApp.Services = make([]string, 0, len(bindings.Resources))
	for _, binding := range bindings.Resources {
		var siResp *http.Response
		err := ctx.ExportCFClient.DoWithRetry(func() error {
			req := ctx.ExportCFClient.NewRequest(http.MethodGet, binding.Entity.ServiceInstanceUrl)
			siResp, err = ctx.ExportCFClient.DoRequest(req)
			if err == nil {
				if siResp.StatusCode >= 500 && siResp.StatusCode <= 599 {
					defer siResp.Body.Close()
					return cf.ErrRetry
				}
			}
			return err
		})
		if err != nil {
			return err
		}
		defer siResp.Body.Close()

		var si cfclient.ServiceInstanceResource
		if err = json.NewDecoder(siResp.Body).Decode(&si); err != nil {
			if siResp != nil && siResp.Body != nil {
				siResp.Body.Close()
			}

			return err
		}
		siResp.Body.Close()

		manifestApp.Services = append(manifestApp.Services, si.Entity.Name)
	}

	manifestFilePath := path.Join(appExportDir, getAppFileName(app.Name)+"_manifest.yml")
	manifestFile, err := os.Create(manifestFilePath)
	if err != nil {
		return err
	}
	defer manifestFile.Close()

	return yaml.NewEncoder(manifestFile).Encode(AppManifest{Applications: []Application{manifestApp}})
}

func getSizeString(size int64) string {
	suffix := "M"
	if size >= 1024 {
		size = size / 1024
		suffix = "G"
	}

	return fmt.Sprintf("%d%s", size, suffix)
}
