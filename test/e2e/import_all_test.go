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
	"fmt"
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/test"
	"path"
	"strings"
	"testing"
)

func TestImportCommand_ExcludeOrgs(t *testing.T) {
	test.Setup(t)
	cwd := test.SetupImportCommand(t)
	test.RunMigratorCommand(t, "import", "--export-dir", path.Join(cwd, fmt.Sprintf("export-all-but-%s-tests", test.OrgName)), "--exclude-orgs", strings.Join([]string{"^system$", "^broker$", "p-*"}, ","), "--debug")
}

func TestImportCommand_IncludeOrgs(t *testing.T) {
	test.Setup(t)
	cwd := test.SetupImportCommand(t)
	test.RunMigratorCommand(t, "import", "--export-dir", path.Join(cwd, fmt.Sprintf("export-%s-tests", test.OrgName)), "--include-orgs", test.OrgName, "--debug")
}
