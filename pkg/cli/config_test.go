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

package cli_test

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/cache"
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/cli"
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/export"
)

func TestNewDefaultConfig(t *testing.T) {
	pwd, err := os.Getwd()
	require.NoError(t, err)
	tests := []struct {
		name       string
		want       *cli.Config
		wantErr    bool
		beforeFunc func()
	}{
		{
			name: "creates a valid config when config dir is a file",
			want: &cli.Config{
				ConfigDir:        filepath.Join(pwd, "testdata") + "/",
				ConfigFile:       filepath.Join(pwd, "testdata", "config_no_orgs.yml"),
				Name:             "app-migrator",
				ConcurrencyLimit: export.DefaultConcurrencyLimit,
				DomainsToReplace: map[string]string{
					"apps.cf1.example.com": "apps.cf2.example.com",
				},
				ExportDir:    filepath.Join(pwd, "export"),
				IncludedOrgs: nil,
				ExcludedOrgs: []string{},
				Debug:        false,
			},
			wantErr: false,
			beforeFunc: func() {
				err = os.Setenv("APP_MIGRATOR_CONFIG_HOME", filepath.Join(pwd, "testdata", "config_no_orgs.yml"))
				require.NoError(t, err)
			},
		},
		{
			name: "creates a valid config when file location is specified",
			want: &cli.Config{
				ConfigDir:        "",
				ConfigFile:       filepath.Join(pwd, "testdata", "config_no_orgs.yml"),
				Name:             "app-migrator",
				ConcurrencyLimit: export.DefaultConcurrencyLimit,
				DomainsToReplace: map[string]string{
					"apps.cf1.example.com": "apps.cf2.example.com",
				},
				ExportDir:    filepath.Join(pwd, "export"),
				IncludedOrgs: nil,
				ExcludedOrgs: []string{},
				Debug:        false,
			},
			wantErr: false,
			beforeFunc: func() {
				err = os.Setenv("APP_MIGRATOR_CONFIG_FILE", filepath.Join(pwd, "testdata", "config_no_orgs.yml"))
				require.NoError(t, err)
			},
		},
	}
	for _, tt := range tests {
		_ = os.Unsetenv("APP_MIGRATOR_CONFIG_FILE")
		_ = os.Unsetenv("APP_MIGRATOR_CONFIG_HOME")
		t.Run(tt.name, func(t *testing.T) {
			t.Cleanup(func() {
				cache.Cache = nil
			})
			if tt.beforeFunc != nil {
				tt.beforeFunc()
			}
			got, err := cli.NewDefaultConfig()
			if (err != nil) != tt.wantErr {
				t.Errorf("NewDefaultConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewDefaultConfig() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewConfig(t *testing.T) {
	pwd, err := os.Getwd()
	require.NoError(t, err)
	type args struct {
		configFile string
		configDir  string
	}
	tests := []struct {
		name    string
		args    args
		want    *cli.Config
		wantErr bool
	}{
		{
			name: "creates a default config when config found in dir",
			args: args{
				configDir: filepath.Join(pwd, "testdata"),
			},
			want: &cli.Config{
				ConfigDir:        filepath.Join(pwd, "testdata"),
				ConfigFile:       "",
				Name:             "app-migrator",
				ConcurrencyLimit: export.DefaultConcurrencyLimit,
				DomainsToReplace: map[string]string{
					"apps.cf1.example.com": "apps.cf2.example.com",
				},
				ExportDir: "service-export",
				SourceApi: cli.CloudController{
					URL:          "https://api.cf1.example.com",
					Username:     "cf1-api-username",
					Password:     "cf1-api-password",
					ClientID:     "cf1-api-client",
					ClientSecret: "cf1-api-client-secret",
				},
				TargetApi: cli.CloudController{
					URL:          "https://api.cf2.example.com",
					Username:     "cf2-api-username",
					Password:     "cf2-api-password",
					ClientID:     "cf2-api-client",
					ClientSecret: "cf2-api-client-secret",
				},
				IncludedOrgs: nil,
				ExcludedOrgs: nil,
				Debug:        false,
			},
			wantErr: false,
		},
		{
			name: "creates a config from provided file",
			args: args{
				configFile: filepath.Join(pwd, "testdata", "config_no_orgs.yml"),
				configDir:  filepath.Join(pwd, "testdata"),
			},
			want: &cli.Config{
				ConfigDir:        filepath.Join(pwd, "testdata"),
				ConfigFile:       filepath.Join(pwd, "testdata", "config_no_orgs.yml"),
				Name:             "app-migrator",
				ConcurrencyLimit: export.DefaultConcurrencyLimit,
				DomainsToReplace: map[string]string{
					"apps.cf1.example.com": "apps.cf2.example.com",
				},
				ExportDir:    filepath.Join(pwd, "export"),
				IncludedOrgs: nil,
				ExcludedOrgs: []string{},
				Debug:        false,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Cleanup(func() {
				cache.Cache = nil
			})
			got := cli.New(tt.args.configDir, tt.args.configFile)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("New() got = %+v, want %+v", got, tt.want)
			}
		})
	}
}
