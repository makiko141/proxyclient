package proxyclient

import (
	"net"
	"net/http"
	"time"
)

type Options struct {
	DialTimeout   time.Duration
	WithTransport []OptionTransport
}

type OptionDialer func(*net.Dialer)
type OptionTransport func(*http.Transport)
