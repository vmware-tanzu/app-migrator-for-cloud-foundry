# app-migrator-for-cloud-foundry

## Overview

The `app-migrator` is a command-line tool for migrating [Application Instances](https://docs.cloudfoundry.org/concepts/diego/diego-architecture.html) from one [Cloud Foundry](https://docs.cloudfoundry.org/) (CF) or [Tanzu Application Service](https://tanzu.vmware.com/application-service) (TAS) to another.

## Documentation

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

## Cloud Controller API to Migration Process Mapping

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

## Logs

By default, all log output is appended to `/tmp/app-migrator.log`. You can override this location by setting the
`APP_MIGRATOR_LOG_FILE` environment variable.

## For Developers

This project uses Go 1.16+ and Go modules. Clone the repo to any directory.

Build and run all checks

```shell
make all
```

Build the project

```shell
make build
```

Run the tests

```shell
make test
```

Run `make help` for all other tasks.

### Integration tests

To run the integration tests, you need to place an `app-migrator.yml` file under the [test/e2e](./test/e2e) directory.
To control the number of apps that are created as part of the test fixture, set `TEST_APP_COUNT`
(e.g. `export TEST_APP_COUNT=3`). The default is `3`.

To run all the integration tests

```shell
make test-integration
```

Or to run just `export org` tests

```shell
make test-export-org
```

See `Makefile` or run `make help` for all test targets.

## Contributing

The app-migrator-for-cloud-foundry project team welcomes contributions from the community. Before you start working with app-migrator-for-cloud-foundry, please
read our [Developer Certificate of Origin](https://cla.vmware.com/dco). All contributions to this repository must be
signed as described on that page. Your signature certifies that you wrote the patch or have the right to pass it on
as an open-source patch. For more detailed information, refer to [CONTRIBUTING.md](CONTRIBUTING.md).

## License

Refer to [LICENSE](LICENSE) for details.
