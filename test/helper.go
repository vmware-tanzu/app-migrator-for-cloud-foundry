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

//go:build integration
// +build integration

package test

import (
	"context"
	"fmt"
	"go/build"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/cloudfoundry-community/go-cfclient"
	"github.com/go-git/go-git/v5"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/cf"
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/cli"
	aio "github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/io"
	"golang.org/x/sync/errgroup"
)

const (
	packagePath = "github.com/vmware-tanzu/app-migrator-for-cloud-foundry/cmd/app-migrator"
	AppCount = 3
	OrgName     = "app-migrator-test-org"
	SpaceName   = "app-migrator-test-space"
	QuotaName = "runaway"
)

var AppMigratorPath string

func Setup(t *testing.T) {
	InitLogger(log.InfoLevel.String())
	AppMigratorPath = buildAppMigrator(t)
}

func InitLogger(level string) {
	// Output to stdout instead of the default stderr
	log.SetOutput(os.Stdout)
	log.SetFormatter(&log.TextFormatter{
		ForceColors: true,
	})

	if level == "" {
		var ok bool
		level, ok = os.LookupEnv("LOG_LEVEL")
		if !ok {
			level = log.InfoLevel.String()
		}
	}

	l, err := log.ParseLevel(level)
	if err != nil {
		l = log.InfoLevel
	}
	log.SetLevel(l)
}

func SetupExportCommand(t *testing.T) string {
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatalf("could not get current working dir, %s", err)
	}

	cfg := cli.New("", path.Join(cwd, "app-migrator.yml"))

	err = cfg.SourceApi.Validate()
	if err != nil {
		log.Fatalf("%v", err)
	}

	client := NewCFClient(t, &cf.Config{
		Target:      cfg.SourceApi.URL,
		Username:    cfg.SourceApi.Username,
		Password:    cfg.SourceApi.Password,
		SSLDisabled: true,
	})

	DeleteOrg(t, client)
	org := CreateOrg(t, client)

	count := AppCount
	if os.Getenv("TEST_APP_COUNT") != "" {
		count, err = strconv.Atoi(os.Getenv("TEST_APP_COUNT"))
		assert.NoErrorf(t, err, "must set TEST_APP_COUNT env var to an integer")
	}

	dir := CloneTestApp(t, "https://github.com/cloudfoundry-samples/test-app.git", "test-app")
	g, gctx := errgroup.WithContext(context.Background())

	for n := 0; n < count; n++ {
		func(i int) {
			g.Go(func() error {
				cfHome := Login(t, gctx, client)
				space := CreateSpace(t, client, fmt.Sprintf("%s-%d", SpaceName, i), org.Guid)
				return CreateApp(t, gctx, cfHome, fmt.Sprintf("test-app-%d", i), OrgName, space.Name, dir)
			})
		}(n)
	}
	err = g.Wait()
	require.NoError(t, err)

	return cwd
}

func SetupImportCommand(t *testing.T) string {
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatalf("could not get current working dir, %s", err)
	}

	cfg := cli.New("", path.Join(cwd, "app-migrator.yml"))

	err = cfg.TargetApi.Validate()
	if err != nil {
		log.Fatalf("%v", err)
	}

	client := NewCFClient(t, &cf.Config{
		Target:      cfg.TargetApi.URL,
		Username:    cfg.TargetApi.Username,
		Password:    cfg.TargetApi.Password,
		SSLDisabled: true,
	})

	count := AppCount
	if os.Getenv("TEST_APP_COUNT") != "" {
		count, err = strconv.Atoi(os.Getenv("TEST_APP_COUNT"))
		assert.NoErrorf(t, err, "must set TEST_APP_COUNT env var to an integer")
	}

	DeleteOrg(t, client)
	org := CreateOrg(t, client)

	g, _ := errgroup.WithContext(context.Background())

	for n := 0; n < count; n++ {
		func(i int) {
			g.Go(func() error {
				CreateSpace(t, client, fmt.Sprintf("%s-%d", SpaceName, i), org.Guid)
				return nil
			})
		}(n)
	}
	err = g.Wait()
	require.NoError(t, err)

	return cwd
}

func NewCFClient(t *testing.T, config *cf.Config) cf.Client {
	client, err := cf.NewClient(config)
	require.NoError(t, err, "error creating cf client")
	return client
}

func DeleteApp(t *testing.T, client cf.Client, appName string) error {
	org, err := client.GetOrgByName(OrgName)
	if err != nil {
		if cfclient.IsOrganizationNotFoundError(err) {
			return nil
		}
		require.NoError(t, err, "error getting org %s by name", OrgName)
	}

	space, err := client.GetSpaceByName(SpaceName, org.Guid)
	if err != nil {
		if cfclient.IsSpaceNotFoundError(err) {
			return nil
		}
		require.NoError(t, err, "error getting space %s by name from org %s", SpaceName, org.Name)
	}

	app, err := client.AppByName(appName, space.Guid, org.Guid)
	if err != nil {
		if cfclient.IsAppNotFoundError(err) {
			return nil
		}
		require.NoError(t, err, "error getting app %s by name from org %s and space %s", appName, org.Name, space.Name)
	}

	return client.DeleteApp(app.Guid)
}

