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
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"
	"github.com/stretchr/testify/require"
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/cache"
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/cli"
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/context"
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/metadata"
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/report"
)

func TestRootFlags(t *testing.T) {
	type fields struct {
		config      *cli.Config
		command     *cobra.Command
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
			name: "export dir from config is set when flag is not given",
			fields: fields{
				config: &cli.Config{
					ExportDir: "my-export-dir",
				},
				commandArgs: []string{"fake"},
				command:     NewFakeCommand(),
			},
			want: &cli.Config{
				ExportDir: "my-export-dir",
			},
			afterFunc: func(t *testing.T, expected *cli.Config, actual *cli.Config, flags *flag.FlagSet) {
				require.Equal(t, expected, actual)
			},
		},
		{
			name: "export dir flag overrides export dir from config",
			fields: fields{
				config: &cli.Config{
					ExportDir: "wrong-export-dir",
				},
				commandArgs: []string{"fake", "--export-dir", "correct-export-dir"},
				command:     NewFakeCommand(),
			},
			want: &cli.Config{
				ExportDir: "correct-export-dir",
			},
			afterFunc: func(t *testing.T, expected *cli.Config, actual *cli.Config, flags *flag.FlagSet) {
				exportDir, err := flags.GetString("export-dir")
				require.NoError(t, err)
				require.Equal(t, expected.ExportDir, exportDir)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Cleanup(func() {
				_ = os.Unsetenv("APP_MIGRATOR_CONFIG_FILE")
				cache.Cache = nil
			})
			pwd, err := os.Getwd()
			require.NoError(t, err)
			err = os.Setenv("APP_MIGRATOR_CONFIG_FILE", filepath.Join(pwd, "..", "cli", "testdata", "app-migrator.yml"))
			require.NoError(t, err)
			ctx := &context.Context{
				ExportDir: tt.fields.config.ExportDir,
				Metadata:  metadata.NewMetadata(),
				Summary:   report.NewSummary(&bytes.Buffer{}),
			}
			rootCmd := CreateRootCommand(ctx)
			rootCmd.PersistentPreRun = nil
			rootCmd.PersistentPostRun = nil
			rootCmd.AddCommand(tt.fields.command)
			rootCmd.SetArgs(tt.fields.commandArgs)
			err = rootCmd.Execute()
			if (err != nil) != tt.wantErr {
				t.Errorf("rootCmd.Execute() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			tt.afterFunc(t, tt.want, tt.fields.config, rootCmd.Flags())
		})
	}
}

func NewFakeCommand() *cobra.Command {
	return &cobra.Command{
		Use: "fake",
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}
}
