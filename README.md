![Antares Logo](./docs/antares-logo.png)

# Antares

[![standard-readme compliant](https://img.shields.io/badge/readme%20style-standard-brightgreen.svg)](https://github.com/RichardLitt/standard-readme)
[![readme antares](https://img.shields.io/badge/readme-Antares-blue)](README.md)
[![GitHub license](https://img.shields.io/github/license/dennis-tra/antares)](https://github.com/dennis-tra/antares/blob/main/LICENSE)

A gateway and pinning service probing tool. It allows you to track information about the peers that are powering those services. This includes but is not limited to PeerIDs, agent versions, supported protocols, and geo-locations.

## Table of Contents

- [Usage](#usage)
- [How does it work?](#how-does-it-work)
- [Install](#install)
  - [Release download](#release-download)
  - [From source](#from-source)
- [Development](#development)
  - [Database](#database)
- [Database](#database-1)
  - [Create a new migration](#create-a-new-migration)
- [Maintainers](#maintainers)
- [Contributing](#contributing)
- [Other Projects](#other-projects)
- [License](#license)

## Usage

Antares is a command line tool and just provides the `start` sub command. To simply start tracking run:

```shell
antares crawl --dry-run
```

Usually results are persisted in a postgres database - the `--dry-run` flag prevents it from doing that and prints them to the console.

See the command line help page below for configuration options:

```shell
NAME:
   antares - A tool that can detect peer information of gateways and pinning services.

USAGE:
   antares [global options] command [command options] [arguments...]

VERSION:
   vdev+5f3759df

AUTHOR:
   Dennis Trautwein <antares@dtrautwein.eu>

COMMANDS:
   start    Starts to provide content to the network and request it through gateways and pinning services.
   help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --config FILE        Load configuration from FILE [$ANTARES_CONFIG_FILE]
   --db-host value      On which host address can antares reach the database (default: 0.0.0.0) [$ANTARES_DATABASE_HOST]
   --db-name value      The name of the database to use (default: antares) [$ANTARES_DATABASE_NAME]
   --db-password value  The password for the database to use (default: password) [$ANTARES_DATABASE_PASSWORD]
   --db-port value      On which port can antares reach the database (default: 5432) [$ANTARES_DATABASE_PORT]
   --db-sslmode value   The sslmode to use when connecting the the database (default: disable) [$ANTARES_DATABASE_SSL_MODE]
   --db-user value      The user with which to access the database to use (default: antares) [$ANTARES_DATABASE_USER]
   --debug              Set this flag to enable debug logging (default: false) [$ANTARES_DEBUG]
   --dry-run            Don't persist anything to a database (you don't need a running DB) (default: false) [$ANTARES_DATABASE_DRY_RUN]
   --help, -h           show help (default: false)
   --host value         On which network interface should Antares listen on (default: 0.0.0.0) [$ANTARES_HOST]
   --log-level value    Set this flag to a value from 0 (least verbose) to 6 (most verbose). Overrides the --debug flag (default: 4) [$ANTARES_LOG_LEVEL]
   --port value         On which port should Antares listen on (default: 2002) [$ANTARES_Port]
   --pprof-port value   Port for the pprof profiling endpoint (default: 2003) [$ANTARES_PPROF_PORT]
   --prom-host value    Where should prometheus serve the metrics endpoint (default: 0.0.0.0) [$ANTARES_PROMETHEUS_HOST]
   --prom-port value    On which port should prometheus serve the metrics endpoint (default: 2004) [$ANTARES_PROMETHEUS_PORT]
   --version, -v        print the version (default: false)
```

## How does it work?

TODO

## Install

### Release download

There is no release yet.

### From source

To compile it yourself run:

```shell
go install github.com/dennis-tra/antares/cmd/antares@latest
```

Make sure the `$GOPATH/bin` is in your PATH variable to access the installed `antares` executable.

## Development

To develop this project you need Go `> 1.19` and the tools:

- [`golang-migrate/migrate`](https://github.com/golang-migrate/migrate) to manage the SQL migration `v4.15.2`
- [`volatiletech/sqlboiler`](https://github.com/volatiletech/sqlboiler) to generate Go ORM `v4.13.0`
- `docker` to run a local postgres instance

To install the necessary tools you can run `make tools`. This will use the `go install` command to download and install the tools into your `$GOPATH/bin` directory. So make sure you have it in your `$PATH` environment variable.

### Database

You need a running postgres instance to persist tracking results. Use the following command to start a local instance of postgres:

```shell
make database

# OR

docker run --rm -p 5432:5432 -e POSTGRES_PASSWORD=password -e POSTGRES_USER=antares -e POSTGRES_DB=antares postgres:14
```

> **Info:** You can use the `start` sub-command with the `--dry-run` option that skips all database operations.

The default database settings are:

```
Name     = "antares",
Password = "password",
User     = "antares",
Host     = "localhost",
Port     = 5432,
```

To apply migrations then run:

```shell
# Up migrations
migrate -database 'postgres://antares:password@localhost:5432/antares?sslmode=disable' -path migrations up
# OR
make migrate-up

# Down migrations
migrate -database 'postgres://antares:password@localhost:5432/antares?sslmode=disable' -path migrations down
# OR
make migrate-down

# Create new migration
migrate create -ext sql -dir migrations -seq some_migration_name
```

To generate the ORM with SQLBoiler run:

```shell
sqlboiler psql
```

## Database

### Create a new migration

```shell
make tools
migrate create -ext sql -dir migrations -seq my_migration_name
```

## Maintainers

[@dennis-tra](https://github.com/dennis-tra).

## Contributing

Feel free to dive in! [Open an issue](https://github.com/dennis-tra/nebula-crawler/issues/new) or submit PRs.

## Other Projects

You may be interested in one of my other projects:

- [`nebula`](https://github.com/dennis-tra/nebula-crawler) - A [libp2p](https://github.com/libp2p/go-libp2p) DHT crawler and monitoring tool.
- [`pcp`](https://github.com/dennis-tra/pcp) - Command line peer-to-peer data transfer tool based on [libp2p](https://github.com/libp2p/go-libp2p).
- [`image-stego`](https://github.com/dennis-tra/image-stego) - A novel way to image manipulation detection. Steganography-based image integrity - Merkle tree nodes embedded into image chunks so that each chunk's integrity can be verified on its own.

## License

[Apache License Version 2.0](LICENSE) © Dennis Trautwein
