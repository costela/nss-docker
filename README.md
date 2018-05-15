# nss-docker

Simple backend plugin for `libc`'s [name service switch](http://www.gnu.org/software/libc/manual/html_node/Name-Service-Switch.html) to query a local docker for running containers.

This enables resolving containers' IPs locally based on container name.

Additionally, `nss-docker` supports `docker-compose` by optionally organizing containers' domain names by project.

## Installation

In order to compile `nss-docker` you will need the C headers for `libc`. These can be installed in Debian variants
(including Ubuntu) with the `libc6-dev` package and in Redhat variants with the `glibc-headers` package.

After that, simply:
```
make install
```

And lastly, add `docker` to the `hosts` line in `/etc/nsswitch.conf`. The entry should be placed before other network
backends like `dns` or `mdns`, to ensure faster resolution.

Also: `nss-docker` requires access to the docker daemon as the user performing the queries (commonly achieved by adding
the user in question to the `docker` group)

## Configuration

The configuration is stored as JSON and searched in this order: `~/.nss_docker.json`, `/etc/nss_docker.json`

The following configuration keys are currently supported:

* `Suffix`: (default: `.docker`) the TLD which will be appended to all containers. Searches not under this TLD will
bypass this plugin.
* `IncludeComposeProject`: (default: `true`) whether to include the `docker-compose` project name in the search. When
true, services will be found with the form `SERVICE.PROJECT.SUFFIX`.

## Multiple docker-compose projects

If your workflow involves multiple docker-compose projects using the same service names (e.g. a generic name like
"frontend"), a simple `SERVICE_NAME.SUFFIX` search will not be enough. In this case, a lookup will always return the
first found container.

To avoid this problem, `nss-lookup` includes the docker-compose project name in the domain, i.e.: `SERVICE_NAME.PROJECT_NAME.SUFFIX`.

An alternative to this approach would be to disable project-based search (`IncludeComposeProject: false` in the
settings) and use aliases to make those services unique which you wish to make accessible:

In `projectA/docker-compose.yml`:
```
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
```
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

### Cross-container name resolution

The approach taken by this plugin will not enable containers to resolve accross `docker-compose` projects. Name resolution inside containers uses DNS and bypasses the host system's NSS.
To solve this particular need, a possible solution would be to use `docker-compose`'s shared network (introduced with configuration file version 3.5, `docker-compose` version ):

In project A:
```
services:
  serviceA:
    networks:
      shared:

networks:
  shared:
    name: shared
```

In project B:
```
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

This version of the plugin is written in Go, which means an automatic significant increase in memory usage. A future version will probably be rewritten in pure C to avoid this.