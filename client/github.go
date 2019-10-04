package client

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/go-github/github"
	"github.com/prometheus/common/log"
	"golang.org/x/oauth2"
)

func NewClient(ctx context.Context, token string, maxReleases int) Client {
	var client = github.NewClient(nil)
	if token != "" {
		client = github.NewClient(oauth2.NewClient(
			ctx,
			oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token}),
		))
	}
	return githubClient{
		ctx:    ctx,
		client: client,
		max:    maxReleases,
	}
}

type githubClient struct {
	client *github.Client
	ctx    context.Context
	max    int
}

func (c githubClient) Releases(repository string) ([]Release, error) {
	var allReleases []Release
	owner, repo, err := splitRepo(repository)
	if err != nil {
		return allReleases, err
	}
	var opt = &github.ListOptions{PerPage: 100}
	for {
		releases, resp, err := c.client.Repositories.ListReleases(c.ctx, owner, repo, opt)
		if rateLimited(err) {
			continue
		}
		if err != nil {
			return allReleases, err
		}
		for _, release := range releases {
			allReleases = append(allReleases, Release{
				ID:  release.GetID(),
				Tag: release.GetTagName(),
			})
			if c.max > 0 && len(allReleases) >= c.max {
				return allReleases, nil
			}
		}
		if resp.NextPage == 0 {
			return allReleases, nil
		}
		opt.Page = resp.NextPage
	}
}

func (c githubClient) Assets(repository string, id int64) ([]Asset, error) {
	var allAssets = []Asset{}
	owner, repo, err := splitRepo(repository)
	if err != nil {
		return allAssets, err
	}
	var opt = &github.ListOptions{PerPage: 100}
	for {
		assets, resp, err := c.client.Repositories.ListReleaseAssets(c.ctx, owner, repo, id, opt)
		if rateLimited(err) {
			continue
		}
		if err != nil {
			return allAssets, err
		}
		for _, asset := range assets {
			allAssets = append(allAssets, Asset{
				Name:      asset.GetName(),
				Downloads: asset.GetDownloadCount(),
			})
		}
		if resp.NextPage == 0 {
			return allAssets, nil
		}
		opt.Page = resp.NextPage
	}
}

// nolint: godox
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

func splitRepo(repository string) (string, string, error) {
	parts := strings.Split(repository, "/")
	if len(parts) < 2 {
		return "", "", fmt.Errorf(
			"invalid repository: %s. should be in the owner/repo format",
			repository,
		)
	}
	return parts[0], parts[1], nil
}
