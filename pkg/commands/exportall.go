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
	"errors"
	"net/url"
	"regexp"
	"strconv"

	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/context"

	"github.com/cloudfoundry-community/go-cfclient"
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/cf"
)

type ExportAll struct {
}

func (e *ExportAll) Run(ctx *context.Context) error {
	page := 1
	var (
		orgs []cfclient.Org
		err  error
	)
	for {
		orgs = []cfclient.Org{}
		err := ctx.ExportCFClient.DoWithRetry(func() error {
			params := url.Values{
				"results-per-page": []string{"50"},
				"page":             []string{strconv.Itoa(page)},
			}

			orgs, err = ctx.ExportCFClient.ListOrgsByQuery(params)
			if err != nil {
				cfErr := cfclient.CloudFoundryHTTPError{}
				if errors.As(err, &cfErr) {
					if cfErr.StatusCode >= 500 && cfErr.StatusCode <= 599 {
						return cf.ErrRetry
					}
				}
			}

			return err
		})
		if err != nil {
			return err
		}

		for _, org := range orgs {
			if isOrgExcluded(ctx, org.Name) || !isOrgIncluded(ctx, org.Name) {
				continue
			}

			exportOrgCmd := &ExportOrg{
				Org: org.Name,
			}

			err := exportOrgCmd.Run(ctx, org.Name)
			if err != nil {
				// TODO: Should we ignore errors here?
				continue
			}
		}

		if len(orgs) < 50 {
			break
		}

		page++
	}

	return nil
}

func isOrgExcluded(ctx *context.Context, orgName string) bool {
	for _, re := range ctx.ExcludedOrgs {
		ok, err := regexp.Match(re, []byte(orgName))
		if ok || err != nil {
			return true
		}
	}

	return false
}

func isOrgIncluded(ctx *context.Context, orgName string) bool {
	if len(ctx.IncludedOrgs) == 0 {
		return true
	}

	for _, re := range ctx.IncludedOrgs {
		ok, err := regexp.Match(re, []byte(orgName))
		if ok || err != nil {
			return true
		}
	}

	return false
}
