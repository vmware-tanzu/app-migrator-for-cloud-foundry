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

func CreateExportOrgCommand(ctx *context.Context, r context.OrgCommandRunner) *cobra.Command {
	var exportOrg = &cobra.Command{
		Use:     "org",
		Aliases: []string{"o"},
		Short:   "Export org",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return errors.New("requires the name of the org to export")
			}
			return nil
		},
		Example: "app-migrator export org sample-org",
		RunE:    exportOrg(ctx, r),
	}
	return exportOrg
}

func exportOrg(ctx *context.Context, runner context.OrgCommandRunner) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		org := args[0]
		if err := runner.Run(ctx, org); err != nil {
			return err
		}
		return nil
	}
}
