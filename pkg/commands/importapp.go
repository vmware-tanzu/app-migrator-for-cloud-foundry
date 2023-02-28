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

package commands

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	appcontext "github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/context"

	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/cache"
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/cf"
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/export"

	"github.com/cloudfoundry-community/go-cfclient"
	"gopkg.in/yaml.v2"
)

type ImportApp struct {
	ImportSpace
	Sequence Sequence
	AppName  string `help:"the app to import" short:"a" env:"CF_APP_NAME"`
	appGUID  string
	AppCount int
}

func (i *ImportApp) SetOrgName(name string) {
	i.Org = name
}

func (i *ImportApp) SetSpaceName(name string) {
	i.Space = name
}

func (i *ImportApp) SetAppName(name string) {
	i.AppName = name
}

func (i *ImportApp) Run(ctx *appcontext.Context) error {

	if i.Sequence == nil {
		i.Sequence = NewImportAppSequence(i)
	}

	if _, err := i.Sequence.Run(ctx, nil); err != nil {
		ctx.Logger.Errorf("Error occurred importing app %s/%s/%s: %s", i.Org, i.Space, i.AppName, err)
		return err
	}

	return nil
}

func NewImportAppSequence(i *ImportApp) Sequence {
	return RunSequence(
		fmt.Sprintf("\x1b[31m%v\x1b[0m", i.AppName),
		i.AppName,
		StepWithProgressBar(
			func(ctx *appcontext.Context, r Result) (Result, error) {
				var err error
				i.AppName, err = i.getAppNameFromManifest(ctx)
				return nil, err
			},
			"Getting name from manifest",
		),
		StepWithProgressBar(
			func(ctx *appcontext.Context, r Result) (Result, error) {
				err := i.createApp(ctx)
				return nil, err
			},
			"Creating app",
		),
		StepWithProgressBar(
			func(ctx *appcontext.Context, r Result) (Result, error) {
				err := i.uploadBlob(ctx)
				return nil, err
			},
			"Uploading blob",
		),
		StepWithProgressBar(
			func(ctx *appcontext.Context, r Result) (Result, error) {
				err := i.applyAutoscalerRules(ctx)
				return nil, err
			},
			"Applying AutoScaler rules",
		),
		StepWithProgressBar(
			func(ctx *appcontext.Context, r Result) (Result, error) {
				err := i.applyAutoscalerInstances(ctx)
				return nil, err
			},
			"Applying AutoScaler instances",
		),
		StepWithProgressBar(
			func(ctx *appcontext.Context, r Result) (Result, error) {
				err := i.applyAutoscalerSchedules(ctx)
				return nil, err
			},
			"Applying AutoScaler schedules",
		),
	)
}

