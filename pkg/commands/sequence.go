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
	"fmt"

	"github.com/cloudfoundry-community/go-cfclient"
	"github.com/vbauerster/mpb/v7"
	"github.com/vbauerster/mpb/v7/decor"
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/context"
)

type Sequence interface {
	Run(ctx *context.Context, r Result) (Result, error)
}

type ProgressBarStep struct {
	stepFn  StepFunc
	display string
}

func (p ProgressBarStep) String() string {
	return p.display
}

type StepFunc func(ctx *context.Context, r Result) (Result, error)

func (fn StepFunc) Run(ctx *context.Context, r Result) (Result, error) {
	return fn(ctx, r)
}

type Result interface {
	GetOrg() cfclient.Org
	GetSpace() cfclient.Space
	GetApp() cfclient.App
}

type ExportAppResult struct {
	org   cfclient.Org
	space cfclient.Space
	app   cfclient.App
}

func (r ExportAppResult) GetOrg() cfclient.Org {
	return r.org
}

func (r ExportAppResult) GetSpace() cfclient.Space {
	return r.space
}

func (r ExportAppResult) GetApp() cfclient.App {
	return r.app
}

func RunSequence(msg string, completeMessage string, steps ...*ProgressBarStep) Sequence {
	return StepFunc(func(ctx *context.Context, r Result) (Result, error) {
		var bar *mpb.Bar
		if ctx.Progress != nil {
			p := ctx.Progress
			bar = p.AddBar(int64(len(steps)),
				mpb.PrependDecorators(
					Any(msg, steps, decor.WC{W: len(msg) + 1, C: decor.DSyncSpace}),
					OnComplete(completeMessage, steps, decor.WC{W: len(msg) + 1, C: decor.DSyncSpace}),
				),
				mpb.AppendDecorators(
					decor.Percentage(decor.WCSyncSpace)),
			)
		}

		var err error
		res := r
		for _, step := range steps {
			res, err = step.stepFn(ctx, res)
			if err != nil {
				return res, err
			}
			if bar != nil {
				bar.Increment()
			}
		}
		return res, nil
	})
}

func StepWithProgressBar(step StepFunc, display string) *ProgressBarStep {
	return &ProgressBarStep{stepFn: step, display: display}
}

func Any(msg string, steps []*ProgressBarStep, wcc ...decor.WC) decor.Decorator {
	return decor.Any(func(s decor.Statistics) string {
		if s.Current >= int64(len(steps)) {
			return ""
		}
		return msg
	}, wcc...)
}

func OnComplete(msg string, steps []*ProgressBarStep, wcc ...decor.WC) decor.Decorator {
	return decor.OnComplete(
		decor.Any(func(s decor.Statistics) string {
			return fmt.Sprintf("[\x1b[31m%v\x1b[0m]", steps[s.Current])
		}, wcc...), msg)
}
