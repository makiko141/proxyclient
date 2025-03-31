package proxyclient

import (
	"net/http"
	"time"
)

type Options struct {
	Timeout   time.Duration
	Client    *http.Client
	Transport *http.Transport
}

type Option func(*Options)

func WithClient(c *http.Client) Option {
	return func(o *Options) {
		o.Client = c
	}
}

func WithTransport(tr *http.Transport) Option {
	return func(o *Options) {
		o.Transport = tr
	}
}

func WithTimeout(d time.Duration) Option {
	return func(o *Options) {
		if d > 0 {
			o.Timeout = d
		}
	}
}
