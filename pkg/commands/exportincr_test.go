//go:build !integration || all
// +build !integration all

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
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/context/fakes"

	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/context"

	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/cache"
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/export"
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/metadata"
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/report"
	. "github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/testsupport"
)

func TestExportIncremental(t *testing.T) {
	pwd, _ := os.Getwd()
	logger := log.New()
	tests := []struct {
		name             string
		context          *context.Context
		want             string
		wantErr          bool
		shouldContain    string
		shouldNotContain string
		assertion        AssertionFunc
		app              ExportIncremental
		handler          http.Handler
		err              error
	}{
		{
			name:    "success",
			app:     ExportIncremental{},
			handler: ExportTestHandler(t),
			context: &context.Context{
				Logger:    logger,
				ExportDir: filepath.Join(pwd, "testdata/apps"),
				DirWriter: &fakes.FakeDirWriter{
					MkdirStub: func(s string) error {
						return nil
					},
				},
				Metadata:           metadata.NewMetadata(),
				Summary:            report.NewSummary(&bytes.Buffer{}),
				DropletExporter:    export.NewDropletExporter(),
				ManifestExporter:   export.NewManifestExporter(),
				AutoScalerExporter: export.NewAutoScalerExporter(),
			},
			shouldNotContain: "Error occurred",
			assertion:        SuccessfulAppCount(t, 1),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Cleanup(func() {
				cache.Cache = nil
			})
			s := httptest.NewServer(tt.handler)
			defer s.Close()
			tt.context.ExportCFClient = NewTestCFClient(t, s)
			err := tt.app.Run(tt.context)
			require.NoError(t, err)
		})
	}
}
