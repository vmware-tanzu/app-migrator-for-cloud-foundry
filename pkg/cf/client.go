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
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/cloudfoundry-community/go-cfclient"
)

const (
	// DefaultRetryTimeout sets the amount of time before a retry times out
	DefaultRetryTimeout = time.Minute
	// DefaultRetryPause sets the amount of time to wait before retrying
	DefaultRetryPause = 3 * time.Second
)

var ErrRetry = errors.New("retry")

// You only need **one** of these per package
//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -generate

// APIClient defines requests that can be made to the APIClient API
type APIClient interface {
	BindRoute(routeGUID, appGUID string) error

	CreateApp(request cfclient.AppCreateRequest) (cfclient.App, error)
	CreateOrg(req cfclient.OrgRequest) (cfclient.Org, error)
	CreateSpace(req cfclient.SpaceRequest) (cfclient.Space, error)
	CreateRoute(request cfclient.RouteRequest) (cfclient.Route, error)
	CreateServiceBinding(appGUID, serviceInstanceGUID string) (*cfclient.ServiceBinding, error)

	DeleteApp(guid string) error
	DeleteOrg(guid string, recursive, async bool) error

	DoRequest(req *cfclient.Request) (*http.Response, error)
	Do(req *http.Request) (*http.Response, error)

	AppByName(appName, spaceGuid, orgGuid string) (cfclient.App, error)
	GetAppByGuidNoInlineCall(guid string) (cfclient.App, error)
	GetDomainByName(name string) (cfclient.Domain, error)
	GetOrgByGuid(guid string) (cfclient.Org, error)
	GetOrgByName(name string) (cfclient.Org, error)
	GetOrgQuotaByName(name string) (cfclient.OrgQuota, error)
	GetSharedDomainByName(name string) (cfclient.SharedDomain, error)
	GetSpaceByGuid(guid string) (cfclient.Space, error)
	GetSpaceByName(name string, orgGUID string) (cfclient.Space, error)
	GetStackByGuid(guid string) (cfclient.Stack, error)

	ListOrgs() ([]cfclient.Org, error)
	ListSpaces() ([]cfclient.Space, error)

	ListAppsByQuery(params url.Values) ([]cfclient.App, error)
	ListOrgsByQuery(params url.Values) ([]cfclient.Org, error)
	ListRoutesByQuery(params url.Values) ([]cfclient.Route, error)
	ListServiceInstancesByQuery(params url.Values) ([]cfclient.ServiceInstance, error)
	ListSpacesByQuery(query url.Values) ([]cfclient.Space, error)
	ListStacksByQuery(params url.Values) ([]cfclient.Stack, error)
	ListUserProvidedServiceInstancesByQuery(params url.Values) ([]cfclient.UserProvidedServiceInstance, error)

	NewRequest(method, path string) *cfclient.Request
	NewRequestWithBody(method, path string, body io.Reader) *cfclient.Request

	UpdateApp(guid string, aur cfclient.AppUpdateResource) (cfclient.UpdateResponse, error)
	UpdateV3App(guid string, req cfclient.UpdateV3AppRequest) (*cfclient.V3App, error)

	UploadAppBits(io.Reader, string) error
	UploadDropletBits(io.Reader, string) (string, error)
}

//counterfeiter:generate -o fakes . Client

type Client interface {
	APIClient
	DoWithRetry(f func() error) error
	Get(url string) ([]byte, error)
	GetClientConfig() *cfclient.Config
	HTTPClient() *http.Client
	Target() string
}

type client struct {
	CachingClient *CachingClient
	RetryPause    time.Duration
	RetryTimeout  time.Duration
	Config        *Config
	cfConfig      *cfclient.Config
	*cfclient.Client
}

func NewClient(cfg *Config, options ...func(*client)) (*client, error) {
	client := &client{
		Config:       cfg,
		RetryTimeout: DefaultRetryTimeout,
		RetryPause:   DefaultRetryPause,
	}

	for _, o := range options {
		o(client)
	}

	if client.Config == nil {
		return nil, fmt.Errorf("cf client must be configured")
	}

	return client, nil
}

func (c *client) AppByName(appName, spaceGuid, orgGuid string) (cfclient.App, error) {
	return c.lazyLoadCacheClientOrDie().AppByName(appName, spaceGuid, orgGuid)
}

func (c *client) BindRoute(routeGUID, appGUID string) error {
	return c.lazyLoadCacheClientOrDie().BindRoute(routeGUID, appGUID)
}

