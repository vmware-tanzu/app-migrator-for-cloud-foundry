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

package context

import (
	"os"

	"github.com/cloudfoundry-community/go-cfclient"
	log "github.com/sirupsen/logrus"
	"github.com/vbauerster/mpb/v7"
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/cf"
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/metadata"
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/report"
)

// You only need **one** of these per package
//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -generate

//counterfeiter:generate -o fakes . DirWriter

type DirWriter interface {
	Mkdir(dir string) error
	MkdirAll(dir string, perm os.FileMode) error
	IsEmpty(name string) (bool, error)
}

//counterfeiter:generate -o fakes . CommandRunner

type CommandRunner interface {
	Run(ctx *Context) error
}

//counterfeiter:generate -o fakes . OrgCommandRunner

type OrgCommandRunner interface {
	Run(ctx *Context, org string) error
}

//counterfeiter:generate -o fakes . SpaceCommandRunner

type SpaceCommandRunner interface {
	Run(ctx *Context, org, space string) error
}

//counterfeiter:generate -o fakes . AppCommandRunner

type AppCommandRunner interface {
	Run(ctx *Context, orgName, spaceName string) error
	SetAppName(name string)
}

//counterfeiter:generate -o fakes . AppImportRunner

type AppImportRunner interface {
	Run(ctx *Context) error
	SetOrgName(name string)
	SetSpaceName(name string)
	SetAppName(name string)
}

//counterfeiter:generate -o fakes . SpaceImporter

type SpaceImporter interface {
	ImportSpace(ctx *Context, processor ProcessFunc, files []string) (<-chan ProcessResult, error)
}

//counterfeiter:generate -o fakes . SpaceExporter

type SpaceExporter interface {
	ExportSpace(ctx *Context, space cfclient.Space, processor ProcessFunc) (<-chan ProcessResult, error)
}

//counterfeiter:generate -o fakes . DropletExporter

type DropletExporter interface {
	NumberOfPackages(ctx *Context, app cfclient.App) (float64, error)
	DownloadDroplet(ctx *Context, org cfclient.Org, space cfclient.Space, app cfclient.App, exportDir string) error
	DownloadPackages(ctx *Context, org cfclient.Org, space cfclient.Space, app cfclient.App, exportDir string) error
}

//counterfeiter:generate -o fakes . ManifestExporter

type ManifestExporter interface {
	ExportAppManifest(ctx *Context, org cfclient.Org, space cfclient.Space, app cfclient.App, appExportDir string) error
}

//counterfeiter:generate -o fakes . AutoScalerExporter

type AutoScalerExporter interface {
	ExportAutoScalerRules(ctx *Context, org cfclient.Org, space cfclient.Space, app cfclient.App, exportDir string) error
	ExportAutoScalerInstances(ctx *Context, org cfclient.Org, space cfclient.Space, app cfclient.App, exportDir string) error
	ExportAutoScalerSchedules(ctx *Context, org cfclient.Org, space cfclient.Space, app cfclient.App, exportDir string) error
}

type ProcessResult struct {
	Value interface{}
	Err   error
}

type QueryResult struct {
	Value interface{}
}

type ProcessFunc func(*Context, QueryResult) ProcessResult

//counterfeiter:generate -o fakes . QueryResultsProcessor

type QueryResultsProcessor interface {
	ExecuteQuery(ctx *Context, queryResultsCollector QueryResultsCollector, query func(collector QueryResultsCollector) (int, error), processor func(ctx *Context, value QueryResult) ProcessResult) (<-chan ProcessResult, error)
	ExecutePageQuery(ctx *Context, queryResultsCollector QueryResultsCollector, query func(page int, collector QueryResultsCollector) func() (int, error), processor ProcessFunc) (<-chan ProcessResult, error)
}

//counterfeiter:generate -o fakes . QueryResultsCollector

type QueryResultsCollector interface {
	ResultCount() int
	ResultsPerPage() int
	GetResults() <-chan QueryResult
	AddResult(QueryResult)
	Close()
}

type Context struct {
	Debug              bool
	DirWriter          DirWriter
	ExportDir          string
	IncludedOrgs       []string
	ExcludedOrgs       []string
	DomainsToAdd       []string
	DomainsToReplace   map[string]string
	DropletCountToKeep int
	ConcurrencyLimit   int
	Metadata           *metadata.Metadata
	Summary            *report.Summary
	ExportCFClient     cf.Client
	ImportCFClient     cf.Client
	SpaceImporter      SpaceImporter
	SpaceExporter      SpaceExporter
	DropletExporter    DropletExporter
	ManifestExporter   ManifestExporter
	AutoScalerExporter AutoScalerExporter
	Logger             *log.Logger
	Progress           *mpb.Progress
	DisplayProgress    bool
}

func (ctx *Context) InitLogger() {
	logger := log.New()
	ctx.Logger = logger

	var (
		path string
		ok   bool
	)
	if path, ok = os.LookupEnv("APP_MIGRATOR_LOG_FILE"); !ok {
		path = "/tmp/app-migrator.log"
	}

	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		panic(err)
	}

	logger.Formatter = &log.JSONFormatter{
		PrettyPrint: true,
	}
	logger.SetOutput(f)

	var level string
	if ctx.Debug {
		level = log.DebugLevel.String()
	} else {
		var ok bool
		level, ok = os.LookupEnv("LOG_LEVEL")
		if !ok {
			level = log.InfoLevel.String()
		}
	}

	l, err := log.ParseLevel(level)
	if err != nil {
		l = log.InfoLevel
	}
	logger.SetLevel(l)
}
