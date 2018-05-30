package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/alecthomas/kingpin"
	"github.com/google/go-github/github"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/log"
	"golang.org/x/oauth2"
	yaml "gopkg.in/yaml.v1"
)

var (
	bind       = kingpin.Flag("bind", "addr to bind the server").Short('b').Default(":9333").String()
	debug      = kingpin.Flag("debug", "show debug logs").Default("false").Bool()
	token      = kingpin.Flag("github-token", "github token").Envar("GITHUB_TOKEN").String()
	configFile = kingpin.Flag("config.file", "config file").Default("releases.yml").ExistingFile()
	interval   = kingpin.Flag("refresh.interval", "time between refreshes with github api").Default("15m").Duration()
	version    = "master"
)

type Config struct {
	Repositories []string `yaml:"repositories"`
}

func main() {
	kingpin.Version("github_releases_exporter version " + version)
	kingpin.HelpFlag.Short('h')
	kingpin.Parse()
	log.Info("starting github_releases_exporter", version)

	if *debug {
		_ = log.Base().SetLevel("debug")
		log.Debug("enabled debug mode")
	}

	var config Config
	bts, err := ioutil.ReadFile(*configFile)
	if err != nil {
		log.Fatal(err)
	}
	if err := yaml.Unmarshal(bts, &config); err != nil {
		log.Fatal(err)
	}

	var ctx = context.Background()
	var client = github.NewClient(nil)
	if *token != "" {
		client = github.NewClient(oauth2.NewClient(
			ctx,
			oauth2.StaticTokenSource(&oauth2.Token{AccessToken: *token}),
		))
	}

	go keepCollecting(ctx, client, config)
	prometheus.MustRegister(scrapeDuration)
	prometheus.MustRegister(downloadCount)
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

const ns = "github_release"

var (
	scrapeDuration = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: prometheus.BuildFQName(ns, "", "scrape_duration_seconds"),
		Help: "Returns how long the probe took to complete in seconds",
	})
	downloadCount = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: prometheus.BuildFQName(ns, "asset", "download_count"),
		Help: "Download count of each asset of a github release",
	},
		[]string{"tag", "name"},
	)
)

func keepCollecting(ctx context.Context, client *github.Client, config Config) {
	for {
		if err := collectOnce(ctx, client, config); err != nil {
			log.Fatal(err)
		}
		time.Sleep(*interval)
	}
}

func collectOnce(ctx context.Context, client *github.Client, config Config) error {
	log.Info("collecting")

	var start = time.Now()
	for _, repo := range config.Repositories {
		parts := strings.Split(repo, "/")
		if len(parts) < 2 {
			log.Fatalf("invalid repository: %s", repo)
		}
		owner, repo := parts[0], parts[1]
		log.Infof("collecting %s/%s", owner, repo)
		var allReleases []*github.RepositoryRelease
		var opt = &github.ListOptions{PerPage: 100}
		for {
			releases, resp, err := client.Repositories.ListReleases(ctx, owner, repo, opt)
			if err != nil {
				return err
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
				assets, resp, err := client.Repositories.ListReleaseAssets(ctx, owner, repo, release.GetID(), opt)
				if err != nil {
					return err
				}
				for _, asset := range assets {
					downloadCount.WithLabelValues(
						release.GetTagName(),
						asset.GetName(),
					).Set(float64(asset.GetDownloadCount()))
				}
				if resp.NextPage == 0 {
					break
				}
				opt.Page = resp.NextPage
			}
		}
	}
	scrapeDuration.Set(time.Since(start).Seconds())
	return nil
}
