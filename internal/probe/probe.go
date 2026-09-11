package probe

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"
)

type Result struct {
	Name    string
	Target  string
	OK      bool
	Latency time.Duration
	Error   string
}

type Config struct {
	DNSHost    string
	TCPAddress string
	HTTPURL    string
	Timeout    time.Duration
}

var httpClient = &http.Client{
	Transport: &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		MaxIdleConns:          4,
		MaxIdleConnsPerHost:   2,
		IdleConnTimeout:       30 * time.Second,
		TLSHandshakeTimeout:   5 * time.Second,
		ResponseHeaderTimeout: 5 * time.Second,
	},
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	},
}

func DNS(ctx context.Context, cfg Config) Result {
	start := time.Now()
	probeCtx, cancel := context.WithTimeout(ctx, cfg.Timeout)
	defer cancel()

	_, err := net.DefaultResolver.LookupHost(probeCtx, cfg.DNSHost)
	latency := time.Since(start)

	if err != nil {
		return Result{
			Name:    "dns",
			Target:  cfg.DNSHost,
			Latency: latency,
			Error:   err.Error(),
		}
	}

	return Result{
		Name:    "dns",
		Target:  cfg.DNSHost,
		OK:      true,
		Latency: latency,
	}
}

func TCP(ctx context.Context, cfg Config) Result {
	start := time.Now()
	probeCtx, cancel := context.WithTimeout(ctx, cfg.Timeout)
	defer cancel()

	var dialer net.Dialer
	conn, err := dialer.DialContext(probeCtx, "tcp", cfg.TCPAddress)
	latency := time.Since(start)

	if err != nil {
		return Result{
			Name:    "tcp",
			Target:  cfg.TCPAddress,
			Latency: latency,
			Error:   err.Error(),
		}
	}

	_ = conn.Close()

	return Result{
		Name:    "tcp",
		Target:  cfg.TCPAddress,
		OK:      true,
		Latency: latency,
	}
}

func HTTP(ctx context.Context, cfg Config) Result {
	start := time.Now()
	probeCtx, cancel := context.WithTimeout(ctx, cfg.Timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(probeCtx, http.MethodGet, cfg.HTTPURL, nil)
	if err != nil {
		return Result{
			Name:   "http",
			Target: cfg.HTTPURL,
			Error:  err.Error(),
		}
	}

	resp, err := httpClient.Do(req)
	latency := time.Since(start)

	if err != nil {
		return Result{
			Name:    "http",
			Target:  cfg.HTTPURL,
			Latency: latency,
			Error:   err.Error(),
		}
	}
	defer resp.Body.Close()

	_, _ = io.Copy(io.Discard, resp.Body)

	if resp.StatusCode != http.StatusNoContent {
		return Result{
			Name:    "http",
			Target:  cfg.HTTPURL,
			Latency: latency,
			Error:   fmt.Sprintf("unexpected HTTP status %d", resp.StatusCode),
		}
	}

	return Result{
		Name:    "http",
		Target:  cfg.HTTPURL,
		OK:      true,
		Latency: latency,
	}
}
