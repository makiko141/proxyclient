package proxyclient

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"testing"
	"time"
)

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

// Simplified SOCKS4 handler for testing
func handleSocks4(conn net.Conn, t *testing.T) {
	defer conn.Close()

	// Read SOCKS4 request
	buf := make([]byte, 128)
	n, err := conn.Read(buf)
	if err != nil || n < 9 {
		t.Logf("Error reading SOCKS4 request: %v", err)
		return
	}

	// Extract destination port and IP
	port := uint16(buf[2])<<8 | uint16(buf[3])
	ip := net.IPv4(buf[4], buf[5], buf[6], buf[7])

	// Skip user ID (read until null byte)
	// userIDEnd := 8
	// for i := 8; i < n; i++ {
	// 	if buf[i] == 0 {
	// 		userIDEnd = i
	// 		break
	// 	}
	// }

	// Connect to target
	target := fmt.Sprintf("%s:%d", ip.String(), port)
	targetConn, err := net.DialTimeout("tcp", target, 10*time.Second)
	if err != nil {
		// Send failure response
		conn.Write([]byte{0, 91, 0, 0, 0, 0, 0, 0}) // Request rejected
		return
	}

	// Send success response
	conn.Write([]byte{0, 90, 0, 0, 0, 0, 0, 0}) // Request granted

	// Proxy data between client and target
	go transfer(targetConn, conn)
	transfer(conn, targetConn)
}

// Simplified SOCKS5 handler for testing
func handleSocks5(conn net.Conn, t *testing.T) {
	defer conn.Close()

	// 1. Read auth methods negotiation
	buf := make([]byte, 2)
	_, err := io.ReadFull(conn, buf)
	if err != nil || buf[0] != 5 {
		t.Logf("Error reading SOCKS5 version: %v", err)
		return
	}

	numMethods := int(buf[1])
	methods := make([]byte, numMethods)
	_, err = io.ReadFull(conn, methods)
	if err != nil {
		t.Logf("Error reading SOCKS5 auth methods: %v", err)
		return
	}

	// 2. Send auth method choice (no auth: 0x00)
	conn.Write([]byte{5, 0})

	// 3. Read connection request
	header := make([]byte, 4)
	_, err = io.ReadFull(conn, header)
	if err != nil || header[0] != 5 {
		t.Logf("Error reading SOCKS5 request: %v", err)
		return
	}

	// Extract address type and destination
	cmd := header[1]
	atyp := header[3]

	var host string
	var port uint16

	switch atyp {
	case 1: // IPv4
		addr := make([]byte, 4)
		_, err = io.ReadFull(conn, addr)
		if err != nil {
			conn.Write([]byte{5, 1, 0, 1, 0, 0, 0, 0, 0, 0}) // General failure
			return
		}
		host = net.IPv4(addr[0], addr[1], addr[2], addr[3]).String()

	case 3: // Domain name
		lenByte := make([]byte, 1)
		_, err = io.ReadFull(conn, lenByte)
		if err != nil {
			conn.Write([]byte{5, 1, 0, 1, 0, 0, 0, 0, 0, 0})
			return
		}
		domainLen := int(lenByte[0])
		domain := make([]byte, domainLen)
		_, err = io.ReadFull(conn, domain)
		if err != nil {
			conn.Write([]byte{5, 1, 0, 1, 0, 0, 0, 0, 0, 0})
			return
		}
		host = string(domain)

	case 4: // IPv6
		addr := make([]byte, 16)
		_, err = io.ReadFull(conn, addr)
		if err != nil {
			conn.Write([]byte{5, 1, 0, 1, 0, 0, 0, 0, 0, 0})
			return
		}
		host = net.IP(addr).String()

	default:
		conn.Write([]byte{5, 8, 0, 1, 0, 0, 0, 0, 0, 0}) // Address type not supported
		return
	}

	// Read port
	portBytes := make([]byte, 2)
	_, err = io.ReadFull(conn, portBytes)
	if err != nil {
		conn.Write([]byte{5, 1, 0, 1, 0, 0, 0, 0, 0, 0})
		return
	}
	port = uint16(portBytes[0])<<8 | uint16(portBytes[1])

	// Connect to destination if CMD is CONNECT (1)
	if cmd != 1 {
		conn.Write([]byte{5, 7, 0, 1, 0, 0, 0, 0, 0, 0}) // Command not supported
		return
	}

	targetConn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", host, port), 10*time.Second)
	if err != nil {
		conn.Write([]byte{5, 4, 0, 1, 0, 0, 0, 0, 0, 0}) // Host unreachable
		return
	}

	// Send success response
	conn.Write([]byte{5, 0, 0, 1, 0, 0, 0, 0, 0, 0}) // Success

	// Proxy data between client and target
	go transfer(targetConn, conn)
	transfer(conn, targetConn)
}
