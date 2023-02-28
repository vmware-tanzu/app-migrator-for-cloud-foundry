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
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/context"

	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/cf"

	"github.com/cloudfoundry-community/go-cfclient"
)

type limits struct {
	Min float64 `json:"min"`
	Max float64 `json:"max"`
}

type autoscalerRule struct {
	ComparisonMetric string `json:"comparison_metric"`
	Metric           string `json:"metric"`
	QueueName        string `json:"queue_name"`
	RuleSubType      string `json:"rule_sub_type"`
	RuleType         string `json:"rule_type"`
	Threshold        limits `json:"threshold"`
}

type autoscalerRulesResponse struct {
	Pagination pagination       `json:"pagination,omitempty"`
	Resources  []autoscalerRule `json:"resources"`
}

type autoscalerAppInstance struct {
	Enabled        bool   `json:"enabled"`
	InstanceLimits limits `json:"instance_limits"`
}

type autoscalerSchedule struct {
	Enabled        bool    `json:"enabled"`
	ExecutesAt     string  `json:"executes_at"`
	InstanceLimits limits  `json:"instance_limits"`
	Recurrence     float64 `json:"recurrence"`
}

type link struct {
	Href string `json:"href"`
}

type pagination struct {
	TotalPages   float64 `json:"total_pages"`
	TotalResults float64 `json:"total_results"`
	First        link    `json:"first"`
	Last         link    `json:"last"`
	Next         link    `json:"next"`
	Previous     link    `json:"previous"`
}

type autoscalerSchedulesResponse struct {
	Pagination pagination           `json:"pagination,omitempty"`
	Resources  []autoscalerSchedule `json:"resources"`
}

type DefaultAutoScalerExporter struct {
}

func NewAutoScalerExporter() *DefaultAutoScalerExporter {
	return &DefaultAutoScalerExporter{}
}

func (e *DefaultAutoScalerExporter) ExportAutoScalerRules(ctx *context.Context, org cfclient.Org, space cfclient.Space, app cfclient.App, exportDir string) error {
	autoscalerURL := strings.Replace(ctx.ExportCFClient.Target(), "api", "autoscale", 1)
	rulesURL := fmt.Sprintf("%s/api/v2/apps/%s", autoscalerURL, app.Guid)

	ctx.Logger.Infof("exporting autoscaler rules for %s/%s/%s", org.Name, space.Name, app.Name)

	rules := make([]autoscalerRule, 0)
	nextURL := rulesURL + "/rules"

	for {
		respObj := autoscalerRulesResponse{}

		req, err := http.NewRequest(http.MethodGet, nextURL, nil)
		if err != nil {
			return err
		}

		var (
			resp *http.Response
		)
		err = ctx.ExportCFClient.DoWithRetry(func() error {
			resp, err = ctx.ExportCFClient.Do(req)
			if err == nil {
				if resp.StatusCode >= 500 && resp.StatusCode <= 599 {
					defer resp.Body.Close()
					return cf.ErrRetry
				}
			}
			return err
		})

		if err != nil {
			cfErr := &cfclient.CloudFoundryHTTPError{}
			if errors.As(err, cfErr) {
				if cfErr.StatusCode == http.StatusNotFound {
					return nil
				}
			}
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusNotFound {
			return nil
		}

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("unexpected autoscaler status returned: '%s'", http.StatusText(resp.StatusCode))
		}

		if err = json.NewDecoder(resp.Body).Decode(&respObj); err != nil {
			return err
		}

		rules = append(rules, respObj.Resources...)

		if nextURL = respObj.Pagination.Next.Href; nextURL == "" || nextURL == "null" {
			break
		}
	}

	rulesJSONFile := filepath.Join(exportDir, getAppFileName(app.Name)+"_autoscale_rules.json")
	file, err := os.Create(rulesJSONFile)
	if err != nil {
		return err
	}
	defer file.Close()

	if err = json.NewEncoder(file).Encode(rules); err != nil {
		return err
	}

	return nil
}

