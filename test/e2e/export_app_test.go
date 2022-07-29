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
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/cf"
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/cli"
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/test"
	"golang.org/x/sync/errgroup"
	"os"
	"path"
	"testing"
)

func Test_ExportAppCommand(t *testing.T) {
	test.Setup(t)
	exportApp(t)
}

func exportApp(t *testing.T) {
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

	dir := test.CloneTestApp(t, "https://github.com/cloudfoundry-samples/test-app.git", "test-app")
	g, gctx := errgroup.WithContext(context.Background())

	g.Go(func() error {
		cfHome := test.Login(t, gctx, client)
		return test.CreateApp(t, gctx, cfHome, "test-app", test.OrgName, test.SpaceName, dir)
	})
	err = g.Wait()
	require.NoError(t, err)

	test.RunMigratorCommand(t, "export", "app", "test-app", "--space", test.SpaceName, "--org", test.OrgName, "--export-dir", path.Join(cwd, "export-app-tests"), "--debug")
}
