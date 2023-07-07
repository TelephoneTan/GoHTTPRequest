package net

import (
	"net/url"
	"strconv"
)

const (
	HTTP  = "HTTP"
	SOCKS = "SOCKS"
)

func Port(port int) *int {
	return &port
}

type Proxy struct {
	Type string
	Host string
	Port *int
}

func (p *Proxy) URL() *url.URL {
	if p.Host == "" || p.Port == nil {
		return nil
	}
	t := p.Type
	switch t {
	case SOCKS:
		t = "socks5"
	case HTTP:
		t = "http"
	default:
		return nil
	}
	u, err := url.Parse(t + "://" + p.Host + ":" + strconv.Itoa(*p.Port))
	if err != nil {
		return nil
	}
	return u
}
