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

//go:build export || all
// +build export all

package commands_test

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/cf"
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/cmd"
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/export"
	aio "github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/io"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/cli"
	cmdcontext "github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/context"
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/metadata"
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/report"

	"github.com/cloudfoundry-community/go-cfclient"
	"github.com/stretchr/testify/assert"
)

func TestExportSpace(t *testing.T) {
	var (
		count     = 3
		limit     = export.DefaultConcurrencyLimit
		orgName   = "app-migrator-test-org"
		spaceName = "app-migrator-test-space"
	)

	pwd, err := os.Getwd()
	require.NoErrorf(t, err, "error should not occur")

	if os.Getenv("TEST_APP_COUNT") != "" {
		count, err = strconv.Atoi(os.Getenv("TEST_APP_COUNT"))
		require.NoErrorf(t, err, "must set TEST_APP_COUNT env var to an integer")
	}

	if os.Getenv("CONCURRENCY_LIMIT") != "" {
		limit, err = strconv.Atoi(os.Getenv("CONCURRENCY_LIMIT"))
		require.NoErrorf(t, err, "must set CONCURRENCY_LIMIT env var to an integer")
	}

	require.NotEmpty(t, os.Getenv("APP_MIGRATOR_CONFIG_FILE"), "env var APP_MIGRATOR_CONFIG_FILE is not set")

	exportDir := filepath.Join(pwd, "testdata", "integration")
	cmdContext := CreateCmdContext(exportDir)

	rootCmd := cmd.CreateRootCommand(cmdContext)
	rootCmd.SetArgs([]string{"export", "space", spaceName, "-o", orgName, "--export-dir", exportDir, "--concurrency-limit", strconv.Itoa(limit)})

	CreateOrgSpace(t, cmdContext.ExportCFClient, orgName, spaceName)

	_, err = SeedApps(t, cmdContext, orgName, spaceName, count)
	require.NoErrorf(t, err, "error seeding apps")

	keep, _ := strconv.ParseBool(os.Getenv("KEEP_TEST_APPS"))
	if !keep {
		t.Cleanup(func() {
			DeleteOrg(t, cmdContext.ExportCFClient, orgName)
		})
	}

	RunCommand(rootCmd, cmdContext)

	assert.True(t,
		cmdContext.Summary.AppSuccessCount() == count,
		fmt.Sprintf("successful app count is %d, expected %d", cmdContext.Summary.AppSuccessCount(), count),
	)
}

func CreateCmdContext(exportDir string) *cmdcontext.Context {
	ctx := &cmdcontext.Context{
		DirWriter:          aio.NewDirWriter(),
		ExportDir:          exportDir,
		Metadata:           metadata.NewMetadata(),
		Summary:            report.NewSummary(os.Stdout),
		DropletExporter:    export.NewDropletExporter(),
		ManifestExporter:   export.NewManifestExporter(),
		AutoScalerExporter: export.NewAutoScalerExporter(),
	}
	ctx.InitLogger()
	return ctx
}

func RunCommand(cmd *cobra.Command, c *cmdcontext.Context) {
	if err := cmd.Execute(); err != nil {
		log.Fatal(err)
	}

	cli.DisplaySummary(c)
}

func SeedApps(t *testing.T, cmdContext *cmdcontext.Context, orgName, spaceName string, num int) ([]cfclient.AppResource, error) {
	path := os.Getenv("TEST_APP_PATH")
	if os.Getenv("TEST_APP_PATH") == "" {
		t.Fatalf("TEST_APP_PATH not set")
	}

	var apps []cfclient.AppResource
	for i := 0; i < num; i++ {
		apps = append(apps, cfclient.AppResource{
			Entity: cfclient.App{
				Name: fmt.Sprintf("test-app-%d", i+1),
			},
		})
	}
	t.Logf("Seeding cf with '%d' apps; this may take awhile", num)

	pCtx, pCancel := context.WithTimeout(context.Background(), 15*time.Minute) // timeout if loop takes over 15min
	defer pCancel()
	type wrapper struct {
		result cfclient.AppResource
		err    error
	}
	ch := make(chan wrapper, 1)
	var pushedApps []cfclient.AppResource
	for i, a := range apps {
		go func(i int, app cfclient.AppResource) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute) // timeout if a push takes over 5min
			defer cancel()
			err := PushApp(t, ctx, cmdContext.ExportCFClient, orgName, spaceName, app.Entity.Name, path)
			if err != nil {
				assert.NoError(t, err, fmt.Sprintf("failed to run 'cf push %s'", app.Entity.Name))
			}
			ch <- wrapper{app, err}
		}(i, a)
	}
	for {
		select {
		case data := <-ch:
			if data.err != nil {
				return pushedApps, data.err
			}
			pushedApps = append(pushedApps, data.result)
			if len(pushedApps) == num {
				return pushedApps, nil
			}
			continue
		case <-pCtx.Done():
			return pushedApps, pCtx.Err()
		}
	}
}

