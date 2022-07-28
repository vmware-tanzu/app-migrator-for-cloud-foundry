//go:build integration
// +build integration

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

package e2e

import (
	"context"
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/cf"
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/cli"
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/test"
	"golang.org/x/sync/errgroup"
	"os"
	"path"
	"strconv"
	"testing"
)

func Test_ExportOrgSpaceCommand(t *testing.T) {
	test.Setup(t)
	exportOrgSpace(t)
}

func exportOrgSpace(t *testing.T) {
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

	test.DeleteOrg(t, client)
	test.CreateOrgSpace(t, client, test.OrgName, test.SpaceName, test.QuotaName)

	count := test.AppCount
	if os.Getenv("TEST_APP_COUNT") != "" {
		count, err = strconv.Atoi(os.Getenv("TEST_APP_COUNT"))
		assert.NoErrorf(t, err, "must set TEST_APP_COUNT env var to an integer")
	}

	dir := test.CloneTestApp(t, "https://github.com/cloudfoundry-samples/test-app.git", "test-app")
	g, gctx := errgroup.WithContext(context.Background())

	for n := 0; n < count; n++ {
		func(i int) {
			g.Go(func() error {
				cfHome := test.Login(t, gctx, client)
				return test.CreateApp(t, gctx, cfHome, fmt.Sprintf("test-app-%d", i), test.OrgName, test.SpaceName, dir)
			})
		}(n)
	}
	err = g.Wait()
	require.NoError(t, err)

	test.RunMigratorCommand(t, "export", "space", test.SpaceName, "--org", test.OrgName, "--export-dir", path.Join(cwd, "export-space-tests"), "--debug")
}
