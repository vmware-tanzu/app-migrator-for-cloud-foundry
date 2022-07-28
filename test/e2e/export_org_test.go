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
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/test"
	"path"
	"testing"
)

func TestExportOrgCommand(t *testing.T) {
	test.Setup(t)
	cwd := test.SetupExportCommand(t)
	test.RunMigratorCommand(t, "export", "org", test.OrgName, "--export-dir", path.Join(cwd, "export-org-tests"), "--debug")
}