func CreateOrgSpace(t *testing.T, client cf.Client, orgName, spaceName string) (cfclient.Org, cfclient.Space) {
	org, err := client.CreateOrg(cfclient.OrgRequest{Name: orgName})
	if err != nil {
		if !cfclient.IsOrganizationNameTakenError(err) {
			require.NoErrorf(t, err, "error creating org %s", orgName)
		}
		org, err = client.GetOrgByName(orgName)
		require.NoError(t, err, "error getting org %s by name", orgName)
	}

	space, err := client.CreateSpace(cfclient.SpaceRequest{
		Name:             spaceName,
		OrganizationGuid: org.Guid,
	})
	if err != nil {
		if !cfclient.IsSpaceNameTakenError(err) {
			require.NoErrorf(t, err, "error creating space %s", spaceName)
		}
		space, err = client.GetSpaceByName(spaceName, org.Guid)
		require.NoError(t, err, "error getting space %s by name", spaceName)
	}

	return org, space
}

func PushApp(t *testing.T, ctx context.Context, client cf.Client, orgName, spaceName, appName string, path string) error {
	id, err := uuid.NewUUID()
	require.NoError(t, err)
	cfHome, err := os.MkdirTemp("", id.String())
	if err != nil {
		panic("failed to create CF_HOME")
	}
	type wrapper struct {
		result string
		err    error
	}
	ch := make(chan wrapper, 1)
	go func() {
		manifest := filepath.Join(path, "/manifest.yml")
		lines := []string{
			fmt.Sprintf("CF_HOME=%s cf api %s --skip-ssl-validation > /dev/null 2>&1", cfHome, client.Target()),
			fmt.Sprintf("CF_HOME=%s cf auth %q %q > /dev/null 2>&1", cfHome, client.GetClientConfig().Username, client.GetClientConfig().Password),
			fmt.Sprintf("CF_HOME=%s cf target -o %q -s %q > /dev/null 2>&1", cfHome, orgName, spaceName),
			fmt.Sprintf("CF_HOME=%s cf push %q -f %s -p %q > /dev/null 2>&1", cfHome, appName, manifest, path),
		}
		contents := strings.Join(lines, "\n")
		script, err := copyToTempFile(strings.NewReader(contents))
		command := exec.Command("/bin/bash", script.Name())
		command.Stdout = os.Stdout
		command.Stdin = os.Stdin
		command.Stderr = os.Stderr
		err = command.Run()
		ch <- wrapper{"", err}
	}()
	select {
	case data := <-ch:
		return data.err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func DeleteOrg(t *testing.T, client cf.Client, orgName string) {
	org, err := client.GetOrgByName(orgName)
	if err != nil {
		if cfclient.IsOrganizationNotFoundError(err) {
			return
		}
		require.NoError(t, err, "error getting org %s by name", orgName)
	}
	err = client.DeleteOrg(org.Guid, true, false)
	require.NoError(t, err, "error deleting org %s", orgName)
}

func copyToTempFile(r io.Reader) (*os.File, error) {
	script, err := aio.CopyToTempFile(r)
	if err != nil {
		return nil, fmt.Errorf("failed to save script to temp file: %v: %w", r, err)
	}

	return script, err
}
