package proxy

type ProxyCacheItem struct {
	ClusterURL string
	NS         string
	Route      string
	Scheme     string
	OsoToken   string
}

func NewProxyCacheItem(ns string, scheme string, route string, clusterURL string, osoToken string) ProxyCacheItem {
	return ProxyCacheItem{
		NS:         ns,
		Scheme:     scheme,
		Route:      route,
		ClusterURL: clusterURL,
		OsoToken:   osoToken,
	}
}
