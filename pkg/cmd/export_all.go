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
	"github.com/spf13/cobra"
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/context"
)

func CreateExportCommand(ctx *context.Context, e context.CommandRunner) *cobra.Command {
	export := &cobra.Command{
		Use:   "export",
		Short: "Export Cloud Foundry applications",
		Example: `app-migrator export
app-migrator export --exclude-orgs='^system$",p-*'
app-migrator export --include-orgs='system,test' --domains-to-replace 'tas1.example.com=tas2.example.com'
app-migrator export --exclude-orgs='system,p-spring-cloud-services' --domains-to-replace 'tas1.example.com=tas2.example.com'
app-migrator export --exclude-orgs='system,p-spring-cloud-services' --domains-to-replace 'tas1.example.com=tas2.example.com' --export-dir=/tmp`,
		RunE: exportAll(ctx, e),
	}
	return export
}

func exportAll(ctx *context.Context, e context.CommandRunner) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		if err := e.Run(ctx); err != nil {
			return err
		}
		return nil
	}
}
