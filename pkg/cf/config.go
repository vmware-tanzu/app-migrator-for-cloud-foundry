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
	"crypto/tls"
	"fmt"
	"net/http"
	"sync"
	"time"

	"golang.org/x/oauth2"
)

type Config struct {
	AccessToken           string
	RefreshToken          string
	Target                string
	AuthorizationEndpoint string
	OrganizationFields    struct {
		Name string
	}
	SpaceFields struct {
		Name string
	}
	SSLDisabled  bool
	Username     string
	Password     string
	ClientID     string
	ClientSecret string

	mutex sync.RWMutex
	hc    *http.Client
}

func (c *Config) HTTPClient() *http.Client {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.hc != nil {
		return c.hc
	}

	oauthConfig := &oauth2.Config{
		ClientID:     "cf",
		ClientSecret: "",
		Endpoint: oauth2.Endpoint{
			AuthURL:   fmt.Sprintf("%s/oauth/authorize", c.AuthorizationEndpoint),
			TokenURL:  fmt.Sprintf("%s/oauth/token", c.AuthorizationEndpoint),
			AuthStyle: oauth2.AuthStyleInHeader,
		},
	}

	token := &oauth2.Token{
		AccessToken:  c.AccessToken,
		RefreshToken: c.RefreshToken,
		Expiry:       time.Now().Add(10 * time.Minute),
	}

	parentHC := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: c.SSLDisabled,
			},
		},
	}

	ctx := context.WithValue(context.TODO(), oauth2.HTTPClient, parentHC)
	c.hc = oauthConfig.Client(ctx, token)
	return c.hc
}