func (i *ImportApp) createApp(ctx *appcontext.Context) error {
	c := cache.GetCache(ctx.ImportCFClient)

	org, err := c.GetOrgByName(i.Org)
	if err != nil {
		return err
	}

	space, err := c.GetSpaceByName(i.Space, org.Guid)
	if err != nil {
		return err
	}

	cachedApp, err := c.GetAppByName(i.AppName, space.Guid)
	if err != nil && !cache.IsNotFound(err) {
		return err
	}

	manifestPath := filepath.Join(ctx.ExportDir, i.Org, i.Space, i.AppName+"_manifest.yml")
	manifest := export.AppManifest{}

	manifestFile, err := os.Open(manifestPath)
	if err != nil {
		return err
	}
	defer manifestFile.Close()

	ctx.Logger.Infof("Attempting to read %s to create app %s/%s/%s", manifestFile.Name(), i.Org, i.Space, i.AppName)

	if err = yaml.NewDecoder(manifestFile).Decode(&manifest); err != nil {
		return err
	}

	app := manifest.Applications[0]

	var cfApp cfclient.App
	var retryFunc func() error

	if cachedApp.Guid == "" {
		retryFunc = func() error {
			appRequest := cfclient.AppCreateRequest{}
			appRequest.Name = app.Name
			appRequest.Environment = app.Env
			appRequest.HealthCheckType = cfclient.HealthCheckType(app.HealthCheckType)
			appRequest.HealthCheckHttpEndpoint = app.HealthCheckHTTPEndpoint
			appRequest.HealthCheckTimeout = int(app.Timeout)
			appRequest.State = cfclient.APP_STOPPED
			appRequest.Memory = getSizeFromString(app.Memory)
			appRequest.DiskQuota = getSizeFromString(app.DiskQuota)
			appRequest.Command = app.Command
			appRequest.Instances = int(app.Instances)
			appRequest.SpaceGuid = space.Guid
			if app.Docker.Image != "" {
				appRequest.DockerImage = app.Docker.Image
				appRequest.DockerCredentials = cfclient.DockerCredentials{
					Username: app.Docker.Username,
					Password: "",
				}
			}

			if len(app.Buildpacks) == 1 {
				appRequest.Buildpack = app.Buildpacks[0]
			}

			if app.Stack != "" {
				guid, err := c.GetStackGUIDByName(app.Stack)
				if err != nil {
					return err
				}

				appRequest.StackGuid = guid
			}

			cfApp, err = ctx.ImportCFClient.CreateApp(appRequest)
			if err != nil {
				cfErr := cfclient.CloudFoundryHTTPError{}
				if errors.As(err, &cfErr) {
					if cfErr.StatusCode >= 500 && cfErr.StatusCode <= 599 {
						return cf.ErrRetry
					}
				}
			}

			return err
		}
	} else {
		retryFunc = func() error {
			appRequest := cfclient.AppUpdateResource{}
			appRequest.Name = app.Name
			appRequest.Environment = app.Env
			appRequest.HealthCheckType = app.HealthCheckType
			appRequest.HealthCheckHttpEndpoint = app.HealthCheckHTTPEndpoint
			appRequest.HealthCheckTimeout = int(app.Timeout)
			appRequest.Memory = getSizeFromString(app.Memory)
			appRequest.DiskQuota = getSizeFromString(app.DiskQuota)
			appRequest.Command = app.Command
			appRequest.Instances = int(app.Instances)
			appRequest.SpaceGuid = space.Guid
			if app.Docker.Image != "" {
				appRequest.DockerImage = app.Docker.Image
				appRequest.DockerCredentials = map[string]interface{}{
					"username": app.Docker.Username,
					"password": "",
				}
			}

			if len(app.Buildpacks) == 1 {
				appRequest.Buildpack = app.Buildpacks[0]
			}

			if app.Stack != "" {
				guid, err := c.GetStackGUIDByName(app.Stack)
				if err != nil {
					return err
				}

				appRequest.StackGuid = guid
			}
			appRes, err := ctx.ImportCFClient.UpdateApp(cachedApp.Guid, appRequest)
			if err != nil {
				cfErr := cfclient.CloudFoundryHTTPError{}
				if errors.As(err, &cfErr) {
					if cfErr.StatusCode >= 500 && cfErr.StatusCode <= 599 {
						return cf.ErrRetry
					}
				}

				return err
			}

			cfApp = cfclient.App{
				Guid:                    appRes.Metadata.Guid,
				CreatedAt:               appRes.Metadata.CreatedAt,
				UpdatedAt:               appRes.Metadata.UpdatedAt,
				Name:                    appRes.Entity.Name,
				Memory:                  appRes.Entity.Memory,
				Instances:               appRes.Entity.Instances,
				DiskQuota:               appRes.Entity.DiskQuota,
				SpaceGuid:               appRes.Entity.SpaceGuid,
				StackGuid:               appRes.Entity.StackGuid,
				State:                   appRes.Entity.State,
				PackageState:            appRes.Entity.PackageState,
				Command:                 appRes.Entity.Command,
				Buildpack:               appRes.Entity.Buildpack,
				DetectedBuildpack:       appRes.Entity.DetectedBuildpack,
				DetectedBuildpackGuid:   appRes.Entity.DetectedBuildpackGuid,
				HealthCheckHttpEndpoint: appRes.Entity.HealthCheckHttpEndpoint,
				HealthCheckType:         appRes.Entity.HealthCheckType,
				HealthCheckTimeout:      appRes.Entity.HealthCheckTimeout,
				Diego:                   appRes.Entity.Diego,
				EnableSSH:               appRes.Entity.EnableSSH,
				DetectedStartCommand:    appRes.Entity.DetectedStartCommand,
				DockerImage:             appRes.Entity.DockerImage,
				DockerCredentialsJSON: map[string]interface{}{
					"username": appRes.Entity.DockerCredentials.Username,
					"password": appRes.Entity.DockerCredentials.Password,
				},
				DockerCredentials:        cfclient.DockerCredentials(appRes.Entity.DockerCredentials),
				Environment:              appRes.Entity.Environment,
				StagingFailedReason:      appRes.Entity.StagingFailedReason,
				StagingFailedDescription: appRes.Entity.StagingFailedDescription,
				Ports:                    appRes.Entity.Ports,
				SpaceURL:                 appRes.Entity.SpaceURL,
				PackageUpdatedAt:         appRes.Entity.PackageUpdatedAt,
			}

			return nil
		}
	}

	err = ctx.ImportCFClient.DoWithRetry(retryFunc)
	if err != nil {
		return err
	}

	appResponse := cfclient.AppResource{
		Meta: cfclient.Meta{
			Guid:      cfApp.Guid,
			CreatedAt: cfApp.CreatedAt,
			UpdatedAt: cfApp.UpdatedAt,
		},
		Entity: cfApp,
	}
	newApp := c.AddApp(appResponse)

	i.appGUID = newApp.Guid

	if app.Docker.Image == "" && len(app.Buildpacks) > 1 {
		stackName := "cflinuxfs3"
		if newApp.StackGuid != "" {
			sn, err := c.GetStackNameByGUID(newApp.StackGuid)
			if err != nil {
				return err
			}
			stackName = sn
		}

		updateRequest := cfclient.UpdateV3AppRequest{}
		updateRequest.Name = i.AppName
		updateRequest.Lifecycle = &cfclient.V3Lifecycle{
			BuildpackData: cfclient.V3BuildpackLifecycle{
				Buildpacks: app.Buildpacks,
				Stack:      stackName,
			},
			Type: "buildpack",
		}
		updateRequest.Metadata = &cfclient.V3Metadata{}

		ctx.Logger.Infof("Attempting to update app %s/%s/%s by using V3 API", i.Org, i.Space, i.AppName)
		err = ctx.ImportCFClient.DoWithRetry(func() error {
			_, err = ctx.ImportCFClient.UpdateV3App(i.appGUID, updateRequest)
			if err != nil {
				cfErr := cfclient.CloudFoundryHTTPError{}
				if errors.As(err, &cfErr) {
					if cfErr.StatusCode >= 500 && cfErr.StatusCode <= 599 {
						return cf.ErrRetry
					}
				}
			}

			return err
		})
		if err != nil {
			return err
		}
	}

	routes := make([]string, 0, len(app.Routes))
	for _, r := range app.Routes {
		routes = append(routes, r.Route)
	}

	if !app.NoRoute || len(app.Routes) > 0 {
		if err = i.bindRoutes(ctx, routes); err != nil {
			return err
		}
	}

	err = i.bindServices(ctx, app.Services)

	if err != nil {
		return err
	}

	ctx.Summary.AddSuccessfulApp(org.Name, space.Name, app.Name)

	return nil
}

