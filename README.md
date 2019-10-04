# github_releases_exporter

Exports GitHub release metrics to the Prometheus format.

[![Release](https://img.shields.io/github/release/caarlos0/github_releases_exporter.svg?style=flat-square)](https://github.com/caarlos0/github_releases_exporter/releases/latest)
[![Software License](https://img.shields.io/badge/license-MIT-brightgreen.svg?style=flat-square)](LICENSE.md)
[![Build Status](https://travis-ci.com/caarlos0/github_releases_exporter.svg?branch=master)](https://travis-ci.com/caarlos0/github_releases_exporter)
[![Coverage Status](https://img.shields.io/codecov/c/github/caarlos0/github_releases_exporter/master.svg?style=flat-square)](https://codecov.io/gh/caarlos0/github_releases_exporter)
[![Go Doc](https://img.shields.io/badge/godoc-reference-blue.svg?style=flat-square)](http://godoc.org/github.com/caarlos0/github_releases_exporter)
[![Go Report Card](https://goreportcard.com/badge/github.com/caarlos0/github_releases_exporter?style=flat-square)](https://goreportcard.com/report/github.com/caarlos0/github_releases_exporter)
[![SayThanks.io](https://img.shields.io/badge/SayThanks.io-%E2%98%BC-1EAEDB.svg?style=flat-square)](https://saythanks.io/to/caarlos0)
[![Powered By: GoReleaser](https://img.shields.io/badge/powered%20by-goreleaser-green.svg?style=flat-square)](https://github.com/goreleaser)

## Running

```console
./github_releases_exporter
```

Or with docker:

```console
docker run -p 127.0.0.1:9222:9222 caarlos0/github_releases_exporter
```

## Configuration

You can set it up on docker compose like:

```yaml
version: '3'
services:
  releases:
    image: caarlos0/github_releases_exporter:v1
    restart: always
    volumes:
    - /path/to/releases.yml:/etc/releases.yml:ro
    command:
    - '--config.file=/etc/releases.yml'
    ports:
    - 127.0.0.1:9222:9222
    env_file:
    - .env
```

The `releases.yml` file should look like this:

```yaml
repositories:
- goreleaser/goreleaser
- caarlos0/github_releases_exporter
```

On the prometheus settings, add the releases job like this:

```yaml
- job_name: 'releases'
  static_configs:
  - targets: ['releases:9222']
```

And you are done!

## Configuration Reload

You can reload the configuration at any time by sending a `SIGHUP` to the
process.

## Grafana Dashboard

I have a dashvboard like this:

![](https://grafana.com/api/dashboards/6328/images/4078/image)

You can get it from [the grafana website](https://grafana.com/dashboards/6328)
or just import the dashboard `6328`.

## Building locally

Install the needed tooling and libs:

```console
make setup
```

Run with:

```console
go run main.go
```

Run tests with:

```console
make test
```
