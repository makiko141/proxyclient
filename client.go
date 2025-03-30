package proxyclient

import (
	"errors"
	"net/http"
	"net/url"
	"strings"
)

var (
	ErrUnknownProxy = errors.New("proxyclient: unknown proxy protocol")
)

func New(proxyURL string) (*http.Client, error) {
	return With(proxyURL, &http.Client{})
}

func With(proxyURL string, c *http.Client) (*http.Client, error) {
	if proxyURL == "" {
		return c, nil
	}

	u, err := url.Parse(proxyURL)
	if err != nil {
		return nil, err
	}

	f, ok := supportProxies[strings.ToLower(u.Scheme)]
	if !ok {
		return nil, ErrUnknownProxy
	}

	c.Transport = f(u)

	return c, nil
}