func (i *ImportApp) bindRoutes(ctx *appcontext.Context, routes []string) error {
	globalCache := cache.GetCache(ctx.ImportCFClient)

	for _, route := range routes {
		ctx.Logger.Infof("Binding route %s to app %s in org/space %s/%s\n", route, i.AppName, i.Org, i.Space)
		hostParts := strings.SplitN(route, ".", 2)
		host := hostParts[0]

		domainPath := strings.SplitN(hostParts[1], "/", 2)
		domain := domainPath[0]
		path := ""
		if len(domainPath) > 1 {
			path = domainPath[1]
		}

		if len(path) > 1 && !strings.HasPrefix(path, "/") {
			path = "/" + path
		}

		org, err := globalCache.GetOrgByName(i.Org)
		if err != nil {
			return err
		}

		space, err := globalCache.GetSpaceByName(i.Space, org.Guid)
		if err != nil {
			return err
		}

		domainGUID, err := globalCache.GetDomainGUIDByName(domain)
		if err != nil {
			return err
		}

		var routes []cfclient.Route
		err = ctx.ImportCFClient.DoWithRetry(func() error {
			routes, err = ctx.ImportCFClient.ListRoutesByQuery(url.Values{"q": []string{fmt.Sprintf("host:%s", host), fmt.Sprintf("domain_guid:%s", domainGUID), fmt.Sprintf("path:%s", path)}})
			cfErr := cfclient.CloudFoundryHTTPError{}
			if ok := errors.As(err, &cfErr); ok {
				if cfErr.StatusCode >= 500 && cfErr.StatusCode <= 599 {
					return fmt.Errorf("received HTTP error: %d", cfErr.StatusCode)
				}
			}
			return err
		})
		if err != nil {
			return err
		}

		routeGUID := ""
		switch {
		case len(routes) == 0:
			req := cfclient.RouteRequest{
				DomainGuid: domainGUID,
				SpaceGuid:  space.Guid,
				Host:       host,
				Path:       path,
			}

			var cfRoute cfclient.Route
			err = ctx.ImportCFClient.DoWithRetry(func() error {
				cfRoute, err = ctx.ImportCFClient.CreateRoute(req)
				cfErr := cfclient.CloudFoundryHTTPError{}
				if ok := errors.As(err, &cfErr); ok {
					if cfErr.StatusCode >= 500 && cfErr.StatusCode <= 599 {
						return fmt.Errorf("received HTTP error: %d", cfErr.StatusCode)
					}
				}
				return err
			})

			if err != nil {
				return err
			}

			routeGUID = cfRoute.Guid
		case len(routes) == 1:
			routeGUID = routes[0].Guid
			if routes[0].SpaceGuid != space.Guid {
				return fmt.Errorf("route %s is defined in a different space and cannot be bound", route)
			}
		default:
			return fmt.Errorf("should have found at most 1 route, but found %d", len(routes))
		}

		if err = ctx.ImportCFClient.DoWithRetry(func() error {
			err = ctx.ImportCFClient.BindRoute(routeGUID, i.appGUID)
			cfErr := cfclient.CloudFoundryHTTPError{}
			if ok := errors.As(err, &cfErr); ok {
				if cfErr.StatusCode >= 500 && cfErr.StatusCode <= 599 {
					return fmt.Errorf("received HTTP error: %d", cfErr.StatusCode)
				}
			}
			return err
		}); err != nil && !cfclient.IsRouteMappingTakenError(err) {
			return err
		}
	}

	return nil
}

