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

package cli

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/mitchellh/go-homedir"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/export"
)

type Config struct {
	ConfigDir        string
	ConfigFile       string
	Name             string
	DomainsToReplace map[string]string
	DomainsToAdd     []string        `mapstructure:"domains_to_add"`
	ExportDir        string          `mapstructure:"export_dir"`
	IncludedOrgs     []string        `mapstructure:"include_orgs"`
	ExcludedOrgs     []string        `mapstructure:"exclude_orgs"`
	SourceApi        CloudController `mapstructure:"source_api"`
	TargetApi        CloudController `mapstructure:"target_api"`
	ConcurrencyLimit int             `mapstructure:"concurrency_limit"`
	DisplayProgress  bool            `mapstructure:"display_progress"`
	Debug            bool
}

type CloudController struct {
	URL          string `mapstructure:"url"`
	Username     string `mapstructure:"username"`
	Password     string `mapstructure:"password"`
	ClientID     string `mapstructure:"client_id"`
	ClientSecret string `mapstructure:"client_secret"`
}

func NewDefaultConfig() (*Config, error) {
	configDir := ""

	if cfgHome, ok := os.LookupEnv("APP_MIGRATOR_CONFIG_HOME"); ok {
		configDir = cfgHome
	}

	if configFile, ok := os.LookupEnv("APP_MIGRATOR_CONFIG_FILE"); ok {
		return New(configDir, configFile), nil
	}

	if _, hasSuffix := hasSuffix(configDir); hasSuffix {
		configFile := configDir
		configDir, _ = filepath.Split(configFile)
		return New(configDir, configFile), nil
	}

	return New(configDir, ""), nil
}

func New(configDir string, configFile string) *Config {
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatalf("could not get current working dir, %s", err)
	}

	c := &Config{
		Name:             "app-migrator",
		ConcurrencyLimit: export.DefaultConcurrencyLimit,
		ConfigDir:        configDir,
		ConfigFile:       configFile,
		ExportDir:        path.Join(cwd, "export"),
	}

	c.initViperConfig()

	return c
}

func (c CloudController) Validate() error {
	if c.URL == "" {
		return newFieldError("cf url", errors.New("can't be empty"))
	}
	if c.Username == "" && c.ClientID == "" {
		return newFieldsError([]string{"cf username", "client_id"}, errors.New("can't be empty"))
	}
	if c.Password == "" && c.ClientSecret == "" {
		return newFieldsError([]string{"cf password", "client_secret"}, errors.New("can't be empty"))
	}
	return nil
}

type FieldError struct {
	Field string
	Msg   string
}

func (f FieldError) Error() string {
	return fmt.Sprintf("%s %s", f.Field, f.Msg)
}

func newFieldError(field string, err error) error {
	switch newErr := err.(type) {
	case *FieldError:
		newErr.Field = fmt.Sprintf("%s.%s", field, newErr.Field)
		return newErr
	default:
		return &FieldError{
			Field: field,
			Msg:   err.Error(),
		}
	}
}

func newFieldsError(fields []string, err error) error {
	switch newErr := err.(type) {
	case *FieldError:
		var newFields []string
		for _, f := range fields {
			field := fmt.Sprintf("%s.%s", f, newErr.Field)
			newFields = append(newFields, field)
		}
		newErr.Field = strings.Join(newFields, ",")
		return newErr
	default:
		return &FieldError{
			Field: strings.Join(fields, ","),
			Msg:   err.Error(),
		}
	}
}

/// initConfig reads in config file and ENV variables if set.
func (c *Config) initViperConfig() {
	v := viper.New()
	if c.ConfigFile != "" {
		// Use config file from the flag.
		v.SetConfigFile(c.ConfigFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			log.WithError(err).Error("failed to find home directory")
			os.Exit(1)
		}

		v.SetConfigName(c.Name)
		v.SetConfigType("yaml")
		if c.ConfigDir != "" {
			// Add ConfigDir to the search path
			v.AddConfigPath(c.ConfigDir)
		}
		// Search config in home directory
		v.AddConfigPath(home)
		v.AddConfigPath(fmt.Sprintf("$HOME/.config/%s", c.Name)) // optionally look for config in the XDG_CONFIG_HOME
		v.AddConfigPath(".")                                     // finally, look in the working directory
	}
	v.SetEnvPrefix(c.Name)
	v.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	v.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			log.Fatalf("Failed to load config file: %s, error: %s", v.ConfigFileUsed(), err)
		}
	}
	c.applyViperOverrides(v)
}

func (c *Config) applyViperOverrides(v *viper.Viper) {
	err := v.Unmarshal(c)
	if err != nil {
		log.Fatalf("unable to decode into struct, %v", err)
		return
	}
	if len(v.GetStringMapString("domains_to_replace")) > 0 {
		c.DomainsToReplace = v.GetStringMapString("domains_to_replace")
	}
}

func hasSuffix(configDir string) (string, bool) {
	if strings.HasSuffix(configDir, "yml") {
		return "yml", true
	}
	if strings.HasSuffix(configDir, "yaml") {
		return "yaml", true
	}
	return "", false
}
