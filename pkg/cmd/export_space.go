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

package cmd

import (
	"errors"

	"github.com/spf13/cobra"
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/context"
)

func CreateExportSpaceCommand(ctx *context.Context, r context.SpaceCommandRunner) *cobra.Command {
	var exportSpace = &cobra.Command{
		Use:     "space",
		Aliases: []string{"s"},
		Short:   "Export space",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return errors.New("requires the name of the space to export")
			}
			return nil
		},
		Example: `app-migrator export space sample-space -o sample-org
app-migrator export space sample-space --org sample-org --export-dir=./export --domains-to-replace="foo.com=bar.com"
`,
		RunE: exportSpace(ctx, r),
	}
	return exportSpace
}

func exportSpace(ctx *context.Context, runner context.SpaceCommandRunner) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		org, err := cmd.Flags().GetString("org")
		if err != nil {
			return err
		}

		if err := runner.Run(ctx, org, args[0]); err != nil {
			return err
		}
		return nil
	}
}