func (i *ImportApp) bindServices(ctx *appcontext.Context, serviceNames []string) error {
	if len(serviceNames) == 0 {
		return nil
	}

	c := cache.GetCache(ctx.ImportCFClient)
	org, err := c.GetOrgByName(i.Org)
	if err != nil {
		return err
	}

	space, err := c.GetSpaceByName(i.Space, org.Guid)
	if err != nil {
		return err
	}

	siGUID := ""
	for _, serviceName := range serviceNames {
		ctx.Logger.Infof("Binding service %s to app %s in org/space %s/%s\n", serviceName, i.AppName, i.Org, i.Space)
		// find the SI that we're going to bind to this app
		params := url.Values{"q": []string{"name:" + serviceName, "space_guid:" + space.Guid}}

		var sis []cfclient.ServiceInstance
		err := ctx.ImportCFClient.DoWithRetry(func() error {
			var err error
			sis, err = ctx.ImportCFClient.ListServiceInstancesByQuery(params)
			cfErr := cfclient.CloudFoundryHTTPError{}
			if ok := errors.As(err, &cfErr); ok {
				if cfErr.StatusCode >= 500 && cfErr.StatusCode <= 599 {
					return fmt.Errorf("received HTTP error: %d", cfErr.StatusCode)
				}
			}
			return err
		})

		if err != nil {
			return err
		}
		if len(sis) != 1 {
			var upsis []cfclient.UserProvidedServiceInstance
			err = ctx.ImportCFClient.DoWithRetry(func() error {
				upsis, err = ctx.ImportCFClient.ListUserProvidedServiceInstancesByQuery(params)
				cfErr := cfclient.CloudFoundryHTTPError{}
				if ok := errors.As(err, &cfErr); ok {
					if cfErr.StatusCode >= 500 && cfErr.StatusCode <= 599 {
						return fmt.Errorf("received HTTP error: %d", cfErr.StatusCode)
					}
				}
				return err
			})

			if err != nil {
				return err
			}

			if len(upsis) != 1 {
				return &cache.ServiceInstanceNotFoundError{
					ServiceInstanceName: serviceName,
					Count:               len(upsis),
				}
			}

			siGUID = upsis[0].Guid
		} else {
			siGUID = sis[0].Guid
		}
		// bind the SI to the app
		err = ctx.ImportCFClient.DoWithRetry(func() error {
			_, err = ctx.ImportCFClient.CreateServiceBinding(i.appGUID, siGUID)
			if err != nil && !cfclient.IsServiceBindingAppServiceTakenError(err) {
				cfErr := cfclient.CloudFoundryHTTPError{}
				if ok := errors.As(err, &cfErr); ok {
					if cfErr.StatusCode >= 500 && cfErr.StatusCode <= 599 {
						return fmt.Errorf("received HTTP error: %d", cfErr.StatusCode)
					}
				}
				return err
			}
			return nil
		})
	}

	return nil
}

