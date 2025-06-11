package socks5proxy

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"golang.org/x/net/proxy"
)

func NewProxiedHttpClient(socks5Addr string) (*http.Client, error) {

	dialer, err := proxy.SOCKS5("tcp", socks5Addr, nil, proxy.Direct)
	if err != nil {
		return nil, fmt.Errorf("failed to create socks5 dialer: %v", err)
	}

	dialContext := func(ctx context.Context, network, addr string) (net.Conn, error) {
		return dialer.Dial(network, addr)
	}

	transport := &http.Transport{
		DialContext:           dialContext,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second,
	}

	return client, nil
}
