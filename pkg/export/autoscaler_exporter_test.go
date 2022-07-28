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

func TestDefaultAutoScalerExporter_ExportAutoScalerInstances(t *testing.T) {
	type args struct {
		ctx       *context.Context
		org       cfclient.Org
		space     cfclient.Space
		app       cfclient.App
		exportDir string
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
			e := &DefaultAutoScalerExporter{}
			if err := e.ExportAutoScalerInstances(tt.args.ctx, tt.args.org, tt.args.space, tt.args.app, tt.args.exportDir); (err != nil) != tt.wantErr {
				t.Errorf("ExportAutoScalerInstances() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDefaultAutoScalerExporter_ExportAutoScalerRules(t *testing.T) {
	type args struct {
		ctx       *context.Context
		org       cfclient.Org
		space     cfclient.Space
		app       cfclient.App
		exportDir string
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
			e := &DefaultAutoScalerExporter{}
			if err := e.ExportAutoScalerRules(tt.args.ctx, tt.args.org, tt.args.space, tt.args.app, tt.args.exportDir); (err != nil) != tt.wantErr {
				t.Errorf("ExportAutoScalerRules() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDefaultAutoScalerExporter_ExportAutoScalerSchedules(t *testing.T) {
	type args struct {
		ctx       *context.Context
		org       cfclient.Org
		space     cfclient.Space
		app       cfclient.App
		exportDir string
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
			e := &DefaultAutoScalerExporter{}
			if err := e.ExportAutoScalerSchedules(tt.args.ctx, tt.args.org, tt.args.space, tt.args.app, tt.args.exportDir); (err != nil) != tt.wantErr {
				t.Errorf("ExportAutoScalerSchedules() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNewAutoScalerExporter(t *testing.T) {
	tests := []struct {
		name string
		want *DefaultAutoScalerExporter
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewAutoScalerExporter(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewAutoScalerExporter() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getAppFileName(t *testing.T) {
	type args struct {
		appName string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getAppFileName(tt.args.appName); got != tt.want {
				t.Errorf("getAppFileName() = %v, want %v", got, tt.want)
			}
		})
	}
}
