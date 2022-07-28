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
)

type ImportAll struct {
}

func (i *ImportAll) Run(ctx *context.Context) error {
	err := filepath.WalkDir(ctx.ExportDir, func(path string, d fs.DirEntry, err error) error {
		if path == ctx.ExportDir {
			return nil
		}

		if d.IsDir() {
			fmt.Fprintf(os.Stderr, "Importing from org %s\n", d.Name())
			if isOrgExcluded(ctx, d.Name()) || !isOrgIncluded(ctx, d.Name()) {
				return filepath.SkipDir
			}

			importOrg := &ImportOrg{
				Org: d.Name(),
			}

			err := importOrg.Run(ctx)
			if err != nil {
				return err
			}

			return filepath.SkipDir
		}

		return nil
	})

	return err
}
