![Antares Logo](./docs/antares-logo.png)

# Antares

[![standard-readme compliant](https://img.shields.io/badge/readme%20style-standard-brightgreen.svg)](https://github.com/RichardLitt/standard-readme)
[![readme antares](https://img.shields.io/badge/readme-Antares-blue)](README.md)
[![GitHub license](https://img.shields.io/github/license/dennis-tra/antares)](https://github.com/dennis-tra/antares/blob/main/LICENSE)

A gateway and pinning service probing tool. It allows you to track information about the peers that are powering those services. This includes but is not limited to PeerIDs, agent versions, supported protocols, and geo-locations.

## Table of Contents

- [Usage](#usage)
- [How does it work?](#how-does-it-work)
- [Installation](#installation)
- [Development](#development)
  - [Database](#database)
- [Database](#database-1)
  - [Create a new migration](#create-a-new-migration)
- [Targets](#targets)
  - [Gateways](#gateways)
  - [Pinning Services](#pinning-services)
    - [Pinata](#pinata) | [Infura](#infura)
- [Maintainers](#maintainers)
- [Contributing](#contributing)
- [Other Projects](#other-projects)
- [License](#license)

## Usage

Antares is a command line tool and just provides the `start` sub command. To simply start tracking run:

```shell
antares start --dry-run
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

## Installation

For now, you need to compile it yourself:

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

## Targets

### Gateways

When Antares starts it'll create a configuration file at `$XDG_CONFIG_HOME/antares/config.json`. There you can find a `Gateways` field that's an array of gateway names and url format strings. I used the public gateway tracker and extracted the gateways with the following command:

```javascript
JSON.stringify([...document.querySelectorAll("#checker\\.results .Link")].map(a => ({Name: a.textContent, URL: `https://${a.textContent}/ipfs/{cid}`})))
```


<details>
  <summary>This yields the following list (formatted):</summary>

```json
[
  {
    "Name": "Hostname",
    "URL": "https://Hostname/ipfs/{cid}"
  },
  {
    "Name": "ipfs.io",
    "URL": "https://ipfs.io/ipfs/{cid}"
  },
  {
    "Name": "dweb.link",
    "URL": "https://dweb.link/ipfs/{cid}"
  },
  {
    "Name": "gateway.ipfs.io",
    "URL": "https://gateway.ipfs.io/ipfs/{cid}"
  },
  {
    "Name": "ninetailed.ninja",
    "URL": "https://ninetailed.ninja/ipfs/{cid}"
  },
  {
    "Name": "via0.com",
    "URL": "https://via0.com/ipfs/{cid}"
  },
  {
    "Name": "ipfs.eternum.io",
    "URL": "https://ipfs.eternum.io/ipfs/{cid}"
  },
  {
    "Name": "hardbin.com",
    "URL": "https://hardbin.com/ipfs/{cid}"
  },
  {
    "Name": "cloudflare-ipfs.com",
    "URL": "https://cloudflare-ipfs.com/ipfs/{cid}"
  },
  {
    "Name": "astyanax.io",
    "URL": "https://astyanax.io/ipfs/{cid}"
  },
  {
    "Name": "cf-ipfs.com",
    "URL": "https://cf-ipfs.com/ipfs/{cid}"
  },
  {
    "Name": "ipns.co",
    "URL": "https://ipns.co/ipfs/{cid}"
  },
  {
    "Name": "ipfs.mrh.io",
    "URL": "https://ipfs.mrh.io/ipfs/{cid}"
  },
  {
    "Name": "gateway.originprotocol.com",
    "URL": "https://gateway.originprotocol.com/ipfs/{cid}"
  },
  {
    "Name": "gateway.pinata.cloud",
    "URL": "https://gateway.pinata.cloud/ipfs/{cid}"
  },
  {
    "Name": "ipfs.sloppyta.co",
    "URL": "https://ipfs.sloppyta.co/ipfs/{cid}"
  },
  {
    "Name": "ipfs.busy.org",
    "URL": "https://ipfs.busy.org/ipfs/{cid}"
  },
  {
    "Name": "ipfs.greyh.at",
    "URL": "https://ipfs.greyh.at/ipfs/{cid}"
  },
  {
    "Name": "gateway.serph.network",
    "URL": "https://gateway.serph.network/ipfs/{cid}"
  },
  {
    "Name": "jorropo.net",
    "URL": "https://jorropo.net/ipfs/{cid}"
  },
  {
    "Name": "ipfs.fooock.com",
    "URL": "https://ipfs.fooock.com/ipfs/{cid}"
  },
  {
    "Name": "cdn.cwinfo.net",
    "URL": "https://cdn.cwinfo.net/ipfs/{cid}"
  },
  {
    "Name": "aragon.ventures",
    "URL": "https://aragon.ventures/ipfs/{cid}"
  },
  {
    "Name": "permaweb.io",
    "URL": "https://permaweb.io/ipfs/{cid}"
  },
  {
    "Name": "ipfs.best-practice.se",
    "URL": "https://ipfs.best-practice.se/ipfs/{cid}"
  },
  {
    "Name": "storjipfs-gateway.com",
    "URL": "https://storjipfs-gateway.com/ipfs/{cid}"
  },
  {
    "Name": "ipfs.runfission.com",
    "URL": "https://ipfs.runfission.com/ipfs/{cid}"
  },
  {
    "Name": "ipfs.trusti.id",
    "URL": "https://ipfs.trusti.id/ipfs/{cid}"
  },
  {
    "Name": "ipfs.overpi.com",
    "URL": "https://ipfs.overpi.com/ipfs/{cid}"
  },
  {
    "Name": "ipfs.ink",
    "URL": "https://ipfs.ink/ipfs/{cid}"
  },
  {
    "Name": "ipfsgateway.makersplace.com",
    "URL": "https://ipfsgateway.makersplace.com/ipfs/{cid}"
  },
  {
    "Name": "ipfs.funnychain.co",
    "URL": "https://ipfs.funnychain.co/ipfs/{cid}"
  },
  {
    "Name": "ipfs.telos.miami",
    "URL": "https://ipfs.telos.miami/ipfs/{cid}"
  },
  {
    "Name": "ipfs.mttk.net",
    "URL": "https://ipfs.mttk.net/ipfs/{cid}"
  },
  {
    "Name": "ipfs.fleek.co",
    "URL": "https://ipfs.fleek.co/ipfs/{cid}"
  },
  {
    "Name": "ipfs.jbb.one",
    "URL": "https://ipfs.jbb.one/ipfs/{cid}"
  },
  {
    "Name": "ipfs.yt",
    "URL": "https://ipfs.yt/ipfs/{cid}"
  },
  {
    "Name": "hashnews.k1ic.com",
    "URL": "https://hashnews.k1ic.com/ipfs/{cid}"
  },
  {
    "Name": "ipfs.drink.cafe",
    "URL": "https://ipfs.drink.cafe/ipfs/{cid}"
  },
  {
    "Name": "ipfs.kavin.rocks",
    "URL": "https://ipfs.kavin.rocks/ipfs/{cid}"
  },
  {
    "Name": "ipfs.denarius.io",
    "URL": "https://ipfs.denarius.io/ipfs/{cid}"
  },
  {
    "Name": "crustwebsites.net",
    "URL": "https://crustwebsites.net/ipfs/{cid}"
  },
  {
    "Name": "ipfs0.sjc.cloudsigma.com",
    "URL": "https://ipfs0.sjc.cloudsigma.com/ipfs/{cid}"
  },
  {
    "Name": "ipfs.genenetwork.org",
    "URL": "https://ipfs.genenetwork.org/ipfs/{cid}"
  },
  {
    "Name": "ipfs.eth.aragon.network",
    "URL": "https://ipfs.eth.aragon.network/ipfs/{cid}"
  },
  {
    "Name": "ipfs.smartholdem.io",
    "URL": "https://ipfs.smartholdem.io/ipfs/{cid}"
  },
  {
    "Name": "ipfs.xoqq.ch",
    "URL": "https://ipfs.xoqq.ch/ipfs/{cid}"
  },
  {
    "Name": "natoboram.mynetgear.com",
    "URL": "https://natoboram.mynetgear.com/ipfs/{cid}"
  },
  {
    "Name": "video.oneloveipfs.com",
    "URL": "https://video.oneloveipfs.com/ipfs/{cid}"
  },
  {
    "Name": "ipfs.anonymize.com",
    "URL": "https://ipfs.anonymize.com/ipfs/{cid}"
  },
  {
    "Name": "ipfs.scalaproject.io",
    "URL": "https://ipfs.scalaproject.io/ipfs/{cid}"
  },
  {
    "Name": "search.ipfsgate.com",
    "URL": "https://search.ipfsgate.com/ipfs/{cid}"
  },
  {
    "Name": "ipfs.decoo.io",
    "URL": "https://ipfs.decoo.io/ipfs/{cid}"
  },
  {
    "Name": "alexdav.id",
    "URL": "https://alexdav.id/ipfs/{cid}"
  },
  {
    "Name": "ipfs.uploads.nu",
    "URL": "https://ipfs.uploads.nu/ipfs/{cid}"
  },
  {
    "Name": "hub.textile.io",
    "URL": "https://hub.textile.io/ipfs/{cid}"
  },
  {
    "Name": "ipfs1.pixura.io",
    "URL": "https://ipfs1.pixura.io/ipfs/{cid}"
  },
  {
    "Name": "ravencoinipfs-gateway.com",
    "URL": "https://ravencoinipfs-gateway.com/ipfs/{cid}"
  },
  {
    "Name": "konubinix.eu",
    "URL": "https://konubinix.eu/ipfs/{cid}"
  },
  {
    "Name": "ipfs.tubby.cloud",
    "URL": "https://ipfs.tubby.cloud/ipfs/{cid}"
  },
  {
    "Name": "ipfs.lain.la",
    "URL": "https://ipfs.lain.la/ipfs/{cid}"
  },
  {
    "Name": "ipfs.kaleido.art",
    "URL": "https://ipfs.kaleido.art/ipfs/{cid}"
  },
  {
    "Name": "ipfs.slang.cx",
    "URL": "https://ipfs.slang.cx/ipfs/{cid}"
  },
  {
    "Name": "ipfs.arching-kaos.com",
    "URL": "https://ipfs.arching-kaos.com/ipfs/{cid}"
  },
  {
    "Name": "storry.tv",
    "URL": "https://storry.tv/ipfs/{cid}"
  },
  {
    "Name": "ipfs.1-2.dev",
    "URL": "https://ipfs.1-2.dev/ipfs/{cid}"
  },
  {
    "Name": "dweb.eu.org",
    "URL": "https://dweb.eu.org/ipfs/{cid}"
  },
  {
    "Name": "permaweb.eu.org",
    "URL": "https://permaweb.eu.org/ipfs/{cid}"
  },
  {
    "Name": "ipfs.namebase.io",
    "URL": "https://ipfs.namebase.io/ipfs/{cid}"
  },
  {
    "Name": "ipfs.tribecap.co",
    "URL": "https://ipfs.tribecap.co/ipfs/{cid}"
  },
  {
    "Name": "ipfs.kinematiks.com",
    "URL": "https://ipfs.kinematiks.com/ipfs/{cid}"
  },
  {
    "Name": "c4rex.co",
    "URL": "https://c4rex.co/ipfs/{cid}"
  },
  {
    "Name": "nftstorage.link",
    "URL": "https://nftstorage.link/ipfs/{cid}"
  },
  {
    "Name": "gravity.jup.io",
    "URL": "https://gravity.jup.io/ipfs/{cid}"
  },
  {
    "Name": "fzdqwfb5ml56oadins5jpuhe6ki6bk33umri35p5kt2tue4fpws5efid.onion",
    "URL": "https://fzdqwfb5ml56oadins5jpuhe6ki6bk33umri35p5kt2tue4fpws5efid.onion/ipfs/{cid}"
  },
  {
    "Name": "tth-ipfs.com",
    "URL": "https://tth-ipfs.com/ipfs/{cid}"
  },
  {
    "Name": "ipfs.chisdealhd.co.uk",
    "URL": "https://ipfs.chisdealhd.co.uk/ipfs/{cid}"
  },
  {
    "Name": "ipfs.alloyxuast.tk",
    "URL": "https://ipfs.alloyxuast.tk/ipfs/{cid}"
  },
  {
    "Name": "ipfs.litnet.work",
    "URL": "https://ipfs.litnet.work/ipfs/{cid}"
  },
  {
    "Name": "4everland.io",
    "URL": "https://4everland.io/ipfs/{cid}"
  },
  {
    "Name": "ipfs-gateway.cloud",
    "URL": "https://ipfs-gateway.cloud/ipfs/{cid}"
  },
  {
    "Name": "w3s.link",
    "URL": "https://w3s.link/ipfs/{cid}"
  },
  {
    "Name": "cthd.icu",
    "URL": "https://cthd.icu/ipfs/{cid}"
  }
]
```
</details>

You can copy this list of to your own configuration file.

### Pinning Services

As with the gateway configuration, you can find a `PinningServices` field in the configuration file that's an array of pinning service names and authorization information. Each pinning service may have its own authorization format. This is explained in detail next.

#### Pinata

For pinata, grab a JWT that has the permission to `pin` and `unpin` CIDs. Then provide the following configuration:

```json
{
  ...
  "PinningServices": [
    {
      "Target": "pinata",
      "Authorization": "eyJhb..."
    }
  ],
  ...
}
```

You can monitor multiple pinata accounts by adding more `pinata` targets to that list with different JWTs.

#### Infura


For Infura, grab the project ID and API-Key secret from their website. Then concatenate them with the `,` in between:

```json
{
  ...
  "PinningServices": [
    {
      "Target": "infura",
      "Authorization": "PROJECT_ID,API_KEY_SECRET"
    }
  ],
  ...
}
```

## Maintainers

[@dennis-tra](https://github.com/dennis-tra).

## Contributing

Feel free to dive in! [Open an issue](https://github.com/dennis-tra/antares/issues/new) or submit PRs.

## Other Projects

You may be interested in one of my other projects:

- [`nebula`](https://github.com/dennis-tra/nebula-crawler) - A [libp2p](https://github.com/libp2p/go-libp2p) DHT crawler and monitoring tool.
- [`pcp`](https://github.com/dennis-tra/pcp) - Command line peer-to-peer data transfer tool based on [libp2p](https://github.com/libp2p/go-libp2p).
- [`image-stego`](https://github.com/dennis-tra/image-stego) - A novel way to image manipulation detection. Steganography-based image integrity - Merkle tree nodes embedded into image chunks so that each chunk's integrity can be verified on its own.

## License

[Apache License Version 2.0](LICENSE) © Dennis Trautwein
