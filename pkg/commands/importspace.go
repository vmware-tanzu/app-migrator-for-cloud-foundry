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
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/context"
)

type ImportSpace struct {
	ImportOrg
	Space string `help:"the space to import" short:"s" env:"CF_SPACE"`
}

//Run ranges over apps in the export directory and imports all the apps it finds in a given space
func (i *ImportSpace) Run(ctx *context.Context) error {
	var err error
	var files []string
	glob := filepath.Join(ctx.ExportDir, i.Org, i.Space, "*_manifest.yml")
	files, err = filepath.Glob(glob)
	if err != nil {
		return err
	}
	numOfApps := len(files)
	if numOfApps == 0 {
		return errors.New("list of apps is empty")
	}

	importApp := func(ctx *context.Context, r context.QueryResult) context.ProcessResult {
		appImporter := &ImportApp{
			ImportSpace: ImportSpace{
				ImportOrg: i.ImportOrg,
				Space:     i.Space,
			},
			AppName: fmt.Sprintf("%v", r.Value),
		}
		err := appImporter.Run(ctx)
		return context.ProcessResult{Value: r.Value, Err: err}
	}

	var results <-chan context.ProcessResult
	results, err = ctx.SpaceImporter.ImportSpace(ctx, importApp, files)
	if err != nil {
		return err
	}

	for r := range results {
		if r.Err != nil {
			appPath := strings.Join([]string{i.Org, i.Space, fmt.Sprintf("%v", r.Value)}, "/")
			ctx.Summary.AddFailedApp(i.Org, i.Space, appPath, r.Err)
		}
	}

	return nil
}
