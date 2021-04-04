package collector

import (
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/caarlos0/github_releases_exporter/client"
	"github.com/caarlos0/github_releases_exporter/config"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/log"
)

type releasesCollector struct {
	mutex  sync.Mutex
	config *config.Config
	client client.Client

	up             *prometheus.Desc
	downloads      *prometheus.Desc
	scrapeDuration *prometheus.Desc
}

// NewReleasesCollector returns a releases collector
func NewReleasesCollector(config *config.Config, client client.Client) prometheus.Collector {
	const namespace = "github"
	const subsystem = "release"
	return &releasesCollector{
		config: config,
		client: client,
		up: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, "up"),
			"Exporter is being able to talk with GitHub API",
			nil,
			nil,
		),
		downloads: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, "asset_download_count"),
			"Download count of each asset of a github release",
			[]string{"repository", "tag", "name", "extension"},
			nil,
		),
		scrapeDuration: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, "scrape_duration_seconds"),
			"Returns how long the probe took to complete in seconds",
			nil,
			nil,
		),
	}
}

// Describe all metrics
func (c *releasesCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.up
	ch <- c.downloads
	ch <- c.scrapeDuration
}

// Collect all metrics
func (c *releasesCollector) Collect(ch chan<- prometheus.Metric) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	log.Infof("collecting %d repositories", len(c.config.Repositories))

	var start = time.Now()
	var success = 1
	for _, repository := range c.config.Repositories {
		log.Infof("collecting %s", repository)
		releases, err := c.client.Releases(repository)
		if err != nil {
			success = 0
			log.Errorf("failed to collect: %s", err.Error())
			continue
		}

		for _, release := range releases {
			assets, err := c.client.Assets(repository, release.ID)
			if err != nil {
				success = 0
				log.Errorf(
					"failed to collect repo %s, release %s: %s",
					repository,
					release.Tag,
					err.Error(),
				)
				continue
			}
			for _, asset := range assets {
				ext :=  strings.TrimPrefix(filepath.Ext(asset.Name), ".")
				log.Debugf(
					"collecting %s@%s / %s (%s)",
					repository,
					release.Tag,
					asset.Name,
					ext,
				)
				ch <- prometheus.MustNewConstMetric(
					c.downloads,
					prometheus.CounterValue,
					float64(asset.Downloads),
					repository,
					release.Tag,
					asset.Name,
					ext,
				)
			}
		}
	}
	ch <- prometheus.MustNewConstMetric(c.scrapeDuration, prometheus.GaugeValue, time.Since(start).Seconds())
	ch <- prometheus.MustNewConstMetric(c.up, prometheus.GaugeValue, float64(success))
}
