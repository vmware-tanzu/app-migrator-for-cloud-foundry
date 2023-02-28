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

package cache

import (
	"errors"
	"fmt"
	"net/url"
	"sync"

	log "github.com/sirupsen/logrus"

	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/cf"

	"github.com/cloudfoundry-community/go-cfclient"
)

type cache struct {
	orgCache   map[string]cfclient.Org   // org guid -> org
	spaceCache map[string]cfclient.Space // space guid -> space
	appCache   map[string]cfclient.App   // app guid -> app

	orgNameCache   map[string]string
	spaceNameCache map[string]map[string]string // space name -> org guids -> space guid
	appNameCache   map[string]map[string]string // app name -> space guids -> app guid

	spaceOrgGUIDCache map[string]string // space guid -> org guid
	appSpaceGUIDCache map[string]string // app guid -> space guid

	stackNameCache map[string]string // stack name -> guid
	stackGUIDCache map[string]string // stack guid -> name

	domainNameCache map[string]string // domain name -> guid

	cf    cf.Client
	mutex sync.RWMutex
}

func GetCache(cf cf.Client) *cache {
	if Cache == nil {
		if cf == nil {
			log.Fatal("cf client is nil")
		}
		Cache = &cache{
			cf: cf,
		}

		Cache.orgCache = make(map[string]cfclient.Org)
		Cache.spaceCache = make(map[string]cfclient.Space)
		Cache.appCache = make(map[string]cfclient.App)

		Cache.orgNameCache = make(map[string]string)
		Cache.spaceNameCache = make(map[string]map[string]string)
		Cache.appNameCache = make(map[string]map[string]string)

		Cache.spaceOrgGUIDCache = make(map[string]string)
		Cache.appSpaceGUIDCache = make(map[string]string)

		Cache.stackNameCache = make(map[string]string)
		Cache.stackGUIDCache = make(map[string]string)

		Cache.domainNameCache = make(map[string]string)
	}

	return Cache
}

