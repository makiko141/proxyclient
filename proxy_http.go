package proxyclient

import (
	"net"
	"net/http"
	"net/url"
)

func init() {
	supportProxies["http"] = ProxyHTTP
	supportProxies["https"] = ProxyHTTP
}

func ProxyHTTP(u *url.URL, o Options) http.RoundTripper {

	tr := createTransport()

	for _, it := range o.WithTransport {
		it(tr)
	}

	tr.Proxy = http.ProxyURL(u)

	d := &net.Dialer{}
	if o.DialTimeout > 0 {
		d.Timeout = o.DialTimeout
	}

	tr.DialContext = d.DialContext
	tr.DialTLSContext = d.DialContext

	return tr
}
