package client

import (
	"fmt"

	"github.com/patrickmn/go-cache"
	"github.com/prometheus/common/log"
)

func NewCachedClient(client Client, cache *cache.Cache) Client {
	return cachedClient{
		client: client,
		cache:  cache,
	}
}

type cachedClient struct {
	client Client
	cache  *cache.Cache
}

func (c cachedClient) Releases(repo string) ([]Release, error) {
	cached, found := c.cache.Get(repo)
	if found {
		log.Debugf("getting releases for %s from cache", repo)
		return cached.([]Release), nil
	}
	log.Debugf("getting releases for %s from API", repo)
	live, err := c.client.Releases(repo)
	c.cache.Set(repo, live, cache.DefaultExpiration)
	return live, err
}

func (c cachedClient) Assets(repo string, id int64) ([]Asset, error) {
	var key = fmt.Sprintf("%s@%d", repo, id)
	cached, found := c.cache.Get(key)
	if found {
		log.Debugf("getting releases assets for %s from cache", key)
		return cached.([]Asset), nil
	}
	log.Debugf("getting releases assets for %s from API", key)
	live, err := c.client.Assets(repo, id)
	c.cache.Set(key, live, cache.DefaultExpiration)
	return live, err
}

func (c cachedClient) GetLatestRelease(repo string) (*LatestRelease, error) {
	cached, found := c.cache.Get(repo)
	if found {
		log.Debugf("getting releases for %s from cache", repo)
		return cached.(*LatestRelease), nil
	}
	log.Debugf("getting releases for %s from API", repo)
	live, err := c.client.GetLatestRelease(repo)
	c.cache.Set(repo, live, cache.DefaultExpiration)
	return live, err
}