func (i *ImportApp) uploadBlob(ctx *appcontext.Context) error {
	dropletPath := filepath.Join(ctx.ExportDir, i.Org, i.Space, i.AppName+".tgz")
	appBitsPath := filepath.Join(ctx.ExportDir, i.Org, i.Space, i.AppName+".zip")

	dropletInfo, dErr := os.Stat(dropletPath)
	appBitsInfo, bErr := os.Stat(appBitsPath)

	if dErr != nil && bErr != nil {
		app, ok := cache.GetCache(ctx.ImportCFClient).GetAppByGUID(i.appGUID)
		if ok {
			if app.DockerImage != "" {
				return nil
			}
		}

		return fmt.Errorf("could not find %s or %s", dropletPath, appBitsPath)
	}

	switch {
	case dropletInfo == nil:
		return fmt.Errorf("there is no droplet for app %s", i.AppName)
	case appBitsInfo == nil:
		return fmt.Errorf("there are no app bits for app %s", i.AppName)
	}

	if err := i.uploadAppBits(ctx); err != nil {
		ctx.Logger.Error(err)
	}

	if err := i.uploadDroplet(ctx); err != nil {
		ctx.Logger.Error(err)
	}

	return nil
}

func (i *ImportApp) uploadDroplet(c *appcontext.Context) error {
	dropletFilePath := filepath.Join(c.ExportDir, i.Org, i.Space, i.AppName+".tgz")
	c.Logger.Infof("Uploading droplet for app %s/%s/%s", i.Org, i.Space, i.AppName)
	dropletReader, err := os.Open(dropletFilePath)
	if err != nil {
		return err
	}
	defer dropletReader.Close()

	err = c.ImportCFClient.DoWithRetry(func() error {
		_, err = c.ImportCFClient.UploadDropletBits(dropletReader, i.appGUID)
		cfErr := cfclient.CloudFoundryHTTPError{}
		if ok := errors.As(err, &cfErr); ok {
			if cfErr.StatusCode >= 500 && cfErr.StatusCode <= 599 {
				return fmt.Errorf("received HTTP error: %d", cfErr.StatusCode)
			}
		}
		return err
	})
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()

	// This is already in a timeout loop, we're not going to wrap it in another
	req := c.ImportCFClient.NewRequest(http.MethodGet, fmt.Sprintf("/v3/apps/%s/droplets?order_by=-updated_at", i.appGUID))
	var droplet cfclient.V3Droplet
	for {
		var droplets struct {
			Resources []cfclient.V3Droplet `json:"resources"`
		}
		dnsErr := new(net.DNSError)

		resp, err := c.ImportCFClient.DoRequest(req)
		if err != nil && !errors.As(err, &dnsErr) {
			return err
		}
		if resp == nil {
			time.Sleep(5 * time.Second)
			continue
		}

		defer resp.Body.Close()

		if err = json.NewDecoder(resp.Body).Decode(&droplets); err != nil {
			return err
		}

		if len(droplets.Resources) < 1 {
			return fmt.Errorf("no droplets found for app %s", i.AppName)
		}

		droplet = droplets.Resources[0]

		if droplet.State == "STAGED" {
			// workaround for the fact that the APIClient API doesn't mark old droplets as expired
			if len(droplets.Resources) > c.DropletCountToKeep {
				for _, oldDroplet := range droplets.Resources[c.DropletCountToKeep:] {
					req := c.ImportCFClient.NewRequest(http.MethodDelete, fmt.Sprintf("/v3/droplets/%s", oldDroplet.GUID))
					resp, err := c.ImportCFClient.DoRequest(req)
					if err != nil && !errors.As(err, &dnsErr) {
						return err
					}
					if resp == nil {
						time.Sleep(5 * time.Second)
						continue
					}

					defer resp.Body.Close()
				}
			}
			break
		}

		if droplet.State == "FAILED" || droplet.State == "EXPIRED" {
			return fmt.Errorf("bad droplet state %s", droplet.State)
		}

		ticker := time.NewTicker(5 * time.Second)
		select {
		case <-ticker.C:
			continue
		case <-ctx.Done():
			return errors.New("timed out waiting for droplet to stage")
		}
	}
	c.Logger.Infof("Close uploading droplet for app %s/%s/%s", i.Org, i.Space, i.AppName)

	return nil
}

