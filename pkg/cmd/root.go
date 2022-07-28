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
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/cf"
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/cli"
	. "github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/cli"
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/commands"
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/context"
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/export"
	im "github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/import"
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/process"
)

var ErrSilent = errors.New("SilentErr")

func CreateRootCommand(ctx *context.Context) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:           "app-migrator",
		Short:         "The app-migrator CLI is a tool for migrating apps from one TAS (Tanzu Application Service) to another",
		SilenceUsage:  true,
		SilenceErrors: true, // needed because we currently return "success" errors from commands
	}

	// set version
	if !Env.GitDirty {
		rootCmd.Version = fmt.Sprintf("%s (%s)", Env.Version, Env.GitSha)
	} else {
		rootCmd.Version = fmt.Sprintf("%s (%s, with local modifications)", Env.Version, Env.GitSha)
	}
	rootCmd.Flags().Bool("version", false, "display CLI version")
	rootCmd.AddCommand(createCompletionCommand())

	// load metadata before command runs
	rootCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		err := cli.PreRunLoadMetadata(ctx)
		if err != nil {
			log.Fatal(err)
		}
	}

	// show a migration summary for all commands
	rootCmd.PersistentPostRun = func(cmd *cobra.Command, args []string) {
		if cmd.Name() == "help" {
			return
		}
		err := cli.PostRunSaveMetadata(ctx)
		if err != nil {
			log.Fatalln(err)
		}
		cli.DisplaySummary(ctx)
	}

	addExportCommands(rootCmd, ctx)
	addImportCommands(rootCmd, ctx)

	rootCmd.PersistentFlags().BoolVar(&ctx.Debug, "debug", false, "Enable debug logging")
	rootCmd.PersistentFlags().StringVar(&ctx.ExportDir, "export-dir", ctx.ExportDir, "Directory where apps will be placed or read")

	return rootCmd
}

func addExportCommands(rootCmd *cobra.Command, ctx *context.Context) {
	newCFClient(ctx, true)
	exportCmd := CreateExportCommand(ctx, &commands.ExportAll{})
	exportCmd.PersistentFlags().IntVarP(&ctx.ConcurrencyLimit, "concurrency-limit", "l", ctx.ConcurrencyLimit, "Number of apps to export concurrently")
	exportCmd.PersistentFlags().StringArrayVar(&ctx.DomainsToAdd, "domains-to-add", []string{}, "Domains to add in any found application routes")
	exportCmd.PersistentFlags().StringToStringVar(&ctx.DomainsToReplace, "domains-to-replace", map[string]string{}, "Domains to replace in any found application routes")
	exportCmd.Flags().StringSliceVar(&ctx.IncludedOrgs, "include-orgs", ctx.IncludedOrgs, "Only orgs matching the regex(es) specified will be included")
	exportCmd.Flags().StringSliceVar(&ctx.ExcludedOrgs, "exclude-orgs", ctx.ExcludedOrgs, "Any orgs matching the regex(es) specified will be excluded")

	exportAppCmd := CreateExportAppCommand(ctx, &commands.ExportApp{})
	exportAppCmd.Flags().StringP("org", "o", "", "org to which the app belongs")
	exportAppCmd.Flags().StringP("space", "s", "", "space to which the app belongs")
	exportCmd.AddCommand(exportAppCmd)

	exportOrgCmd := CreateExportOrgCommand(ctx, &commands.ExportOrg{})
	exportCmd.AddCommand(exportOrgCmd)

	exportSpaceCmd := CreateExportSpaceCommand(ctx, &commands.ExportSpace{})
	exportSpaceCmd.Flags().StringP("org", "o", "", "org to which the space belongs")
	err := exportSpaceCmd.MarkFlagRequired("org")
	if err != nil {
		log.Fatalln(err.Error())
	}
	exportCmd.AddCommand(exportSpaceCmd)
	rootCmd.AddCommand(exportCmd)

	exportIncCmd := CreateExportIncrementalCommand(ctx, &commands.ExportIncremental{})
	rootCmd.AddCommand(exportIncCmd)
}

func addImportCommands(rootCmd *cobra.Command, ctx *context.Context) {
	newCFClient(ctx, false)
	importCmd := CreateImportCommand(ctx, &commands.ImportAll{})
	importCmd.Flags().StringSliceVar(&ctx.IncludedOrgs, "include-orgs", ctx.IncludedOrgs, "Only orgs matching the regex(es) specified will be included")
	importCmd.Flags().StringSliceVar(&ctx.ExcludedOrgs, "exclude-orgs", ctx.ExcludedOrgs, "Any orgs matching the regex(es) specified will be excluded")

	importAppCmd := CreateImportAppCommand(ctx, &commands.ImportApp{})
	importAppCmd.Flags().StringP("org", "o", "", "org to which the app belongs")
	importAppCmd.Flags().StringP("space", "s", "", "space to which the app belongs")
	importCmd.AddCommand(importAppCmd)

	importOrgCmd := CreateImportOrgCommand(ctx, commands.ImportOrg{})
	importCmd.AddCommand(importOrgCmd)

	importSpaceCmd := CreateImportSpaceCommand(ctx, commands.ImportSpace{})
	importSpaceCmd.Flags().StringP("org", "o", "", "org to which the space belongs")
	err := importSpaceCmd.MarkFlagRequired("org")
	if err != nil {
		log.Fatalln(err.Error())
	}
	importCmd.AddCommand(importSpaceCmd)
	rootCmd.AddCommand(importCmd)

	importIncCmd := CreateImportIncrementalCommand(ctx, &commands.ImportIncremental{})
	rootCmd.AddCommand(importIncCmd)
}

func newCFClient(ctx *context.Context, isExport bool) {
	cfg, err := cli.NewDefaultConfig()
	if err != nil {
		os.Exit(1)
	}

	controller := cfg.TargetApi
	if isExport {
		controller = cfg.SourceApi
	}

	client, err := cf.NewClient(&cf.Config{
		Target:       controller.URL,
		Username:     controller.Username,
		Password:     controller.Password,
		ClientID:     controller.ClientID,
		ClientSecret: controller.ClientSecret,
		SSLDisabled:  true,
	})
	if err != nil {
		log.Fatal(err)
	}

	ctx.ImportCFClient = client
	if isExport {
		ctx.ExportCFClient = client
		ctx.ImportCFClient = nil
	}
	ctx.Debug = cfg.Debug
	ctx.DomainsToAdd = cfg.DomainsToAdd
	ctx.DomainsToReplace = cfg.DomainsToReplace
	ctx.ConcurrencyLimit = cfg.ConcurrencyLimit
	ctx.ExportDir = cfg.ExportDir
	ctx.IncludedOrgs = cfg.IncludedOrgs
	ctx.ExcludedOrgs = cfg.ExcludedOrgs
	ctx.SpaceExporter = export.NewConcurrentSpaceExporter(
		process.NewQueryResultsProcessor(ctx.ExportCFClient),
	)
	ctx.SpaceImporter = im.NewConcurrentSpaceImporter(
		process.NewQueryResultsProcessor(ctx.ImportCFClient),
	)
}