func (c *cache) GetOrgByName(name string) (cfclient.Org, error) {
	c.mutex.RLock()
	guid, ok := c.orgNameCache[name]
	c.mutex.RUnlock()
	if !ok {
		c.mutex.RLock()
		if org, ok := c.orgCache[guid]; ok {
			c.mutex.RUnlock()
			return org, nil
		}
		c.mutex.RUnlock()

		var org cfclient.Org
		err := c.cf.DoWithRetry(func() error {
			var err error
			org, err = c.cf.GetOrgByName(name)
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
			return cfclient.Org{}, err
		}

		c.mutex.Lock()
		c.orgNameCache[name] = org.Guid
		c.orgCache[org.Guid] = org
		c.mutex.Unlock()

		return org, nil
	}

	c.mutex.RLock()
	defer c.mutex.RUnlock()

	return c.orgCache[guid], nil
}

func (c *cache) GetOrgByGUID(orgGUID string) (cfclient.Org, error) {
	var (
		org cfclient.Org
		ok  bool
		err error
	)

	c.mutex.RLock()
	if org, ok = c.orgCache[orgGUID]; ok {
		c.mutex.RUnlock()
		return org, nil
	}
	c.mutex.RUnlock()

	if org, err = c.cf.GetOrgByGuid(orgGUID); err != nil {
		return org, err
	}

	c.mutex.Lock()
	c.orgCache[orgGUID] = org
	c.orgNameCache[org.Name] = orgGUID
	c.mutex.Unlock()

	return org, nil
}

func (c *cache) GetSpaceByName(spaceName, orgGUID string) (cfclient.Space, error) {
	c.mutex.RLock()
	guid, ok := c.spaceNameCache[spaceName][orgGUID]
	c.mutex.RUnlock()
	if !ok {
		c.mutex.RLock()
		if space, ok := c.spaceCache[guid]; ok {
			c.mutex.RUnlock()
			return space, nil
		}
		c.mutex.RUnlock()

		var space cfclient.Space
		err := c.cf.DoWithRetry(func() error {
			var err error
			space, err = c.cf.GetSpaceByName(spaceName, orgGUID)
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
			return cfclient.Space{}, err
		}

		c.mutex.Lock()
		c.spaceCache[space.Guid] = space
		if c.spaceNameCache[spaceName] == nil {
			c.spaceNameCache[spaceName] = make(map[string]string)
		}
		c.spaceNameCache[spaceName][orgGUID] = space.Guid
		c.spaceOrgGUIDCache[space.Guid] = orgGUID
		c.mutex.Unlock()

		return space, nil
	}

	c.mutex.RLock()
	defer c.mutex.RUnlock()

	return c.spaceCache[guid], nil
}

func (c *cache) GetSpaceByGUID(spaceGUID string) (cfclient.Space, error) {
	var (
		space cfclient.Space
		ok    bool
		err   error
	)

	c.mutex.RLock()
	if space, ok = c.spaceCache[spaceGUID]; ok {
		c.mutex.RUnlock()
		return space, nil
	}
	c.mutex.RUnlock()

	err = c.cf.DoWithRetry(func() error {
		if space, err = c.cf.GetSpaceByGuid(spaceGUID); err != nil {
			if err != nil {
				cfErr := cfclient.CloudFoundryHTTPError{}
				if errors.As(err, &cfErr) {
					if cfErr.StatusCode >= 500 && cfErr.StatusCode <= 599 {
						return cf.ErrRetry
					}
				}
			}

			return err
		}

		return nil
	})
	if err != nil {
		return cfclient.Space{}, err
	}

	c.mutex.Lock()
	c.spaceCache[spaceGUID] = space
	if c.spaceNameCache[space.Name] == nil {
		c.spaceNameCache[space.Name] = make(map[string]string)
	}
	c.spaceNameCache[space.Name][space.OrganizationGuid] = spaceGUID
	c.spaceOrgGUIDCache[spaceGUID] = space.OrganizationGuid
	c.mutex.Unlock()

	return space, nil
}

func (c *cache) GetAppByName(name, spaceGUID string) (cfclient.App, error) {
	c.mutex.RLock()
	guid, ok := c.appNameCache[name][spaceGUID]
	c.mutex.RUnlock()
	if !ok {
		var apps []cfclient.App
		var err error
		err = c.cf.DoWithRetry(func() error {
			params := url.Values{
				"q":                      []string{"name:" + name, "space_guid:" + spaceGUID},
				"inline-relations-depth": []string{"0"},
			}
			apps, err = c.cf.ListAppsByQuery(params)
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
			return cfclient.App{}, err
		}

		if len(apps) != 1 {
			return cfclient.App{}, &AppNotFoundError{AppName: name, Space: spaceGUID, Count: len(apps)}
		}

		app := apps[0]

		c.mutex.Lock()
		c.appCache[app.Guid] = app
		if c.appNameCache[name] == nil {
			c.appNameCache[name] = make(map[string]string)
		}
		c.appNameCache[name][spaceGUID] = app.Guid
		c.appSpaceGUIDCache[app.Guid] = spaceGUID
		c.mutex.Unlock()

		return app, nil
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()

	return c.appCache[guid], nil
}

func (c *cache) GetAppByGUID(appGUID string) (cfclient.App, bool) {
	c.mutex.RLock()
	if app, ok := c.appCache[appGUID]; ok {
		c.mutex.RUnlock()
		return app, ok
	}
	c.mutex.RUnlock()

	var (
		app cfclient.App
		err error
	)
	err = c.cf.DoWithRetry(func() error {
		app, err = c.cf.GetAppByGuidNoInlineCall(appGUID)
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
		return cfclient.App{}, false
	}

	c.mutex.Lock()
	c.appCache[appGUID] = app
	if c.appNameCache[app.Name] == nil {
		c.appNameCache[app.Name] = make(map[string]string)
	}

	c.appNameCache[app.Name][app.SpaceGuid] = appGUID
	c.appSpaceGUIDCache[app.Guid] = app.SpaceGuid
	c.mutex.Unlock()

	return app, true
}

func (c *cache) GetStackGUIDByName(name string) (string, error) {
	c.mutex.RLock()
	if guid, ok := c.stackNameCache[name]; ok {
		c.mutex.RUnlock()
		return guid, nil
	}
	c.mutex.RUnlock()

	params := url.Values{
		"q": []string{fmt.Sprintf("name:%s", name)},
	}

	var (
		stacks []cfclient.Stack
		err    error
	)
	err = c.cf.DoWithRetry(func() error {
		stacks, err = c.cf.ListStacksByQuery(params)
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
		return "", err
	}

	if len(stacks) != 1 {
		return "", &StackNotFoundError{StackName: name, Count: len(stacks)}
	}

	return stacks[0].Guid, nil
}

func (c *cache) GetStackNameByGUID(guid string) (string, error) {
	c.mutex.RLock()
	if name, ok := c.stackGUIDCache[guid]; ok {
		c.mutex.RUnlock()
		return name, nil
	}
	c.mutex.RUnlock()

	var (
		stack cfclient.Stack
		err   error
	)
	err = c.cf.DoWithRetry(func() error {
		stack, err = c.cf.GetStackByGuid(guid)
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
		return "", err
	}

	c.mutex.Lock()
	c.stackGUIDCache[guid] = stack.Name
	c.stackNameCache[stack.Name] = guid
	c.mutex.Unlock()

	return stack.Name, nil
}

func (c *cache) GetDomainGUIDByName(domain string) (string, error) {
	c.mutex.RLock()
	if guid, ok := c.domainNameCache[domain]; ok {
		c.mutex.RUnlock()
		return guid, nil
	}
	c.mutex.RUnlock()

	domainGUID := ""
	var (
		d   cfclient.Domain
		err error
	)
	err = c.cf.DoWithRetry(func() error {
		d, err = c.cf.GetDomainByName(domain)

		cfErr := cfclient.CloudFoundryHTTPError{}
		if ok := errors.As(err, &cfErr); ok {
			if cfErr.StatusCode >= 500 && cfErr.StatusCode <= 599 {
				return cf.ErrRetry
			}
		}
		return err
	})

	if err != nil {
		var sd cfclient.SharedDomain
		sderr := c.cf.DoWithRetry(func() error {
			var e error
			sd, e = c.cf.GetSharedDomainByName(domain)
			cfErr := cfclient.CloudFoundryHTTPError{}
			if ok := errors.As(e, &cfErr); ok {
				if cfErr.StatusCode >= 500 && cfErr.StatusCode <= 599 {
					return cf.ErrRetry
				}
			}
			return e
		})
		if sderr != nil {
			return "", sderr
		}

		if sd.Guid == "" {
			return "", err
		}

		domainGUID = sd.Guid
	} else {
		domainGUID = d.Guid
	}

	c.mutex.Lock()
	c.domainNameCache[domain] = domainGUID
	c.mutex.Unlock()

	return domainGUID, nil
}

func (c *cache) AddApp(res cfclient.AppResource) cfclient.App {
	app := res.Entity
	app.Guid = res.Meta.Guid

	c.mutex.Lock()
	c.appCache[app.Guid] = app
	c.appSpaceGUIDCache[app.Guid] = app.SpaceGuid
	if c.appNameCache[app.Name] == nil {
		c.appNameCache[app.Name] = make(map[string]string)
	}
	c.appNameCache[app.Name][app.SpaceGuid] = app.Guid
	c.mutex.Unlock()

	return app
}

var Cache *cache
