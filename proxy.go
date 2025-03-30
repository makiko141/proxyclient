package proxyclient

import (
	"net/http"
	"net/url"
)

type ProxyFunc func(*url.URL, Options) http.RoundTripper

var (
	supportProxies = make(map[string]ProxyFunc)
)

func createTransport() *http.Transport {
	return http.DefaultTransport.(*http.Transport).Clone()
}
