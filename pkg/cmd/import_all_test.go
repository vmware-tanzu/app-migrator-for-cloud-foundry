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

package cmd_test

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"
	"github.com/stretchr/testify/require"
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/cache"
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/cli"
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/cmd"
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/context"
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/context/fakes"
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/metadata"
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/report"
)

func TestImportCommand(t *testing.T) {
	fakeImportCommandRunner := new(fakes.FakeCommandRunner)
	type args struct {
		ctx         *context.Context
		i           context.CommandRunner
		commandArgs []string
	}
	tests := []struct {
		name       string
		args       args
		want       *cobra.Command
		wantErr    bool
		beforeFunc func()
		afterFunc  func(ctx *context.Context)
	}{
		{
			name: "import all orgs",
			args: args{
				ctx:         &context.Context{},
				i:           fakeImportCommandRunner,
				commandArgs: []string{"import", "--export-dir", "/path/to/import-dir", "--domains-to-replace", "foo.com=bar.com"},
			},
			beforeFunc: func() {
				fakeImportCommandRunner.RunCalls(func(c *context.Context) error {
					require.EqualValues(t, map[string]string{"foo.com": "bar.com"}, c.DomainsToReplace)
					return nil
				})
			},
			afterFunc: func(ctx *context.Context) {
				require.Equal(t, 1, fakeImportCommandRunner.RunCallCount())
				require.EqualValues(t, map[string]string{"foo.com": "bar.com"}, ctx.DomainsToReplace)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Cleanup(func() {
				cache.Cache = nil
			})
			ctx := tt.args.ctx
			importCommand := cmd.CreateImportCommand(tt.args.ctx, tt.args.i)
			importCommand.Flags().StringSliceVar(&ctx.IncludedOrgs, "include-orgs", ctx.IncludedOrgs, "Only orgs matching the regex(es) specified will be included")
			importCommand.Flags().StringSliceVar(&ctx.ExcludedOrgs, "exclude-orgs", ctx.ExcludedOrgs, "Any orgs matching the regex(es) specified will be excluded")
			importCommand.PersistentFlags().StringToStringVar(&ctx.DomainsToReplace, "domains-to-replace", map[string]string{}, "Domains to replace in any found application routes")
			importCommand.PersistentFlags().StringVar(&ctx.ExportDir, "export-dir", ctx.ExportDir, "Directory where apps will be placed or read")
			importCommand.SetArgs(tt.args.commandArgs)

			tt.beforeFunc()
			if err := importCommand.Execute(); (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
			}
			tt.afterFunc(ctx)
		})
	}
}

func TestImport_Flags(t *testing.T) {
	type fields struct {
		config      *cli.Config
		commandArgs []string
	}
	tests := []struct {
		name      string
		fields    fields
		wantErr   bool
		want      *cli.Config
		afterFunc func(*testing.T, *cli.Config, *cli.Config, *flag.FlagSet)
	}{
		{
			name: "exclude_orgs from config is set when flag is not given",
			fields: fields{
				config: &cli.Config{
					ExcludedOrgs: []string{"exclude-this-org"},
				},
				commandArgs: []string{},
			},
			want: &cli.Config{
				ExcludedOrgs: []string{"exclude-this-org"},
			},
			afterFunc: func(t *testing.T, expected *cli.Config, actual *cli.Config, flags *flag.FlagSet) {
				require.Equal(t, expected, actual)
			},
		},
		{
			name: "exclude-orgs flag overrides exclude_orgs from config",
			fields: fields{
				config: &cli.Config{
					ExcludedOrgs: []string{"wrong-org"},
				},
				commandArgs: []string{"--exclude-orgs", "correct-org"},
			},
			want: &cli.Config{
				ExcludedOrgs: []string{"correct-org"},
			},
			afterFunc: func(t *testing.T, expected *cli.Config, actual *cli.Config, flags *flag.FlagSet) {
				orgs, err := flags.GetStringSlice("exclude-orgs")
				require.NoError(t, err)
				require.Equal(t, expected.ExcludedOrgs, orgs)
			},
		},
		{
			name: "include_orgs from config is set when flag is not given",
			fields: fields{
				config: &cli.Config{
					IncludedOrgs: []string{"some-org"},
				},
				commandArgs: []string{},
			},
			want: &cli.Config{
				IncludedOrgs: []string{"some-org"},
			},
			afterFunc: func(t *testing.T, expected *cli.Config, actual *cli.Config, flags *flag.FlagSet) {
				require.Equal(t, expected, actual)
			},
		},
		{
			name: "include-orgs flag overrides include_orgs from config",
			fields: fields{
				config: &cli.Config{
					IncludedOrgs: []string{"some-other-org"},
				},
				commandArgs: []string{"--include-orgs", "some-org"},
			},
			want: &cli.Config{
				IncludedOrgs: []string{"some-org"},
			},
			afterFunc: func(t *testing.T, expected *cli.Config, actual *cli.Config, flags *flag.FlagSet) {
				orgs, err := flags.GetStringSlice("include-orgs")
				require.NoError(t, err)
				require.Equal(t, expected.IncludedOrgs, orgs)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Cleanup(func() {
				cache.Cache = nil
			})
			ctx := &context.Context{
				ExportDir: tt.fields.config.ExportDir,
				Metadata:  metadata.NewMetadata(),
				Summary:   report.NewSummary(&bytes.Buffer{}),
			}
			importCommand := cmd.CreateImportCommand(ctx, nil)
			importCommand.RunE = func(cmd *cobra.Command, args []string) error {
				return nil
			}
			importCommand.PersistentFlags().StringArrayVar(&ctx.DomainsToAdd, "domains-to-add", []string{}, "Domains to add in any found application routes")
			importCommand.PersistentFlags().StringToStringVar(&ctx.DomainsToReplace, "domains-to-replace", map[string]string{}, "Domains to replace in any found application routes")
			importCommand.Flags().StringSliceVar(&ctx.IncludedOrgs, "include-orgs", ctx.IncludedOrgs, "Only orgs matching the regex(es) specified will be included")
			importCommand.Flags().StringSliceVar(&ctx.ExcludedOrgs, "exclude-orgs", ctx.ExcludedOrgs, "Any orgs matching the regex(es) specified will be excluded")
			importCommand.SetArgs(tt.fields.commandArgs)
			err := importCommand.Execute()
			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			tt.afterFunc(t, tt.want, tt.fields.config, importCommand.Flags())
		})
	}
}
