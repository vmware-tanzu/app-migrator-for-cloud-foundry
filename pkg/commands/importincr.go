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
	"io/fs"
	"math"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/cache"
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/context"
)

type ImportIncremental struct {
}

func (i *ImportIncremental) Run(ctx *context.Context) error {
	var wg sync.WaitGroup
	workerChan := make(chan string, 1)
	numPerCPU := 5 * runtime.NumCPU()
	maxGoRoutines := int(math.Max(float64(numPerCPU), 1))

	wg.Add(maxGoRoutines)
	for j := 0; j < maxGoRoutines; j++ {
		go func() {
			defer wg.Done()
			for appIdentifier := range workerChan {
				buf := &bytes.Buffer{}
				orgSpaceApp := strings.SplitN(appIdentifier, "/", 3)

				appImporter := &ImportApp{
					ImportSpace: ImportSpace{
						ImportOrg: ImportOrg{
							Org: orgSpaceApp[0],
						},
						Space: orgSpaceApp[1],
					},
					AppName: strings.TrimSuffix(orgSpaceApp[2], "_manifest.yml"),
				}

				ctx.Logger.Infoln(buf.String())

				err := appImporter.Run(ctx)
				if err != nil {
					// TODO: Should we ignore errors here?
					continue
				}
			}
		}()
	}

	err := filepath.WalkDir(ctx.ExportDir, func(path string, d fs.DirEntry, err error) error {

		if !d.IsDir() && strings.HasSuffix(path, "_manifest.yml") {
			if err != nil {
				ctx.Logger.Error(err)
				return nil
			}

			newPath := strings.TrimPrefix(path, ctx.ExportDir+"/")
			orgSpaceApp := strings.SplitN(newPath, "/", 3)

			c := cache.GetCache(ctx.ImportCFClient)

			org, err := c.GetOrgByName(orgSpaceApp[0])
			if err != nil {
				return err
			}

			if isOrgExcluded(ctx, org.Name) || !isOrgIncluded(ctx, org.Name) {
				return nil
			}

			space, err := c.GetSpaceByName(orgSpaceApp[1], org.Guid)
			if err != nil {
				return err
			}

			app, err := c.GetAppByName(strings.TrimSuffix(orgSpaceApp[2], "_manifest.yml"), space.Guid)
			if err != nil && !cache.IsNotFound(err) {
				return err
			}

			if !ctx.Metadata.HasNewerLocally(app, space, org) {
				ctx.Logger.Infof("%s has not been modified since the last run of app-migrator, so skip that app", newPath)
				return nil
			}
			workerChan <- newPath
		}

		return nil
	})
	close(workerChan)
	wg.Wait()

	return err
}