func (i *ImportApp) uploadAppBits(ctx *appcontext.Context) error {
	zipFilePath := filepath.Join(ctx.ExportDir, i.Org, i.Space, i.AppName+".zip")
	ctx.Logger.Infof("Uploading bits for app %s/%s/%s", i.Org, i.Space, i.AppName)
	zipFile, err := os.Open(zipFilePath)
	if err != nil {
		return err
	}
	defer zipFile.Close()

	return ctx.ImportCFClient.DoWithRetry(func() error {
		err := ctx.ImportCFClient.UploadAppBits(zipFile, i.appGUID)
		cfErr := cfclient.CloudFoundryHTTPError{}
		if ok := errors.As(err, &cfErr); ok {
			if cfErr.StatusCode >= 500 && cfErr.StatusCode <= 599 {
				return fmt.Errorf("received HTTP error: %d", cfErr.StatusCode)
			}
		}
		return err
	})
}

func (i *ImportApp) applyAutoscalerRules(ctx *appcontext.Context) error {
	rulesFile := filepath.Join(ctx.ExportDir, i.Org, i.Space, i.AppName+"_autoscale_rules.json")
	contents, err := os.Open(rulesFile)
	if err != nil {
		if os.IsNotExist(err) {
			defer contents.Close()
			ctx.Logger.Infof("No autoscaler rules exist for app %s/%s/%s", i.Org, i.Space, i.AppName)
			return nil
		}

		return err
	}
	defer contents.Close()

	ctx.Logger.Infof("Uploading autoscaler rules for app %s/%s/%s", i.Org, i.Space, i.AppName)

	autoscalerBase := strings.Replace(ctx.ImportCFClient.Target(), "/api.", "/autoscale.", 1)

	req, err := http.NewRequest(http.MethodPut, fmt.Sprintf("%s/api/v2/apps/%s/rules", autoscalerBase, i.appGUID), contents)
	if err != nil {
		defer req.Body.Close()
		return err
	}
	defer req.Body.Close()

	var resp *http.Response
	err = ctx.ImportCFClient.DoWithRetry(func() error {
		resp, err = ctx.ImportCFClient.Do(req)
		if err == nil {
			if resp.StatusCode >= 500 && resp.StatusCode <= 599 {
				defer resp.Body.Close()
				return fmt.Errorf("received HTTP error: %d", resp.StatusCode)
			}
		}
		defer resp.Body.Close()
		return err
	})
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if (resp.StatusCode < 200 || resp.StatusCode > 299) && resp.StatusCode != http.StatusNotFound {
		return fmt.Errorf("expected an HTTP 2xx code, got %d instead", resp.StatusCode)
	}

	return nil

}

