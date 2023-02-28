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

import "strings"

type routeMapper struct {
	DomainsToAdd     []string
	DomainsToReplace map[string]string
}

func (r *routeMapper) AdjustRoutes(existingRoutes []string) []string {
	if len(existingRoutes) == 0 {
		return []string{}
	}

	uriParts := strings.Split(existingRoutes[0], ".")
	host := uriParts[0]

	var adjustedRoutes []string

	for _, r := range r.DomainsToAdd {
		adjustedRoute := host + "." + r
		adjustedRoutes = append(adjustedRoutes, adjustedRoute)
	}

	/*
		for each existing route of an application,
		if there are domain replacement rules:
			for each domain replacement rule:
				check if the existing route qualifies to be replaced (old route matches a rule to replace)
					if qualified, replace and append to final array
					if not qualified, just add directly to array
		if there's no domain replacement rules, append directly to array
	*/

	for _, route := range existingRoutes {
		found := false
		for oldRouteToBeReplaced, newRouteToBeUsed := range r.DomainsToReplace {
			if strings.Contains(route, oldRouteToBeReplaced) {
				// we found an old route in need of replacement
				adjustedRoutes = append(adjustedRoutes, strings.Replace(route, oldRouteToBeReplaced, newRouteToBeUsed, 1))
				found = true
				break
			}
		}
		if !found {
			adjustedRoutes = append(adjustedRoutes, route)
		}
	}

	return adjustedRoutes
}
