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
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
	"time"

	appcontext "github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/context"

	"github.com/vmware-tanzu/app-migrator-for-cloud-foundry/pkg/cf"

	"github.com/cloudfoundry-community/go-cfclient"
)

type PackageDownloader interface {
	downloadPackages(c *appcontext.Context, packageGUID string) ([]byte, error)
}

type PackageRetriever interface {
	getPackages(c *appcontext.Context, appGUID string) (string, error)
}

type DefaultDropletExporter struct {
	PackageRetriever
	PackageDownloader
}

type DefaultPackageDownloader struct {
}

type DefaultPackageRetriever struct {
}

func NewDropletExporter() *DefaultDropletExporter {
	return &DefaultDropletExporter{
		PackageRetriever:  &DefaultPackageRetriever{},
		PackageDownloader: &DefaultPackageDownloader{},
	}
}

func (d *DefaultDropletExporter) NumberOfPackages(ctx *appcontext.Context, app cfclient.App) (float64, error) {
	var resp struct {
		Pagination pagination `json:"pagination"`
	}

	body, err := ctx.ExportCFClient.Get(fmt.Sprintf("/v3/apps/%s/packages", app.Guid))
	if err != nil {
		return float64(0), err
	}

	if err = json.Unmarshal(body, &resp); err != nil {
		return float64(0), err
	}

	return resp.Pagination.TotalResults, nil
}

func (d *DefaultDropletExporter) DownloadDroplet(ctx *appcontext.Context, org cfclient.Org, space cfclient.Space, app cfclient.App, exportDir string) error {
	ctx.Logger.Infof("Downloading %s/%s/%s droplet", org.Name, space.Name, app.Name)

	body, err := ctx.ExportCFClient.Get(path.Join("/v2", "apps", app.Guid, "droplet", "download"))
	if err != nil {
		return err
	}

	var dropletFile *os.File
	dropletFile, err = os.Create(path.Join(exportDir, getAppFileName(app.Name)+".tgz"))
	if err != nil {
		return err
	}
	defer dropletFile.Close()

	_, err = io.Copy(dropletFile, bytes.NewReader(body))

	return err
}

func (d *DefaultDropletExporter) DownloadPackages(c *appcontext.Context, org cfclient.Org, space cfclient.Space, app cfclient.App, exportDir string) error {
	c.Logger.Infof("Downloading %s/%s/%s bits\n", org.Name, space.Name, app.Name)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	for {
		packageGUID, err := d.getPackages(c, app.Guid)
		if err != nil {
			cfErr := cfclient.CloudFoundryError{}

			if errors.As(err, &cfErr) {
				if cfErr.Code == 10001 { // unknown error, we're going to try again until we're successful or the timeout
					select {
					case <-time.After(10 * time.Second):
						continue
					case <-ctx.Done():
						return err
					}
				}
			}

			return err
		}

		var body []byte
		body, err = d.downloadPackages(c, packageGUID)
		if err != nil {
			return err
		}

		zipFileName := fmt.Sprintf("%s/%s.zip", exportDir, getAppFileName(app.Name))
		var zipFile io.WriteCloser
		zipFile, err = os.Create(zipFileName)
		if err != nil {
			return fmt.Errorf("error creating zip file: %w", err)
		}
		defer zipFile.Close()

		if written, err := io.Copy(zipFile, bytes.NewReader(body)); err != nil {
			fmt.Printf("Wrote %d bytes", written)
			return fmt.Errorf("error writing zip file: %w", err)
		}

		break
	}

	return nil
}

func (d *DefaultPackageRetriever) getPackages(c *appcontext.Context, appGUID string) (string, error) {
	var packageGUID string
	err := c.ExportCFClient.DoWithRetry(func() error {
		params := url.Values{
			"states":   []string{"READY"},
			"types":    []string{"bits"},
			"order_by": []string{"-updated_at"},
		}

		req := c.ExportCFClient.NewRequest(http.MethodGet, fmt.Sprintf("/v3/apps/%s/packages?%s", appGUID, params.Encode()))
		httpResp, err := c.ExportCFClient.DoRequest(req)
		if err != nil {
			return err
		}
		if httpResp.StatusCode >= 500 && httpResp.StatusCode <= 599 {
			defer httpResp.Body.Close()
			return cf.ErrRetry
		}

		var resp struct {
			Resources []cfclient.V3Package `json:"resources"`
		}

		if err = json.NewDecoder(httpResp.Body).Decode(&resp); err != nil {
			return err
		}

		if len(resp.Resources) == 0 {
			return fmt.Errorf("expected at least 1 package, but found %d", len(resp.Resources))
		}

		packageGUID = resp.Resources[0].GUID
		return nil
	})

	return packageGUID, err
}

func (d *DefaultPackageDownloader) downloadPackages(c *appcontext.Context, packageGUID string) ([]byte, error) {
	var httpResp *http.Response
	var body []byte
	var err error

	err = c.ExportCFClient.DoWithRetry(func() error {
		var err error
		req := c.ExportCFClient.NewRequest(http.MethodGet, fmt.Sprintf("/v3/packages/%s/download", packageGUID))
		httpResp, err = c.ExportCFClient.DoRequest(req)
		if err == nil {
			defer httpResp.Body.Close()
			if httpResp.StatusCode >= 500 && httpResp.StatusCode <= 599 {
				return cf.ErrRetry
			}
			body, err = ioutil.ReadAll(httpResp.Body)
			if err != nil {
				return err
			}
		}
		return err
	})
	if err != nil {
		return nil, err
	}

	if httpResp.StatusCode == http.StatusFound {
		locationURL := httpResp.Header.Get("location")
		err = c.ExportCFClient.DoWithRetry(func() error {
			client := c.ExportCFClient.HTTPClient()
			resp, err := client.Get(locationURL)
			if err == nil {
				defer resp.Body.Close()
				if resp.StatusCode >= 500 && resp.StatusCode <= 599 {
					return cf.ErrRetry
				}
				body, err = ioutil.ReadAll(resp.Body)
				if err != nil {
					return err
				}
			}
			return err
		})

		if err != nil {
			return body, err
		}
	}

	return body, err
}