func (c *client) CreateApp(request cfclient.AppCreateRequest) (cfclient.App, error) {
	return c.lazyLoadCacheClientOrDie().CreateApp(request)
}

func (c *client) CreateOrg(request cfclient.OrgRequest) (cfclient.Org, error) {
	return c.lazyLoadCacheClientOrDie().CreateOrg(request)
}

func (c *client) CreateSpace(request cfclient.SpaceRequest) (cfclient.Space, error) {
	return c.lazyLoadCacheClientOrDie().CreateSpace(request)
}

func (c *client) CreateRoute(request cfclient.RouteRequest) (cfclient.Route, error) {
	return c.lazyLoadCacheClientOrDie().CreateRoute(request)
}

func (c *client) CreateServiceBinding(appGUID, serviceInstanceGUID string) (*cfclient.ServiceBinding, error) {
	return c.lazyLoadCacheClientOrDie().CreateServiceBinding(appGUID, serviceInstanceGUID)
}

func (c *client) DeleteApp(guid string) error {
	return c.lazyLoadCacheClientOrDie().DeleteApp(guid)
}

func (c *client) DeleteOrg(guid string, recursive, async bool) error {
	return c.lazyLoadCacheClientOrDie().DeleteOrg(guid, recursive, async)
}

func (c *client) DoRequest(req *cfclient.Request) (*http.Response, error) {
	return c.lazyLoadCacheClientOrDie().DoRequest(req)
}

func (c *client) Do(req *http.Request) (*http.Response, error) {
	return c.lazyLoadCacheClientOrDie().Do(req)
}

func (c *client) GetAppByGuidNoInlineCall(guid string) (cfclient.App, error) {
	return c.lazyLoadCacheClientOrDie().GetAppByGuidNoInlineCall(guid)
}

func (c *client) GetClientConfig() *cfclient.Config {
	return c.lazyLoadClientConfig(c.Config)
}

func (c *client) GetDomainByName(name string) (cfclient.Domain, error) {
	return c.lazyLoadCacheClientOrDie().GetDomainByName(name)
}

func (c *client) GetOrgByGuid(guid string) (cfclient.Org, error) {
	return c.lazyLoadCacheClientOrDie().GetOrgByGuid(guid)
}

func (c *client) GetOrgQuotaByName(name string) (cfclient.OrgQuota, error) {
	return c.lazyLoadCacheClientOrDie().GetOrgQuotaByName(name)
}

func (c *client) GetOrgByName(name string) (cfclient.Org, error) {
	return c.lazyLoadCacheClientOrDie().GetOrgByName(name)
}

func (c *client) GetSharedDomainByName(name string) (cfclient.SharedDomain, error) {
	return c.lazyLoadCacheClientOrDie().GetSharedDomainByName(name)
}

func (c *client) GetSpaceByGuid(guid string) (cfclient.Space, error) {
	return c.lazyLoadCacheClientOrDie().GetSpaceByGuid(guid)
}

func (c *client) GetSpaceByName(name string, orgGUID string) (cfclient.Space, error) {
	return c.lazyLoadCacheClientOrDie().GetSpaceByName(name, orgGUID)
}

func (c *client) GetStackByGuid(guid string) (cfclient.Stack, error) {
	return c.lazyLoadCacheClientOrDie().GetStackByGuid(guid)
}

func (c *client) ListAppsByQuery(params url.Values) ([]cfclient.App, error) {
	return c.lazyLoadCacheClientOrDie().ListAppsByQuery(params)
}

func (c *client) ListOrgsByQuery(params url.Values) ([]cfclient.Org, error) {
	return c.lazyLoadCacheClientOrDie().ListOrgsByQuery(params)
}

func (c *client) ListRoutesByQuery(params url.Values) ([]cfclient.Route, error) {
	return c.lazyLoadCacheClientOrDie().ListRoutesByQuery(params)
}

func (c *client) ListServiceInstancesByQuery(params url.Values) ([]cfclient.ServiceInstance, error) {
	return c.lazyLoadCacheClientOrDie().ListServiceInstancesByQuery(params)
}

func (c *client) ListSpacesByQuery(query url.Values) ([]cfclient.Space, error) {
	return c.lazyLoadCacheClientOrDie().ListSpacesByQuery(query)
}

func (c *client) ListStacksByQuery(params url.Values) ([]cfclient.Stack, error) {
	return c.lazyLoadCacheClientOrDie().ListStacksByQuery(params)
}

