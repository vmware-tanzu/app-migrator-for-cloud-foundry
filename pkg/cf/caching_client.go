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

package cf

import (
	"github.com/cloudfoundry-community/go-cfclient"
)

type CachingClient struct {
	*cfclient.Client
	cache *Cache
}

func NewCachingClient(client *cfclient.Client) *CachingClient {
	return &CachingClient{
		cache:  NewCache(),
		Client: client,
	}
}

func (c *CachingClient) ListOrgs() ([]cfclient.Org, error) {
	cachedOrgs, ok := c.cache.Load("orgs")
	if ok {
		return cachedOrgs.([]cfclient.Org), nil
	}
	orgs, err := c.Client.ListOrgs()
	if err == nil {
		c.cache.Store("orgs", orgs)
		for _, org := range orgs {
			c.cache.Store("org:"+org.Name, org)
			c.cache.Store("org_id:"+org.Guid, org)
		}
	}
	return orgs, err
}

func (c *CachingClient) ListSpaces() ([]cfclient.Space, error) {
	cachedSpaces, ok := c.cache.Load("spaces")
	if ok {
		return cachedSpaces.([]cfclient.Space), nil
	}
	spaces, err := c.Client.ListSpaces()
	if err == nil {
		c.cache.Store("spaces", spaces)
		for _, space := range spaces {
			c.cache.Store("space:"+space.Name, space)
			c.cache.Store("space_id:"+space.Guid, space)
		}
	}
	return spaces, err
}

func (c *CachingClient) GetOrgByName(name string) (cfclient.Org, error) {
	cachedOrg, ok := c.cache.Load("org:" + name)
	if ok {
		return cachedOrg.(cfclient.Org), nil
	}
	org, err := c.Client.GetOrgByName(name)
	if err == nil {
		c.cache.Store("org:"+org.Name, org)
		c.cache.Store("org_id:"+org.Guid, org)
	}
	return org, err
}

func (c *CachingClient) GetOrgByGuid(orgGUID string) (cfclient.Org, error) {
	cachedOrg, ok := c.cache.Load("org_id:" + orgGUID)
	if ok {
		return cachedOrg.(cfclient.Org), nil
	}
	org, err := c.Client.GetOrgByGuid(orgGUID)
	if err == nil {
		c.cache.Store("org:"+org.Name, org)
		c.cache.Store("org_id:"+org.Guid, org)
	}
	return org, err
}

func (c *CachingClient) GetSpaceByName(name string, orgGUID string) (cfclient.Space, error) {
	cachedSpace, ok := c.cache.Load("space:" + name)
	if ok {
		return cachedSpace.(cfclient.Space), nil
	}
	space, err := c.Client.GetSpaceByName(name, orgGUID)
	if err == nil {
		c.cache.Store("space:"+space.Name, space)
		c.cache.Store("space_id:"+space.Guid, space)
	}
	return space, err
}

func (c *CachingClient) GetSpaceByGuid(spaceGUID string) (cfclient.Space, error) {
	cachedSpace, ok := c.cache.Load("space_id:" + spaceGUID)
	if ok {
		return cachedSpace.(cfclient.Space), nil
	}
	space, err := c.Client.GetSpaceByGuid(spaceGUID)
	if err == nil {
		c.cache.Store("space:"+space.Name, space)
		c.cache.Store("space_id:"+space.Guid, space)
	}
	return space, err
}

func (c *CachingClient) GetServiceByGuid(serviceGUID string) (cfclient.Service, error) {
	cachedService, ok := c.cache.Load("service_id:" + serviceGUID)
	if ok {
		return cachedService.(cfclient.Service), nil
	}
	service, err := c.Client.GetServiceByGuid(serviceGUID)
	if err == nil {
		c.cache.Store("service_id:"+service.Guid, service)
	}
	return service, err
}

func (c *CachingClient) GetServicePlanByGUID(servicePlanGUID string) (*cfclient.ServicePlan, error) {
	cachedServicePlan, ok := c.cache.Load("serviceplan_id:" + servicePlanGUID)
	if ok {
		return cachedServicePlan.(*cfclient.ServicePlan), nil
	}
	servicePlan, err := c.Client.GetServicePlanByGUID(servicePlanGUID)
	if err == nil {
		c.cache.Store("serviceplan_id:"+servicePlan.Guid, servicePlan)
	}
	return servicePlan, err
}
