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

package main

import (
	"fmt"
	"os"
	"path"

	"github.com/spf13/cobra"
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/export"

	log "github.com/sirupsen/logrus"
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/cmd"
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/context"
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/io"
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/metadata"
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/report"
)

func main() {
	ctx := newContext()

	root := cmd.CreateRootCommand(ctx)
	// To prevent flag errors from being silenced due to SilenceErrors being set to true
	root.SetFlagErrorFunc(func(c *cobra.Command, err error) error {
		c.Printf("Error: %v\n", err)
		c.Println(c.UsageString())
		return cmd.ErrSilent
	})

	if err := root.Execute(); err != nil {
		if err != cmd.ErrSilent {
			_, _ = fmt.Fprintln(os.Stderr, err)
		}
		os.Exit(1)
	}
}

func newContext() *context.Context {
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	ctx := &context.Context{
		ExportDir:          path.Join(cwd, "export"),
		ExcludedOrgs:       []string{"^system$"},
		DropletCountToKeep: 2,
		Metadata:           metadata.NewMetadata(),
		Summary:            report.NewSummary(os.Stdout),
		DirWriter:          io.NewDirWriter(),
		DropletExporter:    export.NewDropletExporter(),
		ManifestExporter:   export.NewManifestExporter(),
		AutoScalerExporter: export.NewAutoScalerExporter(),
	}
	ctx.InitLogger()

	return ctx
}
