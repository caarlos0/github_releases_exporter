package main

import (
	"context"
	"fmt"
	"net/http"

	"github.com/caarlos0/github_releases_exporter/client"

	"github.com/alecthomas/kingpin"
	"github.com/caarlos0/github_releases_exporter/collector"
	"github.com/caarlos0/github_releases_exporter/config"
	"github.com/patrickmn/go-cache"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/log"
)

// nolint: gochecknoglobals
var (
	bind        = kingpin.Flag("bind", "addr to bind the server").Short('b').Default(":9222").String()
	debug       = kingpin.Flag("debug", "show debug logs").Default("false").Bool()
	token       = kingpin.Flag("github.token", "github token").Envar("GITHUB_TOKEN").String()
	configFile  = kingpin.Flag("config.file", "config file").Default("releases.yml").ExistingFile()
	interval    = kingpin.Flag("refresh.interval", "time between refreshes with github api").Default("15m").Duration()
	maxReleases = kingpin.Flag("releases.max", "max amount of releases to go back per repository").Default("-1").Int()
	version     = "master"
)

func main() {
	kingpin.Version("github_releases_exporter version " + version)
	kingpin.HelpFlag.Short('h')
	kingpin.Parse()
	log.Info("starting github_releases_exporter", version)

	if *debug {
		_ = log.Base().SetLevel("debug")
		log.Debug("enabled debug mode")
	}

	var cche = cache.New(*interval, *interval)
	var cfg config.Config
	config.Load(*configFile, &cfg, func() {
		log.Debug("flushing cache...")
		cche.Flush()
	})

	var ctx = context.Background()
	var client = client.NewCachedClient(
		client.NewClient(ctx, *token, *maxReleases),
		cche,
	)

	prometheus.MustRegister(collector.NewReleasesCollector(&cfg, client))
	http.Handle("/metrics", promhttp.Handler())

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(
			w, `
			<html>
			<head><title>GitHub Releases Exporter</title></head>
			<body>
				<h1>GitHub Releases Exporter</h1>
				<p><a href="/metrics">Metrics</a></p>
			</body>
			</html>
			`,
		)
	})
	log.Info("listening on", *bind)
	if err := http.ListenAndServe(*bind, nil); err != nil {
		log.Fatalf("error starting server: %s", err)
	}
}
