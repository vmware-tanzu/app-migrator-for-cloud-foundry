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
	"fmt"
	"github.com/pkg/errors"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type FileDescriptor struct {
	BaseDir   string
	Name      string
	Extension string
	Org       string
	Space     string
}

const pathSep = string(os.PathSeparator)

var supportedExtensions = []string{".yml", ".yaml"}

type InvalidFileExtensionError struct {
	Ext string
}

func (e *InvalidFileExtensionError) Error() string {
	return fmt.Sprintf("extension %q does not match yml or yaml", e.Ext)
}

func NewFileDescriptor(path string) (FileDescriptor, error) {
	dir, filename := filepath.Split(path)
	ext := filepath.Ext(path)
	if !isSupported(ext) {
		return FileDescriptor{}, &InvalidFileExtensionError{Ext: ext}
	}
	name := strings.TrimSuffix(filename, ext)
	org, space := GetOrgSpace(dir)
	orgSpacePath := filepath.Join(org, space)
	dirs := strings.TrimSuffix(dir, pathSep+orgSpacePath+pathSep)
	return FileDescriptor{
		BaseDir:   dirs,
		Extension: strings.TrimPrefix(filepath.Ext(path), "."),
		Name:      name,
		Org:       org,
		Space:     space,
	}, nil
}

func isSupported(ext string) bool {
	for _, s := range supportedExtensions {
		if s == ext {
			return true
		}
	}
	return false
}

func GetOrgSpace(path string) (string, string) {
	parts := strings.Split(path, pathSep)
	orgSpace := parts[len(parts)-3 : len(parts)-1]

	return orgSpace[0], orgSpace[1]
}

type FlatDir struct {
}

func NewDirWriter() FlatDir {
	return FlatDir{}
}

func (fd FlatDir) Mkdir(dir string) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err := os.MkdirAll(dir, 0755)
		if err != nil {
			return err
		}
	}
	return nil
}

func (fd FlatDir) MkdirAll(dir string, perm os.FileMode) error {
	return os.MkdirAll(dir, perm)
}

func (fd FlatDir) IsEmpty(name string) (bool, error) {
	if _, err := os.Stat(name); os.IsNotExist(err) {
		return true, nil
	}
	f, err := os.Open(name)
	if err != nil {
		return false, err
	}
	defer func(f *os.File) {
		_ = f.Close()
	}(f)

	_, err = f.Readdirnames(1)
	if err == io.EOF {
		return true, nil
	}

	return false, err
}

func CreateFileIfNotExist(f string) (*os.File, error) {
	var file *os.File
	var err error
	if _, err = os.Stat(f); os.IsNotExist(err) {
		dir := filepath.Dir(f)
		if err = os.MkdirAll(dir, 0755); err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("failed to create directory %s", dir))
		}
		if file, err = os.Create(f); err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("failed to create file %s", f))
		}
	}

	return file, err
}

func CopyToTempFile(src io.Reader) (*os.File, error) {
	var file *os.File
	var err error

	file, err = os.CreateTemp("", "")
	if err != nil {
		return nil, err
	}

	if _, err := io.Copy(file, src); err != nil {
		return nil, err
	}

	if err = file.Close(); err != nil {
		return nil, err
	}

	return file, nil
}
