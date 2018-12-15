package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/alecthomas/kingpin"
	"github.com/google/go-github/github"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/log"
	"golang.org/x/oauth2"
	yaml "gopkg.in/yaml.v1"
)

// nolint: gochecknoglobals
var (
	bind       = kingpin.Flag("bind", "addr to bind the server").Short('b').Default(":9333").String()
	debug      = kingpin.Flag("debug", "show debug logs").Default("false").Bool()
	token      = kingpin.Flag("github.token", "github token").Envar("GITHUB_TOKEN").String()
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

	var ctx = context.Background()
	var client = github.NewClient(nil)
	if *token != "" {
		client = github.NewClient(oauth2.NewClient(
			ctx,
			oauth2.StaticTokenSource(&oauth2.Token{AccessToken: *token}),
		))
	}

	var config Config
	loadConfig(&config)
	var configCh = make(chan os.Signal, 1)
	signal.Notify(configCh, syscall.SIGHUP)
	go func() {
		for range configCh {
			log.Info("reloading config...")
			downloadCount.Reset()
			loadConfig(&config)
			if err := collectOnce(ctx, client, &config); err != nil {
				log.Error("failed to collect:", err)
			}
		}
	}()

	go keepCollecting(ctx, client, &config)
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

// nolint: gochecknoglobals
var (
	scrapeDuration = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: prometheus.BuildFQName(ns, "", "scrape_duration_seconds"),
		Help: "Returns how long the probe took to complete in seconds",
	})
	downloadCount = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: prometheus.BuildFQName(ns, "asset", "download_count"),
		Help: "Download count of each asset of a github release",
	},
		[]string{"repository", "tag", "name"},
	)
)

func keepCollecting(ctx context.Context, client *github.Client, config *Config) {
	for {
		if err := collectOnce(ctx, client, config); err != nil {
			log.Error("failed to collect:", err)
		}
		time.Sleep(*interval)
	}
}

func collectOnce(ctx context.Context, client *github.Client, config *Config) error {
	log.Infof("collecting %d repositories", len(config.Repositories))

	var start = time.Now()
	for _, repository := range config.Repositories {
		owner, repo, err := splitRepo(repository)
		if err != nil {
			return err
		}
		log.Infof("collecting %s/%s", owner, repo)
		var allReleases []*github.RepositoryRelease
		var opt = &github.ListOptions{PerPage: 100}
		for {
			releases, resp, err := client.Repositories.ListReleases(ctx, owner, repo, opt)
			if rateLimited(err) {
				continue
			}
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
				if rateLimited(err) {
					continue
				}
				if err != nil {
					return err
				}
				for _, asset := range assets {
					// nolint: megacheck
					downloadCount.WithLabelValues(
						repository,
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

// TODO: maybe keep a global rate limiter?
func rateLimited(err error) bool {
	if err == nil {
		return false
	}
	rerr, ok := err.(*github.RateLimitError)
	if ok {
		var d = time.Until(rerr.Rate.Reset.Time)
		log.Warnf("hit rate limit, sleeping for %.0f min", d.Minutes())
		time.Sleep(d)
		return true
	}
	aerr, ok := err.(*github.AbuseRateLimitError)
	if ok {
		var d = aerr.GetRetryAfter()
		log.Warnf("hit abuse mechanism, sleeping for %.f min", d.Minutes())
		time.Sleep(d)
		return true
	}
	return false
}

func loadConfig(config *Config) {
	bts, err := ioutil.ReadFile(*configFile)
	if err != nil {
		log.Fatal(err)
	}
	var newConfig Config
	if err := yaml.Unmarshal(bts, &newConfig); err != nil {
		log.Fatal(err)
	}
	*config = newConfig
}

func splitRepo(repository string) (owner string, repo string, err error) {
	parts := strings.Split(repository, "/")
	if len(parts) < 2 {
		return owner, repo, fmt.Errorf("invalid repository: %s. should be in the owner/repo format", repository)
	}
	owner, repo = parts[0], parts[1]
	if owner == "" || repo == "" {
		return owner, repo, fmt.Errorf("invalid repository: %s. should be in the owner/repo format", repository)
	}
	return
}
