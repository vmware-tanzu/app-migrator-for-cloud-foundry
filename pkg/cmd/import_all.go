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
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/context"

	"github.com/spf13/cobra"
)

func CreateImportCommand(ctx *context.Context, i context.CommandRunner) *cobra.Command {
	export := &cobra.Command{
		Use:   "import",
		Short: "Import Cloud Foundry applications",
		Example: `app-migrator import
app-migrator import --exclude-orgs='^system$",p-*'
app-migrator import --exclude-orgs='system,p-spring-cloud-services' --export-dir=/tmp
app-migrator import --include-orgs='org1,org2' --export-dir=/tmp`,
		RunE: importAll(ctx, i),
	}
	return export
}

func importAll(ctx *context.Context, i context.CommandRunner) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		if err := i.Run(ctx); err != nil {
			return err
		}
		return nil
	}
}
