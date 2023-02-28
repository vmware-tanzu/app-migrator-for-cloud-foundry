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
	"strings"

	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/context"

	"github.com/cloudfoundry-community/go-cfclient"
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/cache"
)

type ExportSpace struct {
	ExportOrg
	Space string `help:"the space to export" short:"s" env:"CF_SPACE"`
}

func (e *ExportSpace) Run(ctx *context.Context, orgName, spaceName string) error {
	c := cache.GetCache(ctx.ExportCFClient)

	var (
		org   cfclient.Org
		space cfclient.Space
		err   error
	)

	org, err = c.GetOrgByName(orgName)
	if err != nil {
		return err
	}

	space, err = c.GetSpaceByName(spaceName, org.Guid)
	if err != nil {
		return err
	}

	exportApp := func(ctx *context.Context, r context.QueryResult) context.ProcessResult {
		appExporter := &ExportApp{
			ExportSpace: *e,
			AppName:     fmt.Sprintf("%v", r.Value),
		}
		err := appExporter.Run(ctx, orgName, spaceName)
		return context.ProcessResult{Value: r.Value, Err: err}
	}

	var results <-chan context.ProcessResult
	results, err = ctx.SpaceExporter.ExportSpace(ctx, space, exportApp)
	if err != nil {
		return err
	}

	for r := range results {
		appPath := strings.Join([]string{orgName, spaceName, fmt.Sprintf("%v", r.Value)}, "/")
		if r.Err != nil {
			ctx.Summary.AddFailedApp(orgName, spaceName, appPath, r.Err)
		}
	}

	return nil
}
