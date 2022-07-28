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
	"strconv"

	log "github.com/sirupsen/logrus"

	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/context"

	"github.com/cloudfoundry-community/go-cfclient"
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/cache"
	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/cf"
)

type ExportOrg struct {
	Org string `help:"the org to export" short:"o" env:"CF_ORG"`
}

func (e *ExportOrg) Run(ctx *context.Context, orgName string) error {
	globalCache := cache.GetCache(ctx.ExportCFClient)

	org, err := globalCache.GetOrgByName(orgName)
	if err != nil {
		return err
	}

	page := 1
	resultsPerPage := 50

	for {
		params := url.Values{
			"page":             []string{strconv.Itoa(page)},
			"results-per-page": []string{strconv.Itoa(resultsPerPage)},
			"q":                []string{"organization_guid:" + org.Guid},
		}

		var spaces []cfclient.Space
		err = ctx.ExportCFClient.DoWithRetry(func() error {
			spaces, err = ctx.ExportCFClient.ListSpacesByQuery(params)
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

		for _, space := range spaces {
			exportSpaceCmd := &ExportSpace{
				ExportOrg: ExportOrg{
					Org: orgName,
				},

				Space: space.Name,
			}

			err := exportSpaceCmd.Run(ctx, orgName, space.Name)
			if err != nil {
				log.Errorf("%v", err)
				// TODO: Should we ignore errors here?
				continue
			}
		}

		if len(spaces) < resultsPerPage {
			break
		}

		page++
	}

	return nil
}
