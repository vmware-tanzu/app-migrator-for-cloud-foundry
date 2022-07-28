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

package commands

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/cache"
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/context/fakes"

	"github.com/stretchr/testify/assert"
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/context"
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/metadata"
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/report"
	. "github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/testsupport"
)

func TestImportAll_Run(t *testing.T) {
	type args struct {
		ctx *context.Context
	}

	pwd, err := os.Getwd()
	assert.NoErrorf(t, err, "error should not occur")

	tests := []struct {
		name               string
		args               args
		handler            http.Handler
		successfulAppCount int
	}{
		{
			name: "returns import error",
			args: args{
				ctx: &context.Context{
					ExportDir: filepath.Join(pwd, "testdata/apps"),
					Metadata:  metadata.NewMetadata(),
					Summary:   report.NewSummary(&bytes.Buffer{}),
					SpaceImporter: &fakes.FakeSpaceImporter{
						ImportSpaceStub: func(ctx *context.Context, processFunc context.ProcessFunc, strings []string) (<-chan context.ProcessResult, error) {
							results := make(chan context.ProcessResult, 1)
							defer close(results)
							results <- context.ProcessResult{Value: "my_app"}
							ctx.Summary.AddSuccessfulApp("my_org", "my_space", "my_app")
							return results, nil
						},
					},
				},
			},
			handler:            ImportTestHandler(t),
			successfulAppCount: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Cleanup(func() {
				cache.Cache = nil
			})
			i := &ImportAll{}
			s := httptest.NewServer(tt.handler)
			defer s.Close()
			tt.args.ctx.ImportCFClient = NewTestCFClient(t, s)
			err := i.Run(tt.args.ctx)
			require.NoError(t, err)
			assert.True(t,
				tt.args.ctx.Summary.AppSuccessCount() == tt.successfulAppCount,
				fmt.Sprintf("successful app count is %d, expected %d", tt.args.ctx.Summary.AppSuccessCount(), tt.successfulAppCount),
			)
		})
	}
}
