//go:build import || all
// +build import all

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

package commands_test

import (
	"fmt"
	"github.com/stretchr/testify/require"
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/cmd"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestImportSpace(t *testing.T) {
	var (
		count     = 3
		orgName   = "app-migrator-test-org"
		spaceName = "app-migrator-test-space"
	)

	pwd, err := os.Getwd()
	assert.NoErrorf(t, err, "error should not occur")

	if os.Getenv("TEST_APP_COUNT") != "" {
		count, err = strconv.Atoi(os.Getenv("TEST_APP_COUNT"))
		require.NoErrorf(t, err, "must set TEST_APP_COUNT env var to an integer")
	}

	require.NotEmpty(t, os.Getenv("APP_MIGRATOR_CONFIG_FILE"), "env var APP_MIGRATOR_CONFIG_FILE is not set")

	exportDir := filepath.Join(pwd, "testdata", "integration")
	cmdContext := CreateCmdContext(exportDir)

	rootCmd := cmd.CreateRootCommand(cmdContext)
	rootCmd.SetArgs([]string{"import", "space", spaceName, "-o", orgName, "--export-dir", exportDir})

	t.Cleanup(func() {
		DeleteOrg(t, cmdContext.ImportCFClient, orgName)
	})

	RunCommand(rootCmd, cmdContext)

	assert.True(t,
		cmdContext.Summary.AppSuccessCount() == count,
		fmt.Sprintf("successful app count is %d, expected %d", cmdContext.Summary.AppSuccessCount(), count),
	)
}
