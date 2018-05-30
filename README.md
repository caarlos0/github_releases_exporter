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

On the prometheus settings, add the domain_expoter prober:

```yaml
- job_name: releases
  scrape_interval: 5m
  metrics_path: /probe
  relabel_configs:
    - source_labels: [__address__]
      target_label: __param_target
    - source_labels: [__param_target]
      target_label: repository
    - target_label: __address__
      replacement: localhost:9333 # github_releases_exporter address
  static_configs:
    - targets:
      - goreleaser/goreleaser
```

It works more or less like prometheus's
[blackbox_exporter](https://github.com/prometheus/blackbox_exporter).

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
