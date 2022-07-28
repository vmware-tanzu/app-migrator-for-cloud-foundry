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

func CreateImportAppCommand(ctx *context.Context, r context.AppImportRunner) *cobra.Command {
	var exportApp = &cobra.Command{
		Use:     "app",
		Aliases: []string{"a"},
		Short:   "Import app",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return errors.New("requires the name of the app to export")
			}
			return nil
		},
		Example: "app-migrator import app sample-app -o my-org -s my-space",
		RunE:    importApp(ctx, r),
	}
	return exportApp
}

func importApp(ctx *context.Context, r context.AppImportRunner) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		app := args[0]
		r.SetAppName(app)

		org, err := cmd.Flags().GetString("org")
		if err != nil {
			return err
		}
		r.SetOrgName(org)

		space, err := cmd.Flags().GetString("space")
		if err != nil {
			return err
		}
		r.SetSpaceName(space)

		if err := r.Run(ctx); err != nil {
			return err
		}
		return nil
	}
}
