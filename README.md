[![Build Status](https://travis-ci.org/costela/nss-docker.svg?branch=master)](https://travis-ci.org/costela/nss-docker)
[![Go Report Card](https://goreportcard.com/badge/github.com/costela/nss-docker)](https://goreportcard.com/report/github.com/costela/nss-docker)

**NOTE**: This project is a proof-of-concept. For an easier, more portable solution, try [docker-etchosts](/costela/docker-etchosts).

# nss-docker

Simple backend plugin for `libc`'s [name service switch](http://www.gnu.org/software/libc/manual/html_node/Name-Service-Switch.html) to query a local docker for running containers.

This enables resolving containers' IPs locally based on container name or alias.

Additionally, `nss-docker` supports `docker-compose` by optionally organizing containers' domain names by project.

## Installation

In order to compile `nss-docker` you will need the C headers for `libc`. These can be installed in Debian variants
(including Ubuntu) with the `libc6-dev` package and in Redhat variants with the `glibc-headers` package.

- `go get -d github.com/costela/nss-docker`
- `cd $(go env GOPATH)/src/github.com/costela/nss-docker`
- `dep ensure -vendor-only` (you may need to install [dep](https://github.com/golang/dep))
- `sudo make install`
- add `docker` to the `hosts` line in `/etc/nsswitch.conf`. The entry should be placed before other network
backends like `dns` or `mdns`, to ensure faster resolution.

⚠ *Note*: `nss-docker` requires access to the docker daemon as the user performing the queries (commonly achieved by adding
the user in question to the `docker` group). This has security implications, since any user with access to the docker daemon can trivially bypass local permission restrictions.

## Configuration

The configuration is stored as JSON and searched in this order: `~/.nss_docker.json`, `/etc/nss_docker.json`

The following configuration keys are currently supported:

* `Suffix`: (default: `.docker`) the TLD which will be appended to all containers. Searches not under this TLD will
bypass this plugin.  
This is useful if you want to test something using OAuth locally, since you can simulate real TLDs (which, of course,
will be shadowed by the `nss-docker` domains)
* `IncludeComposeProject`: (default: `true`) whether to include the `docker-compose` project name in the search. When
true, services will be found with the form `SERVICE.PROJECT.SUFFIX`. Otherwise only `SERVICE.SUFFIX` will be searched,
which simplifies names, but increases collision risk.

## Multiple docker-compose projects

If your workflow involves multiple docker-compose projects using the same service names (e.g. a generic name like
"frontend"), a simple `SERVICE_NAME.SUFFIX` search will not be enough. In this case, a lookup will always return the
first found container.

To avoid this problem, `nss-lookup` includes the docker-compose project name in the domain, i.e.: `SERVICE_NAME.PROJECT_NAME.SUFFIX`.

An alternative to this approach would be to disable project-based search (`IncludeComposeProject: false` in the
settings) and use aliases to make those services unique which you wish to make accessible:

In `projectA/docker-compose.yml`:
```yaml
services:
  frontend:
    ...
    networks:
      default:
        aliases:
          - projectA
    ...
```

And in `projectB/docker-compose.yml`:
```yaml
services:
  frontend:
    ...
    networks:
      default:
        aliases:
          - projectB
    ...
```

## Limitations

### Linux-only

Since this project builds on libc's [name service switch](http://www.gnu.org/software/libc/manual/html_node/Name-Service-Switch.html), it is only compatible with Linux.

### Cross-container name resolution

The approach taken by this plugin will not enable containers to resolve accross `docker-compose` projects. Name resolution inside containers uses DNS and bypasses the host system's NSS.
To solve this particular need, a possible solution would be to use `docker-compose`'s shared network (introduced with configuration file version 3.5, `docker-compose` version ):

In project A:
```yaml
services:
  serviceA:
    networks:
      shared:

networks:
  shared:
    name: shared
```

In project B:
```yaml
services:
  serviceB:
    networks:
      shared:

networks:
  shared:
    name: shared
```
Note the `name: shared` parameters. This might seem redundant, but it ensures the network gets created with the same name, since `docker-compose` would otherwise prefix it with the project name.

This should enable you to resolve and access `serviceB` from `serviceA` and vice versa.

### Memory consumption

This version of the plugin is written in Go, which means an automatic significant increase in memory usage. A future version may be rewritten in pure C to avoid this.