func (c *client) ListUserProvidedServiceInstancesByQuery(params url.Values) ([]cfclient.UserProvidedServiceInstance, error) {
	return c.lazyLoadCacheClientOrDie().ListUserProvidedServiceInstancesByQuery(params)
}

func (c *client) NewRequest(method, path string) *cfclient.Request {
	return c.lazyLoadCacheClientOrDie().NewRequest(method, path)
}

func (c *client) NewRequestWithBody(method, path string, body io.Reader) *cfclient.Request {
	return c.lazyLoadCacheClientOrDie().NewRequestWithBody(method, path, body)
}

func (c *client) UpdateApp(guid string, aur cfclient.AppUpdateResource) (cfclient.UpdateResponse, error) {
	return c.lazyLoadCacheClientOrDie().UpdateApp(guid, aur)
}

func (c *client) UpdateV3App(guid string, req cfclient.UpdateV3AppRequest) (*cfclient.V3App, error) {
	return c.lazyLoadCacheClientOrDie().UpdateV3App(guid, req)
}

func (c *client) UploadAppBits(reader io.Reader, s string) error {
	return c.lazyLoadCacheClientOrDie().UploadAppBits(reader, s)
}

func (c *client) UploadDropletBits(reader io.Reader, s string) (string, error) {
	return c.lazyLoadCacheClientOrDie().UploadDropletBits(reader, s)
}

func (c *client) lazyLoadCacheClientOrDie() *CachingClient {
	if c.CachingClient == nil {
		cf, err := c.lazyLoadCFClient(c.Config)
		if err != nil {
			panic(err)
		}
		c.CachingClient = NewCachingClient(cf)
	}
	return c.CachingClient
}

func (c *client) lazyLoadCFClient(cfg *Config) (*cfclient.Client, error) {
	if c.Client == nil {
		cf, err := cfclient.NewClient(c.lazyLoadClientConfig(cfg))
		if err != nil {
			return nil, err
		}
		if c.Config.hc == nil {
			c.Config.hc = cf.Config.HttpClient
		}
		c.Client = cf
	}

	return c.Client, nil
}

func (c *client) lazyLoadClientConfig(config *Config) *cfclient.Config {
	if c.cfConfig == nil {
		cfg := &cfclient.Config{
			ApiAddress:        config.Target,
			Username:          config.Username,
			Password:          config.Password,
			ClientID:          config.ClientID,
			ClientSecret:      config.ClientSecret,
			SkipSslValidation: config.SSLDisabled,
			HttpClient:        config.hc,
			Token:             config.AccessToken,
		}
		c.cfConfig = cfg
	}

	return c.cfConfig
}

func WithHTTPClient(c *http.Client) func(*client) {
	return func(cf *client) {
		cf.Config.hc = c
	}
}

func WithRetryTimeout(t time.Duration) func(*client) {
	return func(cf *client) {
		cf.RetryTimeout = t
	}
}

func WithRetryPause(t time.Duration) func(*client) {
	return func(cf *client) {
		cf.RetryPause = t
	}
}

func (c *client) HTTPClient() *http.Client {
	return c.Config.HTTPClient()
}

func (c *client) Target() string {
	return c.Config.Target
}

func (c *client) DoWithRetry(f func() error) error {
	ctx, cancel := context.WithTimeout(context.Background(), c.RetryTimeout)
	defer cancel()

	for {
		err := f()
		if err == nil {
			return nil
		}

		dnsErr := &net.DNSError{}
		ok := errors.As(err, &dnsErr)
		if ok || errors.Is(err, ErrRetry) {
			select {
			case <-time.After(c.RetryPause):
				continue
			case <-ctx.Done():
				return fmt.Errorf("timed out retrying operation, %s", err)
			}
		}

		return err
	}
}

func (c *client) Get(url string) ([]byte, error) {
	var body []byte
	err := c.DoWithRetry(func() error {
		req := c.NewRequest(http.MethodGet, url)
		httpResp, err := c.DoRequest(req)
		if err != nil {
			return err
		}

		defer func(body io.ReadCloser) {
			if body != nil {
				closeErr := body.Close()
				if closeErr != nil {
					err = closeErr
				}
			}
		}(httpResp.Body)
		if err == nil {
			if httpResp.StatusCode >= 500 && httpResp.StatusCode <= 599 {
				return ErrRetry
			}
		}

		body, err = ioutil.ReadAll(httpResp.Body)
		if err != nil {
			return err
		}

		return nil
	})

	return body, err
}