func (e *DefaultAutoScalerExporter) ExportAutoScalerInstances(ctx *context.Context, org cfclient.Org, space cfclient.Space, app cfclient.App, exportDir string) error {
	var (
		err  error
		req  *http.Request
		resp *http.Response
	)
	autoscalerURL := strings.Replace(ctx.ExportCFClient.Target(), "api", "autoscale", 1)
	instancesURL := fmt.Sprintf("%s/api/v2/apps/%s", autoscalerURL, app.Guid)
	req, err = http.NewRequest(http.MethodGet, instancesURL, nil)
	if err != nil {
		return err
	}

	err = ctx.ExportCFClient.DoWithRetry(func() error {
		resp, err = ctx.ExportCFClient.Do(req)
		if err == nil {
			if resp.StatusCode >= 500 && resp.StatusCode <= 599 {
				defer resp.Body.Close()
				return cf.ErrRetry
			}
		}
		return err
	})
	if err != nil {
		cfErr := &cfclient.CloudFoundryHTTPError{}
		if errors.As(err, cfErr) {
			if cfErr.StatusCode == http.StatusNotFound {
				return nil
			}
		}
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected autoscaler status returned: '%s'", http.StatusText(resp.StatusCode))
	}

	if resp.StatusCode == http.StatusOK {
		ctx.Logger.Infof("exporting autoscaler instance limits for %s/%s/%s\n", org.Name, space.Name, app.Name)
		respObj := autoscalerAppInstance{}

		if err = json.NewDecoder(resp.Body).Decode(&respObj); err != nil {
			return err
		}

		instanceFileName := filepath.Join(exportDir, getAppFileName(app.Name)+"_autoscale_instances.json")
		file, err := os.Create(instanceFileName)
		if err != nil {
			return err
		}
		defer file.Close()

		return json.NewEncoder(file).Encode(respObj)
	}

	return nil
}

func (e *DefaultAutoScalerExporter) ExportAutoScalerSchedules(ctx *context.Context, org cfclient.Org, space cfclient.Space, app cfclient.App, exportDir string) error {
	autoscalerURL := strings.Replace(ctx.ExportCFClient.Target(), "api", "autoscale", 1)
	rulesURL := fmt.Sprintf("%s/api/v2/apps/%s", autoscalerURL, app.Guid)

	req, err := http.NewRequest(http.MethodHead, rulesURL, nil)
	if err != nil {
		return err
	}

	var resp *http.Response
	err = ctx.ExportCFClient.DoWithRetry(func() error {
		resp, err = ctx.ExportCFClient.Do(req)
		if err == nil {
			if resp.StatusCode >= 500 && resp.StatusCode <= 599 {
				defer resp.Body.Close()
				return cf.ErrRetry
			}
		}
		return err
	})
	if err != nil {
		cfErr := &cfclient.CloudFoundryHTTPError{}
		if errors.As(err, cfErr) {
			if cfErr.StatusCode == http.StatusNotFound {
				return nil
			}
		}
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		ctx.Logger.Infof("exporting autoscaler schedules for %s/%s/%s\n", org.Name, space.Name, app.Name)
		schedules := make([]autoscalerSchedule, 0)
		nextURL := rulesURL + "/scheduled_limit_changes"

		for {
			respObj := autoscalerSchedulesResponse{}

			req, err = http.NewRequest(http.MethodGet, nextURL, nil)
			if err != nil {
				return err
			}

			err = ctx.ExportCFClient.DoWithRetry(func() error {
				resp, err = ctx.ExportCFClient.Do(req)
				if err == nil {
					if resp.StatusCode >= 500 && resp.StatusCode <= 599 {
						defer resp.Body.Close()
						return cf.ErrRetry
					}
				}
				return err
			})
			if err != nil {
				return err
			}

			defer resp.Body.Close()

			if err = json.NewDecoder(resp.Body).Decode(&respObj); err != nil {
				return err
			}

			schedules = append(schedules, respObj.Resources...)
			if nextURL = respObj.Pagination.Next.Href; nextURL == "" {
				break
			}
		}

		file, err := os.Create(filepath.Join(exportDir, strings.ReplaceAll(app.Name, "/", "_")+"_autoscale_schedules.json"))
		if err != nil {
			return err
		}
		defer file.Close()

		if err = json.NewEncoder(file).Encode(schedules); err != nil {
			return err
		}
	}

	return nil
}

func getAppFileName(appName string) string {
	return strings.ReplaceAll(appName, "/", "_")
}
