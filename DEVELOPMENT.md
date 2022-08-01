# Development

This directory contains documentation for Developers in order to start
[contributing](CONTRIBUTING.md) to the Service Instance Migrator for Cloud Foundry.

This project uses Go 1.17+ and Go modules. Clone the repo to any directory.

## Build and run all checks

Before submitting any PRs to upstream, make sure to run:

```shell
make all
```

## Build the project

To build into the local ./bin dir:

```shell
make build
```

## Run the unit tests

```shell
make test
```

## Run integration tests

To run the integration tests, you need to place an `app-migrator.yml` file under the [test/e2e](./test/e2e) directory.
To control the number of apps that are created as part of the test fixture, set `TEST_APP_COUNT`
(e.g. `export TEST_APP_COUNT=3`). The default is `3`.

To run all the integration tests

```shell
make test-e2e
```

Or to run just `export org` tests

```shell
make test-export-org
```

See `Makefile` or run `make help` for all test targets.
