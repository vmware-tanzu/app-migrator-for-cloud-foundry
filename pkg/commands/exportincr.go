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
	"bytes"
	"fmt"
	"net/url"
	"sync"

	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/context"

	"github.com/cloudfoundry-community/go-cfclient"
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/cache"
)

type ExportIncremental struct {
}

func (e *ExportIncremental) Run(ctx *context.Context) error {
	c := cache.GetCache(ctx.ExportCFClient)

	var wg sync.WaitGroup
	var errExpInc error

	q := url.Values{}
	q.Set("inline-relations-depth", "0")
	ctx.Logger.Warnln("Querying the CF API for all apps, this may take a bit.")
	apps, err := ctx.ExportCFClient.ListAppsByQuery(q)
	if err != nil {
		return err
	}
	appCount := len(apps)
	workerChan := make(chan string, appCount)
	wg.Add(appCount)

	for i := 0; i < appCount; i++ {
		go func() {
			defer wg.Done()
			for appGUID := range workerChan {
				var err error
				app, ok := c.GetAppByGUID(appGUID)

				buf := &bytes.Buffer{}

				if !ok {
					ctx.Logger.Warnf("could not find app with guid %s", appGUID)
					fmt.Print(buf.String())
					continue
				}

				var space cfclient.Space
				space, err = c.GetSpaceByGUID(app.SpaceGuid)
				if err != nil {
					errExpInc = err
					ctx.Logger.Error(errExpInc)
					fmt.Print(buf.String())
					continue
				}

				var org cfclient.Org
				org, err = c.GetOrgByGUID(space.OrganizationGuid)
				if err != nil {
					errExpInc = err
					ctx.Logger.Error(errExpInc)
					fmt.Print(buf.String())
					continue
				}

				if isOrgExcluded(ctx, org.Name) || !isOrgIncluded(ctx, org.Name) {
					continue
				}

				if !ctx.Metadata.HasBeenUpdated(app, space, org) {
					ctx.Logger.Infof("App %s/%s/%s has not been updated since the latest run of app-migrator, skip it", org.Name, space.Name, app.Name)
					fmt.Print(buf.String())
					continue
				}

				appExporter := &ExportApp{
					ExportSpace: ExportSpace{
						ExportOrg: ExportOrg{
							Org: org.Name,
						},
						Space: space.Name,
					},
					AppName: app.Name,
				}

				ctx.Logger.Infoln(buf.String())
				err = appExporter.Run(ctx, org.Name, space.Name)
				if err != nil {
					errExpInc = err
					ctx.Logger.Error(errExpInc)
					ctx.Logger.Infoln(buf.String())
				}
			}
		}()
	}

	go func() {
		defer close(workerChan)
		for _, app := range apps {
			workerChan <- app.Guid
		}
	}()
	wg.Wait()

	return errExpInc
}
