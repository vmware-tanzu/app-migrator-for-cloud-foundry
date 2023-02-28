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
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/commands"
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/context"
)

func CreateImportOrgCommand(ctx *context.Context, i commands.ImportOrg) *cobra.Command {
	var importOrg = &cobra.Command{
		Use:     "org",
		Aliases: []string{"o"},
		Short:   "Import org",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return errors.New("requires the name of the org to import")
			}
			if args[0] == "" {
				return errors.New("requires a valid name of the org to import")
			}
			if len(args) > 1 {
				return errors.New("too many arguments passed in. only the name of the org is required")
			}
			return nil
		},
		Example: "service-instance-migrator import org sample-org",
		RunE:    importOrg(ctx, i),
	}
	return importOrg
}

func importOrg(ctx *context.Context, i commands.ImportOrg) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		i.Org = args[0]
		if err := i.Run(ctx); err != nil {
			return err
		}
		return nil
	}
}
