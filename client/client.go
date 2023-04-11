package client

type Client interface {
	Releases(repo string) ([]Release, error)
	Assets(repo string, id int64) ([]Asset, error)
	GetLatestRelease(repo string) (*LatestRelease, error)
}

type Release struct {
	ID  int64
	Tag string
}

type LatestRelease struct {
	Tag      string
	Name     string
	UnixTime int64
}

type Asset struct {
	Name      string
	Downloads int
}