func DeleteOrg(t *testing.T, client cf.Client) {
	org, err := client.GetOrgByName(OrgName)
	if err != nil {
		if cfclient.IsOrganizationNotFoundError(err) {
			return
		}
		require.NoError(t, err, "error getting org %s by name", OrgName)
	}
	err = client.DeleteOrg(org.Guid, true, false)
	require.NoError(t, err, "error deleting org %s", OrgName)
}

func CreateOrg(t *testing.T, client cf.Client) cfclient.Org {
	org, err := client.CreateOrg(cfclient.OrgRequest{Name: OrgName})
	require.NoErrorf(t, err, "error creating org %s", OrgName)

	return org
}

func CreateSpace(t *testing.T, client cf.Client, spaceName string, orgGuid string) cfclient.Space {
	space, err := client.CreateSpace(cfclient.SpaceRequest{
		Name:             spaceName,
		OrganizationGuid: orgGuid,
	})
	require.NoErrorf(t, err, "error creating space %s", spaceName)

	return space
}

func CreateOrgSpace(t *testing.T, client cf.Client, orgName, spaceName, quotaName string) (cfclient.Org, cfclient.Space) {
	orgQuota, err := client.GetOrgQuotaByName(quotaName)
	assert.NoErrorf(t, err, "error getting org quota %s", quotaName)

	org, err := client.CreateOrg(cfclient.OrgRequest{
		Name: OrgName,
		QuotaDefinitionGuid: orgQuota.Guid,
	})
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

func CloneTestApp(t *testing.T, url string, app string) string {
	dir := os.TempDir()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	appPath := path.Join(dir, app)

	if _, err := os.Stat(appPath); os.IsNotExist(err) {
		_, err := git.PlainCloneContext(ctx, appPath, false, &git.CloneOptions{
			URL:               url,
			RecurseSubmodules: git.DefaultSubmoduleRecursionDepth,
		})
		require.NoErrorf(t, err, "error cloning repo %s to %s", url, dir)
	}

	return appPath
}

func CreateApp(t *testing.T, ctx context.Context, cfHome string, appName string, orgName string, spaceName string, path string) error {
	manifest := filepath.Join(path, "/manifest.yml")
	lines := []string{
		fmt.Sprintf("CF_HOME=%s cf target -o %q -s %q >/dev/null", cfHome, orgName, spaceName),
		fmt.Sprintf("CF_HOME=%s cf push %q -f %s -p %q >/dev/null", cfHome, appName, manifest, path),
	}
	contents := strings.Join(lines, "\n")
	script, err := copyToTempFile(strings.NewReader(contents))
	require.NoError(t, err)
	c, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()
	command := exec.CommandContext(c, "/bin/bash", script.Name())
	command.Stdout = os.Stdout
	command.Stdin = os.Stdin
	command.Stderr = os.Stderr
	return command.Run()
}

func Login(t *testing.T, ctx context.Context, client cf.Client) string {
	id, err := uuid.NewUUID()
	require.NoError(t, err)
	cfHome, err := os.MkdirTemp("", id.String())
	if err != nil {
		panic("failed to create CF_HOME")
	}
	lines := []string{
		fmt.Sprintf("CF_HOME=%s cf api %s --skip-ssl-validation >/dev/null", cfHome, client.Target()),
		fmt.Sprintf("CF_HOME=%s cf auth %q %q >/dev/null", cfHome, client.GetClientConfig().Username, client.GetClientConfig().Password),
	}
	contents := strings.Join(lines, "\n")
	script, err := copyToTempFile(strings.NewReader(contents))
	require.NoError(t, err)
	c, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()
	command := exec.CommandContext(c, "/bin/bash", script.Name())
	command.Stdout = os.Stdout
	command.Stdin = os.Stdin
	command.Stderr = os.Stderr
	err = command.Run()
	require.NoError(t, err)
	return cfHome
}

func copyToTempFile(r io.Reader) (*os.File, error) {
	script, err := aio.CopyToTempFile(r)
	if err != nil {
		return nil, fmt.Errorf("failed to save script to temp file: %v: %w", r, err)
	}

	return script, err
}

func RunMigratorCommand(t *testing.T, args ...string) {
	t.Helper()
	cmd := exec.Command(AppMigratorPath, args...)
	stdOutErr, err := cmd.CombinedOutput()
	if exitErr, ok := err.(*exec.ExitError); ok {
		assert.Equal(t, "0", exitErr.Error())
	}
	t.Log(string(stdOutErr))
}

func buildAppMigrator(t *testing.T) string {
	tmpDir, err := ioutil.TempDir("", "si_artifacts")
	require.NoError(t, err, "Error generating a temp artifact dir")

	executable := filepath.Join(tmpDir, path.Base(packagePath))
	if runtime.GOOS == "windows" {
		executable = executable + ".exe"
	}

	cmdArgs := []string{"build"}
	cmdArgs = append(cmdArgs, "-o", executable, packagePath)

	goBuild := exec.Command("go", cmdArgs...)
	goBuild.Env = replaceGoPath(os.Environ(), build.Default.GOPATH)

	output, err := goBuild.CombinedOutput()
	require.NoErrorf(t, err, "Failed to build %s:\n\nError:\n%s\n\nOutput:\n%s", packagePath, err, string(output))
	return executable
}

func replaceGoPath(environ []string, newGoPath string) []string {
	var newEnviron []string
	for _, v := range environ {
		if !strings.HasPrefix(v, "GOPATH=") {
			newEnviron = append(newEnviron, v)
		}
	}
	return append(newEnviron, "GOPATH="+newGoPath)
}
