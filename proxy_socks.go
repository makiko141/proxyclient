package proxyclient

import (
	"context"
	"net"
	"net/http"
	"net/url"

	"golang.org/x/net/proxy"
	"h12.io/socks"
)

func init() {
	supportProxies["socks5"] = ProxySocks5
	supportProxies["socks5h"] = ProxySocks5
	supportProxies["socks4"] = ProxySocks4
	supportProxies["socks4a"] = ProxySocks4

}

func ProxySocks5(u *url.URL, o Options) http.RoundTripper {

	transport := createTransport()

	for _, it := range o.WithTransport {
		it(transport)
	}

	dialer := &net.Dialer{}

	if o.DialTimeout > 0 {
		dialer.Timeout = o.DialTimeout
	}

	var auth *proxy.Auth
	if u.User != nil {
		auth = new(proxy.Auth)
		auth.User = u.User.Username()
		if p, ok := u.User.Password(); ok {
			auth.Password = p
		}
	}

	addr := u.Hostname()
	port := u.Port()
	if port == "" {
		port = "1080"
	}

	d, _ := proxy.SOCKS5("tcp", net.JoinHostPort(addr, port), auth, dialer)

	if xd, ok := d.(proxy.ContextDialer); ok {
		transport.DialContext = xd.DialContext
		transport.DialTLSContext = xd.DialContext
	} else {
		transport.Dial = d.Dial
		transport.DialTLS = dialer.Dial
	}

	return transport
}

func ProxySocks4(u *url.URL, o Options) http.RoundTripper {
	tr := createTransport()

	for _, it := range o.WithTransport {
		it(tr)
	}

	proxyURL := u.String()

	if o.DialTimeout > 0 {
		proxyURL += "?timeout=" + o.DialTimeout.String()

	}

	tr.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		return socks.Dial(proxyURL)(network, addr)
	}
	tr.DialTLSContext = tr.DialContext

	return tr
}
