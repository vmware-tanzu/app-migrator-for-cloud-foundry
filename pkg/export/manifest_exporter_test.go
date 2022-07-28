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

package export

import (
	"reflect"
	"testing"

	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/context"

	"github.com/cloudfoundry-community/go-cfclient"
)

func TestDefaultManifestExporter_ExportAppManifest(t *testing.T) {
	type args struct {
		ctx          *context.Context
		org          cfclient.Org
		space        cfclient.Space
		app          cfclient.App
		appExportDir string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &DefaultManifestExporter{}
			if err := m.ExportAppManifest(tt.args.ctx, tt.args.org, tt.args.space, tt.args.app, tt.args.appExportDir); (err != nil) != tt.wantErr {
				t.Errorf("ExportAppManifest() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNewManifestExporter(t *testing.T) {
	tests := []struct {
		name string
		want *DefaultManifestExporter
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewManifestExporter(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewManifestExporter() = %v, want %v", got, tt.want)
			}
		})
	}
}
