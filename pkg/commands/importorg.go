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
	"io/fs"
	"os"
	"path/filepath"

	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/context"

	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/cache"
)

type ImportOrg struct {
	Org string `help:"the org to import" short:"o" env:"CF_ORG"`
}

func (i *ImportOrg) Run(ctx *context.Context) error {
	rootDir := filepath.Join(ctx.ExportDir, i.Org)

	item, err := os.Stat(rootDir)
	if err != nil {
		return err
	}

	if !item.IsDir() {
		return fmt.Errorf("%s is not a directory", rootDir)
	}

	globalCache := cache.GetCache(ctx.ImportCFClient)
	org, err := globalCache.GetOrgByName(i.Org)
	if err != nil {
		return err
	}

	err = filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, e error) error {
		if rootDir == path {
			return nil
		}

		if d.IsDir() {
			_, _ = fmt.Fprintf(os.Stderr, "Found space %s in org %s\n", d.Name(), i.Org)
			_, err = globalCache.GetSpaceByName(d.Name(), org.Guid)
			if err != nil {
				return err
			}

			importSpace := &ImportSpace{
				ImportOrg: *i,
				Space:     d.Name(),
			}

			if err = importSpace.Run(ctx); err != nil {
				return err
			}

			return filepath.SkipDir
		}

		return filepath.SkipDir
	})

	if err != nil {
		return err
	}

	return nil
}
