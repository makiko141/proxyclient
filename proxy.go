package proxyclient

import (
	"net/http"
	"net/url"
)

type ProxyFunc func(*url.URL, *Options) http.RoundTripper

var (
	supportProxies = make(map[string]ProxyFunc)
)

func RegisterProxy(proto string, f ProxyFunc) {
	supportProxies[proto] = f
}

func createTransport(o *Options) *http.Transport {
	return o.Transport.Clone()
}
