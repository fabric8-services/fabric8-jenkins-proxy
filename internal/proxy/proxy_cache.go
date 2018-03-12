package proxy

// CacheItem represents a cache item  consisting of cluster URL, namespace, route, scheme(HTTP, HTTPS etc).
type CacheItem struct {
	ClusterURL string
	NS         string
	Route      string
	Scheme     string
}

// NewCacheItem creates an instance of cache item.
func NewCacheItem(ns string, scheme string, route string, clusterURL string) CacheItem {
	return CacheItem{
		NS:         ns,
		Scheme:     scheme,
		Route:      route,
		ClusterURL: clusterURL,
	}
}
