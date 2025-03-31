package proxyclient

import (
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestProxyHTTP(t *testing.T) {
	// Create a test server that we'll try to access through the proxy
	targetServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello from target server")
	}))
	defer targetServer.Close()

	t.Run("http", func(t *testing.T) {
		var proxyWasUsed bool
		// Create a mock proxy server
		proxyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			proxyWasUsed = true

			if r.Method == http.MethodConnect {
				handleTunneling(w, r)
			} else {
				handleHTTP(w, r)
			}
		}))
		defer proxyServer.Close()

		client, err := New(proxyServer.URL)
		require.NoError(t, err)

		// Make request to target server
		resp, err := client.Get(targetServer.URL)
		if err != nil {
			t.Fatalf("Failed to make request through proxy: %v", err)
		}
		defer resp.Body.Close()

		// Check if proxy was used
		if !proxyWasUsed {
			t.Error("Proxy was not used for the request")
		}

		// Verify response status
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status OK, got %v", resp.Status)
		}

		// Verify response body
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("Failed to read response body: %v", err)
		}

		expectedBody := "Hello from target server"
		if string(body) != expectedBody {
			t.Errorf("Expected body %q, got %q", expectedBody, string(body))
		}
	})

	t.Run("https", func(t *testing.T) {
		var proxyWasUsed bool

		proxyServer := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			proxyWasUsed = true
			if r.Method == http.MethodConnect {
				handleTunneling(w, r)
			} else {
				handleHTTP(w, r)
			}
		}))

		defer proxyServer.Close()

		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}

		client, err := New(proxyServer.URL, WithTransport(tr), WithTimeout(1*time.Millisecond))
		require.NoError(t, err)

		resp, err := client.Get(targetServer.URL)
		if err != nil {
			t.Fatalf("Failed to make request through proxy: %v", err)
		}
		defer resp.Body.Close()

		// Check if proxy was used
		if !proxyWasUsed {
			t.Error("Proxy was not used for the request")
		}

		// Verify response status
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status OK, got %v", resp.Status)
		}

		// Verify response body
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("Failed to read response body: %v", err)
		}

		expectedBody := "Hello from target server"
		if string(body) != expectedBody {
			t.Errorf("Expected body %q, got %q", expectedBody, string(body))
		}
	})

}

func handleTunneling(w http.ResponseWriter, r *http.Request) {
	dest_conn, err := net.DialTimeout("tcp", r.Host, 10*time.Second)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "Hijacking not supported", http.StatusInternalServerError)
		return
	}
	client_conn, _, err := hijacker.Hijack()
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
	}
	go transfer(dest_conn, client_conn)
	go transfer(client_conn, dest_conn)
}
func transfer(destination io.WriteCloser, source io.ReadCloser) {
	defer destination.Close()
	defer source.Close()
	io.Copy(destination, source)
}
func handleHTTP(w http.ResponseWriter, req *http.Request) {
	resp, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	defer resp.Body.Close()
	copyHeader(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}
func copyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}
