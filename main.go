package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/alecthomas/kingpin"
	"github.com/google/go-github/github"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/log"
	"golang.org/x/oauth2"
)

var (
	bind    = kingpin.Flag("bind", "addr to bind the server").Short('b').Default(":9333").String()
	debug   = kingpin.Flag("debug", "show debug logs").Default("false").Bool()
	token   = kingpin.Flag("github-token", "github token").Envar("GITHUB_TOKEN").String()
	version = "master"
)

func main() {
	kingpin.Version("github_releases_exporter version " + version)
	kingpin.HelpFlag.Short('h')
	kingpin.Parse()

	if *debug {
		_ = log.Base().SetLevel("debug")
		log.Debug("enabled debug mode")
	}

	var ctx = context.Background()
	var client = github.NewClient(nil)
	if *token != "" {
		client = github.NewClient(oauth2.NewClient(
			ctx,
			oauth2.StaticTokenSource(&oauth2.Token{AccessToken: *token}),
		))
	}

	log.Info("starting github_releases_exporter", version)

	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/probe", probeHandler(client))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(
			w, `
			<html>
			<head><title>GitHub Releases Exporter</title></head>
			<body>
				<h1>GitHub Releases Exporter</h1>
				<p><a href="/metrics">Metrics</a></p>
				<p><a href="/probe?target=goreleaser/goreleser">probe goreleaser/goreleaser</a></p>
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

func probeHandler(client *github.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var params = r.URL.Query()
		var target = params.Get("target")
		parts := strings.Split(target, "/")
		if len(parts) < 2 {
			http.Error(w, "invalid target: "+target, http.StatusBadRequest)
			return
		}

		var registry = prometheus.NewRegistry()
		registry.MustRegister(newReleaseCollector(client, parts[0], parts[1]))
		promhttp.HandlerFor(registry, promhttp.HandlerOpts{}).ServeHTTP(w, r)
	}
}

const ns = "github_release"

type releaseCollector struct {
	client         *github.Client
	scrapeDuration *prometheus.Desc
	downloadCount  *prometheus.Desc
	owner          string
	repo           string
}

func newReleaseCollector(client *github.Client, owner, repo string) *releaseCollector {
	return &releaseCollector{
		client: client,
		owner:  owner,
		repo:   repo,
		scrapeDuration: prometheus.NewDesc(
			prometheus.BuildFQName(ns, "", "scrape_duration_seconds"),
			"Returns how long the probe took to complete in seconds",
			nil,
			nil,
		),
		downloadCount: prometheus.NewDesc(
			prometheus.BuildFQName(ns, "asset", "download_count"),
			"Download count of each asset of a github release",
			[]string{"tag", "name"},
			nil,
		),
	}
}

// Describe describes all the metrics exported by the github releases exporter.
func (c *releaseCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.scrapeDuration
	ch <- c.downloadCount
}

// Collect talks to the github api and returns metrics
func (c *releaseCollector) Collect(ch chan<- prometheus.Metric) {
	var ctx = context.Background()
	var start = time.Now()

	var allReleases []*github.RepositoryRelease
	var opt = &github.ListOptions{PerPage: 100}
	for {
		releases, resp, err := c.client.Repositories.ListReleases(ctx, c.owner, c.repo, opt)
		if err != nil {
			log.Error(err)
			return
		}
		allReleases = append(allReleases, releases...)
		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}
	for _, release := range allReleases {
		opt = &github.ListOptions{PerPage: 100}
		for {
			assets, resp, err := c.client.Repositories.ListReleaseAssets(ctx, c.owner, c.repo, release.GetID(), opt)
			if err != nil {
				log.Error(err)
				return
			}
			for _, asset := range assets {
				ch <- prometheus.MustNewConstMetric(
					c.downloadCount,
					prometheus.CounterValue,
					float64(asset.GetDownloadCount()),
					release.GetTagName(),
					asset.GetName(),
				)
			}
			if resp.NextPage == 0 {
				break
			}
			opt.Page = resp.NextPage
		}
	}
	ch <- prometheus.MustNewConstMetric(
		c.scrapeDuration,
		prometheus.GaugeValue,
		time.Since(start).Seconds(),
	)
}
