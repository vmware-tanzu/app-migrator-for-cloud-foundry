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

package cli

import (
	"fmt"
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/context"
)

func PreRunLoadMetadata(ctx *context.Context) error {
	if err := getLatestRunTimes(ctx); err != nil {
		return fmt.Errorf("error parsing metadata.json: %w", err)
	}
	return nil
}

func PostRunSaveMetadata(ctx *context.Context) error {
	return saveLatestRunTime(ctx)
}

func DisplaySummary(commandCtx *context.Context) {
	commandCtx.Summary.Display()
}

func saveLatestRunTime(commandCtx *context.Context) (err error) {
	metadataFile := filepath.Join(commandCtx.ExportDir, "metadata.json")
	if err := commandCtx.DirWriter.MkdirAll(commandCtx.ExportDir, 0755); err != nil {
		log.Fatal(err)
	}
	var file *os.File
	file, err = os.OpenFile(metadataFile, os.O_WRONLY, 0600)
	defer func(file *os.File) {
		closeErr := file.Close()
		if err != nil {
			err = closeErr
		}
	}(file)

	if err != nil {
		if os.IsNotExist(err) {
			if _, err = os.Create(metadataFile); err != nil {
				log.Fatal(err)
			}
			return
		}
		log.Fatal(err)
	}
	err = commandCtx.Metadata.SaveMetadata(file)
	return
}

func getLatestRunTimes(ctx *context.Context) (err error) {
	f := filepath.Join(ctx.ExportDir, "metadata.json")
	if _, err = os.Stat(f); os.IsNotExist(err) {
		if err = os.MkdirAll(ctx.ExportDir, 0700); err != nil {
			log.Fatalf("failed to create directory %s, %s", ctx.ExportDir, err)
		}
		if _, err = os.Create(f); err != nil {
			log.Fatalf("failed to create file %s, %s", f, err)
		}
	}
	if err != nil {
		return
	}
	var file *os.File
	file, err = os.Open(f)
	if err != nil {
		return
	}
	defer func(file *os.File) {
		closeErr := file.Close()
		if err == nil {
			err = closeErr
		}
	}(file)
	err = ctx.Metadata.LoadMetadata(file)
	return
}