func (i *ImportApp) applyAutoscalerInstances(ctx *appcontext.Context) error {
	instanceLimitsFile := filepath.Join(ctx.ExportDir, i.Org, i.Space, i.AppName+"_autoscale_instances.json")

	contents, err := os.Open(instanceLimitsFile)
	if err != nil {
		if os.IsNotExist(err) {
			defer contents.Close()
			ctx.Logger.Infof("No autoscaler instance limits exist for app %s/%s/%s", i.Org, i.Space, i.AppName)
			return nil
		}

		return err
	}
	defer contents.Close()

	ctx.Logger.Infof("Uploading autoscaler instance limits for app %s/%s/%s", i.Org, i.Space, i.AppName)

	autoscalerBase := strings.Replace(ctx.ImportCFClient.Target(), "/api.", "/autoscale.", 1)

	req, err := http.NewRequest(http.MethodPut, fmt.Sprintf("%s/api/v2/apps/%s", autoscalerBase, i.appGUID), contents)
	if err != nil {
		defer req.Body.Close()
		return err
	}
	defer req.Body.Close()

	var resp *http.Response
	err = ctx.ImportCFClient.DoWithRetry(func() error {
		resp, err = ctx.ImportCFClient.Do(req)
		if err == nil {
			if resp.StatusCode >= 500 && resp.StatusCode <= 599 {
				defer resp.Body.Close()
				return fmt.Errorf("received HTTP error: %d", resp.StatusCode)
			}
		}
		defer resp.Body.Close()
		return err
	})

	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if (resp.StatusCode < 200 || resp.StatusCode > 299) && resp.StatusCode != http.StatusNotFound {
		return fmt.Errorf("expected an HTTP 2xx code, got %d instead", resp.StatusCode)
	}

	return nil
}

func (i *ImportApp) applyAutoscalerSchedules(ctx *appcontext.Context) error {
	scheduleFile := filepath.Join(ctx.ExportDir, i.Org, i.Space, i.AppName+"_autoscale_schedules.json")

	contents, err := os.Open(scheduleFile)
	if err != nil {
		if os.IsNotExist(err) {
			defer contents.Close()
			ctx.Logger.Infof("No autoscaler schedules exist for app %s/%s/%s", i.Org, i.Space, i.AppName)
			return nil
		}

		return err
	}
	defer contents.Close()

	ctx.Logger.Infof("Uploading autoscaler schedules for app %s/%s/%s", i.Org, i.Space, i.AppName)

	autoscalerBase := strings.Replace(ctx.ImportCFClient.Target(), "/api.", "/autoscale.", 1)

	req, err := http.NewRequest(http.MethodPut, fmt.Sprintf("%s/api/v2/apps/%s/scheduled_limit_changes", autoscalerBase, i.appGUID), contents)
	if err != nil {
		defer req.Body.Close()
		return err
	}
	defer req.Body.Close()

	var resp *http.Response
	err = ctx.ImportCFClient.DoWithRetry(func() error {
		resp, err = ctx.ImportCFClient.Do(req)
		if err == nil {
			if resp.StatusCode >= 500 && resp.StatusCode <= 599 {
				defer resp.Body.Close()
				return fmt.Errorf("received HTTP error: %d", resp.StatusCode)
			}
		}
		defer resp.Body.Close()
		return err
	})
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if (resp.StatusCode < 200 || resp.StatusCode > 299) && resp.StatusCode != http.StatusNotFound {
		return fmt.Errorf("expected an HTTP 2xx code, got %d instead", resp.StatusCode)
	}

	return nil
}

func (i *ImportApp) getAppNameFromManifest(ctx *appcontext.Context) (string, error) {
	fileName := filepath.Join(ctx.ExportDir, i.Org, i.Space, i.AppName+"_manifest.yml")

	file, err := os.Open(fileName)
	if err != nil {
		return "", err
	}
	defer file.Close()

	manifest := export.AppManifest{}
	if err = yaml.NewDecoder(file).Decode(&manifest); err != nil {
		return "", err
	}

	if len(manifest.Applications) != 1 {
		return "", fmt.Errorf("expected to find one application in manifest, but found %d", len(manifest.Applications))
	}

	// yes i'm aware this will change the app name if they have a '/' in it
	// but honestly, c'mon, slashes in your app name?
	sanitizedAppName := getAppFileName(manifest.Applications[0].Name)

	return sanitizedAppName, nil
}

func getSizeFromString(sizeStr string) int {
	lastChar := sizeStr[len(sizeStr)-1:]
	size := sizeStr[:len(sizeStr)-1]

	sizeInt, err := strconv.Atoi(size)
	if err != nil {
		return -1
	}

	if strings.EqualFold(lastChar, "g") {
		return sizeInt * 1024
	}

	return sizeInt
}

func getAppFileName(appName string) string {
	return strings.ReplaceAll(appName, "/", "_")
}
