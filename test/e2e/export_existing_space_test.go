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

package e2e

import (
	"context"
	"errors"
	"fmt"
	"github.com/cloudfoundry-community/go-cfclient"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/cf"
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/cli"
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/test"
	"golang.org/x/sync/errgroup"
	"net/url"
	"os"
	"path"
	"strconv"
	"testing"
	"time"
)

func Test_ExportExistingSpaceCommand(t *testing.T) {
	test.Setup(t)
	exportExistingSpace(t)
}

func exportExistingSpace(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatalf("could not get current working dir, %s", err)
	}

	cfg := cli.New("", path.Join(cwd, "app-migrator.yml"))

	err = cfg.SourceApi.Validate()
	if err != nil {
		log.Fatalf("%v", err)
	}

	client := test.NewCFClient(t, &cf.Config{
		Target:      cfg.SourceApi.URL,
		Username:    cfg.SourceApi.Username,
		Password:    cfg.SourceApi.Password,
		SSLDisabled: true,
	})

	org, space := test.CreateOrgSpace(t, client, test.OrgName, test.SpaceName, test.QuotaName)

	count := test.AppCount
	if os.Getenv("TEST_APP_COUNT") != "" {
		count, err = strconv.Atoi(os.Getenv("TEST_APP_COUNT"))
		assert.NoErrorf(t, err, "must set TEST_APP_COUNT env var to an integer")
	}

	dir := test.CloneTestApp(t, "https://github.com/cloudfoundry-samples/test-app.git", "test-app")

	q := url.Values{}
	q.Set("inline-relations-depth", "0")
	q.Add("q", fmt.Sprintf("organization_guid:%s", org.Guid))
	q.Add("q", fmt.Sprintf("space_guid:%s", space.Guid))
	log.Infoln("Querying the CF API for apps")
	apps, err := getAppsByQuery(20 * time.Second, client, q)
	var startedApps []cfclient.App
	for _, app := range apps {
		if app.State == string(cfclient.APP_STARTED) {
			startedApps = append(startedApps, app)
		}
	}
	require.NoError(t, err)

	err = createUnstartedApps(t, client, count, startedApps, dir)
	require.NoError(t, err)

	exportDir := path.Join(cwd, "export-space-tests")
	log.Infof("Exporting %s to %s", test.SpaceName, exportDir)
	test.RunMigratorCommand(t, "export", "space", test.SpaceName, "--org", test.OrgName, "--export-dir", exportDir, "--debug")
}

func createUnstartedApps(t *testing.T, client cf.Client, total int, startedApps []cfclient.App, dir string) error {
	t.Helper()
	started := len(startedApps)
	log.Infof("There are %d out of %d started apps", started, total)
	create := total - started
	if create <= 0 {
		return nil
	}
	log.Warnf("Creating %d apps out of %d; this may take awhile", create, total)

	g, gctx := errgroup.WithContext(context.Background())
	for n := started + 1; n <= total; n++ {
		func(i int) {
			g.Go(func() error {
				cfHome := test.Login(t, gctx, client)
				err := test.CreateApp(t, gctx, cfHome, fmt.Sprintf("test-app-%d", i), test.OrgName, test.SpaceName, dir)
				return err
			})
			if create > 20 {
				time.Sleep(3*time.Second) // sleep a little, so we don't DoS the CF api
			}
		}(n)
	}

	return g.Wait()
}

func getAppsByQuery(dur time.Duration, client cf.Client, q url.Values) ([]cfclient.App, error) {
	var result []cfclient.App
	var err error
	done := make(chan struct{})
	go func() {
		result, err = client.ListAppsByQuery(q)
		close(done)
	}()
	select {
	case <-done:
		return result, err
	case <-time.After(dur):
		return []cfclient.App{}, errors.New("timed out fetching apps")
	}
}