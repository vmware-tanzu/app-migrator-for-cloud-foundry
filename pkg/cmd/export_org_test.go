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
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/cache"
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/context"
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/context/fakes"
)

func Test_exportOrg(t *testing.T) {
	fakeCommandRunner1 := new(fakes.FakeOrgCommandRunner)
	fakeCommandRunner2 := new(fakes.FakeOrgCommandRunner)
	fakeCommandRunner3 := new(fakes.FakeOrgCommandRunner)
	type args struct {
		ctx         *context.Context
		runner      *fakes.FakeOrgCommandRunner
		commandArgs []string
	}
	tests := []struct {
		name       string
		args       args
		wantErr    bool
		beforeFunc func()
		afterFunc  func(ctx *context.Context)
	}{
		{
			name: "setting the domains-to-replace flag works",
			args: args{
				ctx: &context.Context{
					DomainsToReplace: map[string]string{},
				},
				runner:      fakeCommandRunner1,
				commandArgs: []string{"org", "some-org", "--domains-to-replace", "foo.com=bar.com"},
			},
			beforeFunc: func() {
				fakeCommandRunner1.RunCalls(func(c *context.Context, org string) error {
					require.Equal(t, "some-org", org)
					require.EqualValues(t, map[string]string{"foo.com": "bar.com"}, c.DomainsToReplace)
					return nil
				})
			},
			afterFunc: func(ctx *context.Context) {
				require.Equal(t, 1, fakeCommandRunner1.RunCallCount())
				require.EqualValues(t, map[string]string{"foo.com": "bar.com"}, ctx.DomainsToReplace)
			},
		},
		{
			name: "setting the domains-to-replace to more than one domain",
			args: args{
				ctx: &context.Context{
					DomainsToReplace: map[string]string{},
				},
				runner:      fakeCommandRunner2,
				commandArgs: []string{"org", "some-org", "--domains-to-replace", "foo1.com=bar1.com,foo2.com=bar2.com"},
			},
			beforeFunc: func() {
				fakeCommandRunner2.RunCalls(func(c *context.Context, org string) error {
					require.Equal(t, "some-org", org)
					require.EqualValues(t, map[string]string{"foo1.com": "bar1.com", "foo2.com": "bar2.com"}, c.DomainsToReplace)
					return nil
				})
			},
			afterFunc: func(ctx *context.Context) {
				require.Equal(t, 1, fakeCommandRunner2.RunCallCount())
				require.EqualValues(t, map[string]string{"foo1.com": "bar1.com", "foo2.com": "bar2.com"}, ctx.DomainsToReplace)
			},
		},
		{
			name: "more than one domain mapping to the same domain",
			args: args{
				ctx: &context.Context{
					DomainsToReplace: map[string]string{},
				},
				runner:      fakeCommandRunner3,
				commandArgs: []string{"org", "some-org", "--domains-to-replace", "foo1.com=bar1.com,foo2.com=bar1.com"},
			},
			beforeFunc: func() {
				fakeCommandRunner3.RunCalls(func(c *context.Context, org string) error {
					require.Equal(t, "some-org", org)
					require.EqualValues(t, map[string]string{"foo1.com": "bar1.com", "foo2.com": "bar1.com"}, c.DomainsToReplace)
					return nil
				})
			},
			afterFunc: func(ctx *context.Context) {
				require.Equal(t, 1, fakeCommandRunner3.RunCallCount())
				require.EqualValues(t, map[string]string{"foo1.com": "bar1.com", "foo2.com": "bar1.com"}, ctx.DomainsToReplace)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Cleanup(func() {
				cache.Cache = nil
			})
			ctx := tt.args.ctx
			rootCmd := &cobra.Command{}
			exportSpaceCmd := CreateExportOrgCommand(ctx, tt.args.runner)
			rootCmd.PersistentFlags().StringToStringVar(&ctx.DomainsToReplace, "domains-to-replace", map[string]string{}, "Domains to replace in any found application routes")
			rootCmd.AddCommand(exportSpaceCmd)
			rootCmd.SetArgs(tt.args.commandArgs)

			tt.beforeFunc()
			if err := rootCmd.Execute(); (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
			}
			tt.afterFunc(ctx)
		})
	}
}
