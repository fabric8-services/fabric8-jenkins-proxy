package proxy

type ProxyCacheItem struct {
	ClusterURL string
	NS string
	Route string
	TLS bool
	OsoToken string
}

func NewProxyCacheItem(ns string, tls bool, route string, clusterURL string, osoToken string) ProxyCacheItem {
	return ProxyCacheItem {
		NS: ns,
		TLS: tls,
		Route: route,
		ClusterURL: clusterURL,
		OsoToken: osoToken,
	}
}