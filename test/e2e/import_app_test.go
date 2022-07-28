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
	log "github.com/sirupsen/logrus"
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/cf"
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/cli"
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/test"
	"os"
	"path"
	"testing"
)

func Test_ImportAppCommand(t *testing.T) {
	test.Setup(t)
	importApp(t)
}

func importApp(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatalf("could not get current working dir, %s", err)
	}

	cfg := cli.New("", path.Join(cwd, "app-migrator.yml"))

	err = cfg.TargetApi.Validate()
	if err != nil {
		log.Fatalf("%v", err)
	}

	client := test.NewCFClient(t, &cf.Config{
		Target:      cfg.TargetApi.URL,
		Username:    cfg.TargetApi.Username,
		Password:    cfg.TargetApi.Password,
		SSLDisabled: true,
	})

	test.DeleteOrg(t, client)
	test.CreateOrgSpace(t, client, test.OrgName, test.SpaceName, test.QuotaName)
	test.RunMigratorCommand(t, "import", "app", "test-app", "--space", test.SpaceName, "--org", test.OrgName, "--export-dir", path.Join(cwd, "export-app-tests"), "--debug")
}
