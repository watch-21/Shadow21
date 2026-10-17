package netcheck

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

const (
	torCheckURL     = "https://check.torproject.org/api/ip"
	maxResponseSize = 64 << 10
)

type Result struct {
	IP        string
	ElapsedMS int64
	IsTor     bool
}

type Checker struct {
	Endpoint  string
	url       string
	transport *http.Transport
}

type apiResponse struct {
	IP    string `json:"IP"`
	IsTor *bool  `json:"IsTor"`
}

func (checker Checker) Check(ctx context.Context) (Result, error) {
	started := time.Now()
	url := checker.url
	if url == "" {
		url = torCheckURL
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return Result{}, err
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Cache-Control", "no-cache, no-store")
	request.Header.Set("Pragma", "no-cache")

	transport := http.DefaultTransport.(*http.Transport).Clone()
	if checker.transport != nil {
		transport = checker.transport.Clone()
	}
	// DefaultTransport could take this path:
	//   request -> HTTP_PROXY -> proxy -> Tor Project
	// The check must instead exercise:
	//   request -> normal socket -> AnonSurf iptables -> Tor -> Tor Project
	// Otherwise a Tor-enabled proxy could return IsTor=true while AnonSurf's
	// transparent routing is broken. Clone keeps the transport defaults; clearing
	// Proxy forces the connection through the system route being verified.
	transport.Proxy = nil
	transport.DisableKeepAlives = true
	defer transport.CloseIdleConnections()
	if checker.Endpoint != "" {
		transport.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
			return DialSOCKS(ctx, checker.Endpoint, address)
		}
	}
	client := &http.Client{
		Transport: transport,
		Timeout:   20 * time.Second,
		CheckRedirect: func(request *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	response, err := client.Do(request)
	if err != nil {
		return Result{}, err
	}
	defer response.Body.Close()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return Result{}, fmt.Errorf("Tor check returned HTTP %d", response.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(response.Body, maxResponseSize+1))
	if err != nil {
		return Result{}, err
	}
	if len(body) > maxResponseSize {
		return Result{}, fmt.Errorf("Tor check response exceeds %d bytes", maxResponseSize)
	}
	result, err := parseTorCheckResponse(string(body))
	if err == nil {
		result.ElapsedMS = time.Since(started).Milliseconds()
	}
	return result, err
}

func parseTorCheckResponse(body string) (Result, error) {
	var response apiResponse
	if err := json.Unmarshal([]byte(body), &response); err != nil {
		return Result{}, fmt.Errorf("decode Tor check response: %w", err)
	}
	ip := strings.TrimSpace(response.IP)
	if net.ParseIP(ip) == nil {
		return Result{}, fmt.Errorf("Tor check response has invalid public IP %q", response.IP)
	}
	if response.IsTor == nil {
		return Result{}, fmt.Errorf("Tor check response does not include Tor status")
	}
	return Result{IP: ip, IsTor: *response.IsTor}, nil
}
