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

package io

import (
	"github.com/stretchr/testify/require"
	"io"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestCopyToTempFile(t *testing.T) {
	type args struct {
		src io.Reader
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "data is written to file",
			args: args{
				src: strings.NewReader("some data"),
			},
			want:    "some data",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CopyToTempFile(tt.args.src)
			if (err != nil) != tt.wantErr {
				t.Errorf("CopyToTempFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			defer func(name string) {
				_ = os.Remove(name)
			}(got.Name())
			actual, err := os.ReadFile(got.Name())
			require.NoError(t, err)
			if string(actual) != tt.want {
				t.Errorf("CopyToTempFile() got = %v, want %v", string(actual), tt.want)
			}
		})
	}
}

func TestCreateFileIfNotExist(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "")
	require.NoError(t, err)
	expectedFilePath := filepath.Join(tmpDir, "my-service", "broker.yml")
	type args struct {
		f string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "creates file if doesn't exist",
			args: args{
				f: expectedFilePath,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CreateFileIfNotExist(tt.args.f)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateFileIfNotExist() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			require.Equal(t, filepath.Dir(got.Name()), filepath.Dir(expectedFilePath))
		})
	}
}

func TestGetOrgSpace(t *testing.T) {
	pwd, err := os.Getwd()
	require.NoError(t, err)
	f := filepath.Join(pwd, "testdata", "cloudfoundry", "test-app", "some-app.yml")
	type args struct {
		path string
	}
	tests := []struct {
		name  string
		args  args
		org   string
		space string
	}{
		{
			name: "parses out the org and space tuple",
			args: args{
				path: f,
			},
			org:   "cloudfoundry",
			space: "test-app",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			org, space := GetOrgSpace(tt.args.path)
			if org != tt.org {
				t.Errorf("GetOrgSpace() org = %v, org %v", org, tt.org)
			}
			if space != tt.space {
				t.Errorf("GetOrgSpace() space = %v, space %v", space, tt.space)
			}
		})
	}
}

func TestMkdir(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "")
	require.NoError(t, err)
	dir := path.Join(tmpDir, "org", "spacey")
	type args struct {
		dir string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "creates temp dir",
			args: args{
				dir: dir,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		fd := FlatDir{}
		t.Run(tt.name, func(t *testing.T) {
			if err := fd.Mkdir(tt.args.dir); (err != nil) != tt.wantErr {
				t.Errorf("Mkdir() error = %v, wantErr %v", err, tt.wantErr)
			}
			require.DirExists(t, dir, "directory exists")
		})
	}
}

func TestNewFileDescriptor(t *testing.T) {
	pwd, err := os.Getwd()
	require.NoError(t, err)
	type args struct {
		path string
	}
	tests := []struct {
		name    string
		args    args
		want    FileDescriptor
		wantErr *InvalidFileExtensionError
	}{
		{
			name: "creates an accurate file descriptor",
			args: args{
				path: filepath.Join(pwd, "testdata", "cloudfoundry", "test-app", "cups-test.yml"),
			},
			want: FileDescriptor{
				Name:      "cups-test",
				Org:       "cloudfoundry",
				Space:     "test-app",
				BaseDir:   filepath.Join(pwd, "testdata"),
				Extension: "yml",
			},
		},
		{
			name: "returns error if not yaml extension",
			args: args{
				path: filepath.Join(pwd, "testdata", "backup.tar"),
			},
			want:    FileDescriptor{},
			wantErr: &InvalidFileExtensionError{Ext: ".tar"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewFileDescriptor(tt.args.path)
			if tt.wantErr != nil {
				require.EqualError(t, err, tt.wantErr.Error())
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewFileDescriptor() = %v, want %v", got, tt.want)
			}
		})
	}
}
