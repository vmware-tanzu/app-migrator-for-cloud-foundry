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
	"fmt"
	"path/filepath"

	"github.com/cloudfoundry-community/go-cfclient"
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/cache"
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/context"
)

type ExportApp struct {
	ExportSpace
	Sequence Sequence
	AppName  string `help:"the app to export" short:"a" env:"CF_APP_NAME"`
}

func (e *ExportApp) SetAppName(name string) {
	e.AppName = name
}

func (e *ExportApp) Run(ctx *context.Context, orgName, spaceName string) error {
	exportDir := filepath.Join(ctx.ExportDir, orgName, spaceName)

	if err := ctx.DirWriter.Mkdir(exportDir); err != nil {
		return fmt.Errorf("cannot create target directory: %w", err)
	}

	if e.Sequence == nil {
		e.Sequence = NewExportAppSequence(orgName, spaceName, e.AppName, exportDir)
	}

	if _, err := e.Sequence.Run(ctx, nil); err != nil {
		ctx.Logger.Errorf("Error occurred exporting app %s/%s/%s: %s", orgName, spaceName, e.AppName, err)
	}

	return nil
}

func NewExportAppSequence(orgName, spaceName, appName string, exportDir string) Sequence {
	return RunSequence(
		fmt.Sprintf("\x1b[31m%v\x1b[0m", appName),
		appName,
		StepWithProgressBar(
			func(ctx *context.Context, r Result) (Result, error) {
				c := cache.GetCache(ctx.ExportCFClient)

				var org cfclient.Org
				org, err := c.GetOrgByName(orgName)
				if err != nil {
					ctx.Logger.Errorf("Error getting org by name %s/%s/%s: %v", orgName, spaceName, appName, err)
					return nil, err
				}

				var space cfclient.Space
				space, err = c.GetSpaceByName(spaceName, org.Guid)
				if err != nil {
					ctx.Logger.Errorf("Error getting space by name %s/%s/%s: %v", orgName, spaceName, appName, err)
					return nil, err
				}

				var app cfclient.App
				app, err = c.GetAppByName(appName, space.Guid)
				if err != nil {
					ctx.Logger.Errorf("Error getting app by name %s/%s/%s: %v", orgName, spaceName, appName, err)
					return nil, err
				}
				return ExportAppResult{
					org:   org,
					space: space,
					app:   app,
				}, nil
			},
			"Loading Cache",
		),
		StepWithProgressBar(
			func(ctx *context.Context, r Result) (Result, error) {
				if err := exportPackages(ctx, r.GetOrg(), r.GetSpace(), r.GetApp(), exportDir); err != nil {
					return nil, err
				}
				return r, nil
			},
			"Exporting Packages",
		),
		StepWithProgressBar(
			func(ctx *context.Context, r Result) (Result, error) {
				if err := ctx.AutoScalerExporter.ExportAutoScalerRules(ctx, r.GetOrg(), r.GetSpace(), r.GetApp(), exportDir); err != nil {
					return nil, err
				}
				return r, nil
			},
			"Exporting AutoScaler Rules",
		),
		StepWithProgressBar(
			func(ctx *context.Context, r Result) (Result, error) {
				if err := ctx.AutoScalerExporter.ExportAutoScalerInstances(ctx, r.GetOrg(), r.GetSpace(), r.GetApp(), exportDir); err != nil {
					return nil, err
				}
				return r, nil
			},
			"Exporting AutoScaler Instances",
		),
		StepWithProgressBar(
			func(ctx *context.Context, r Result) (Result, error) {
				if err := ctx.AutoScalerExporter.ExportAutoScalerSchedules(ctx, r.GetOrg(), r.GetSpace(), r.GetApp(), exportDir); err != nil {
					return nil, err
				}
				return r, nil
			},
			"Exporting AutoScaler Schedules",
		),
		StepWithProgressBar(
			func(ctx *context.Context, r Result) (Result, error) {
				if err := ctx.ManifestExporter.ExportAppManifest(ctx, r.GetOrg(), r.GetSpace(), r.GetApp(), exportDir); err != nil {
					return nil, err
				}
				return r, nil
			},
			"Exporting Manifest",
		),
		StepWithProgressBar(
			func(ctx *context.Context, r Result) (Result, error) {
				if err := ctx.Metadata.RecordUpdate(r.GetApp(), r.GetSpace(), r.GetOrg()); err != nil {
					return nil, err
				}
				ctx.Summary.AddSuccessfulApp(orgName, spaceName, appName)
				return r, nil
			},
			"Recording Results",
		),
	)
}

func exportPackages(ctx *context.Context, org cfclient.Org, space cfclient.Space, app cfclient.App, exportDir string) error {
	// Don't export packages if app is a Docker image
	if app.DockerImage != "" {
		return nil
	}

	numOfPackages, err := ctx.DropletExporter.NumberOfPackages(ctx, app)
	if err != nil {
		ctx.Logger.Errorf("Error getting packages: %s", err)
		return err
	}

	if numOfPackages == 0 {
		ctx.Logger.Warnf("App %s/%s/%s has no packages with bits to download, so it will be skipped", org.Name, space.Name, app.Name)
		return nil
	}

	err = ctx.DropletExporter.DownloadDroplet(ctx, org, space, app, exportDir)
	if err != nil {
		ctx.Logger.Error(err)
		return err
	}

	if err = ctx.DropletExporter.DownloadPackages(ctx, org, space, app, exportDir); err != nil {
		ctx.Logger.Error(err)
		return err
	}

	return nil
}
