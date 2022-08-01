# App Migrator for Cloud Foundry

[![build workflow](https://github.com/vmware-tanzu/app-migrator-for-cloud-foundry/actions/workflows/build.yml/badge.svg?branch=main)](https://github.com/vmware-tanzu/app-migrator-for-cloud-foundry/actions/workflows/build.yml)

## Overview

The `app-migrator` is a command-line tool for migrating [Application Instances](https://docs.cloudfoundry.org/concepts/diego/diego-architecture.html) from one [Cloud Foundry](https://docs.cloudfoundry.org/) (CF) or [Tanzu Application Service](https://tanzu.vmware.com/application-service) (TAS) to another.

### Cloud Controller API to Migration Process Mapping

- Applications                      - App-Migrator
- Application Environment Variables - App-Migrator
- Buildpacks                        - Other
- Default Security Groups           - Other
- Feature Flags                     - Other
- Private Domains                   - CF-Mgmt
- Shared Domains                    - CF-Mgmt
- Routes                            - App-Migrator
- Route Mappings                    - App-Migrator
- Quota Definitions                 - CF-Mgmt
- Application Security Groups       - CF-Mgmt
- Services                          - Other
- Service Brokers                   - Other
- Service Plans                     - CF-Mgmt
- Service Plan Visibility           - CF-Mgmt
- Service Keys                      - Other
- Managed Service Instances         - Service-Instance-Migrator
- User Provided Services            - Service-Instance-Migrator
- Service Bindings                  - App-Migrator
- Orgs                              - CF-Mgmt
- Spaces                            - CF-Mgmt
- Space Quotas                      - CF-Mgmt
- Isolation Segments                - CF-Mgmt
- Stacks                            - Other
- Local UAA Users/Clients           - Other
- LDAP Users                        - CF-Mgmt
- Roles                             - CF-Mgmt

## Getting Started

### Download latest release

Download the `app-migrator-<OS>-amd64.tar.gz` from the most recent release listed on the [App Migrator for Cloud Foundry releases](https://github.com/vmware-tanzu/app-migrator-for-cloud-foundry/releases) page.

Following are the instructions for installing version `v0.0.8`.

#### For macOS

```shell
VERSION=v0.0.8
wget -q https://github.com/vmware-tanzu/app-migrator-for-cloud-foundry/releases/download/${VERSION}/app-migrator-darwin-amd64.tgz
tar -xvf app-migrator-darwin-amd64.tgz -C /usr/local/bin
chmod +x /usr/local/bin/app-migrator
```

#### For linux

```shell
VERSION=v0.0.8
wget -q https://github.com/vmware-tanzu/app-migrator-for-cloud-foundry/releases/download/${VERSION}/app-migrator-linux-amd64.tgz
tar -xvf app-migrator-darwin-amd64.tgz -C /usr/local/bin
chmod +x /usr/local/bin/app-migrator
```

### Build from source

See the [development guide](./DEVELOPMENT.md) for instructions to build from source.

## Documentation

The `app-instance-migrator` requires user credentials or client credentials to communicate with the Cloud Foundry Cloud Controller API.

The configuration for the CLI is specified in a file called `app-migrator.yml` which can be overridden with the following environment variables.

- `APP_MIGRATOR_CONFIG_FILE` will override cli config file location [default: `./app-migrator.yml`]
- `APP_MIGRATOR_CONFIG_HOME` will override cli config directory location [default: `.`, `$HOME`, or `$HOME/.config/app-migrator`]

Create a `app-migrator.yml` using the following template.

```yaml
export_dir: /tmp/export-apps
exclude_orgs:
  - system
source_api:
  url: https://api.src.tas.example.com
  # admin or client credentials (not both)
  username: ""
  password: ""
  client_id: client-with-cloudcontroller-admin-permissions
  client_secret: client-secret
target_api:
  url: https://api.dst.tas.example.com
  # admin or client credentials (not both)
  username: ""
  password: ""
  client_id: client-with-cloudcontroller-admin-permissions
  client_secret: client-secret
```

### Commands

- **export** - Export all applications (from every org and space) from a foundation.
- **export org** - Export only the applications hosted within an organization.
- **export space** - Export only the applications hosted within a space.
- **export app** - Export only a single application.
- **export-incremental** - Export only the applications that have changed (from all orgs and spaces) since a previous export.
- **import** - Import all applications from an export.
- **import org** - Import only the applications hosted within an organization from an export.
- **import space** - Import only the applications hosted within a space from an export.
- **import app** - Import only a single application from an export.
- **import-incremental** - Import only the applications that have changed (from all orgs and spaces) since a previous import.

Check out the [docs](./docs/app-migrator.md) to see usage for all the commands.

## Logs

By default, all log output is appended to `/tmp/app-migrator.log`. You can override this location by setting the
`APP_MIGRATOR_LOG_FILE` environment variable.

## Contributing

The App Migrator for Cloud Foundry project team welcomes contributions from the community. Before you start working with App Migrator for Cloud Foundry, please
read our [Developer Certificate of Origin](https://cla.vmware.com/dco). All contributions to this repository must be
signed as described on that page. Your signature certifies that you wrote the patch or have the right to pass it on
as an open-source patch. For more detailed information, refer to [CONTRIBUTING.md](CONTRIBUTING.md).

## License

Refer to [LICENSE](LICENSE) for details.
