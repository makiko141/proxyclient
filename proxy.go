package proxyclient

import (
	"net/http"
	"net/url"
)

type ProxyFunc func(*url.URL, *Options) http.RoundTripper

var (
	supportProxies = make(map[string]ProxyFunc)
)

func createTransport(o *Options) *http.Transport {
	return o.Transport.Clone()
}
