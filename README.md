# github_releases_exporter

Exports GitHub release metrics to the Prometheus format.

## Running

```console
./github_releases_exporter -b ":9333"
```

Or with docker:

```console
docker run -p 127.0.0.1:9333:9333 caarlos0/github_releases_exporter
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
    - 127.0.0.1:9333:9333
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
  - targets: ['releases:9333']
```

And you are done!

## Grafana Dashboard

I have a dashvboard like this:

![](https://grafana.com/api/dashboards/6328/images/4048/image)

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
