package util

import (
	"context"
	"crypto/tls"
	"log"
	"net/http/httptrace"
	"net/textproto"
)

const prefix = "[CT]"

var secretHeaders = map[string][]string{"X-Sf-Token": {"***"}}

func WithClientTracing(ctx context.Context) context.Context {
	trace := httptrace.ClientTrace{
		GetConn: func(hostPort string) {
			log.Println(prefix, "GetConn")
		},
		GotConn: func(info httptrace.GotConnInfo) {
			log.Println(prefix, "GotConn", info)
		},
		PutIdleConn: func(err error) {
			log.Println(prefix, "PutIdleConn", err)
		},
		GotFirstResponseByte: func() {
			log.Println(prefix, "GotFirstResponseByte")
		},
		Got100Continue: func() {
			log.Println(prefix, "Got100Continue")
		},
		Got1xxResponse: func(code int, header textproto.MIMEHeader) error {
			log.Println(prefix, "Got1xxResponse", code, header)
			return nil
		},
		DNSStart: func(info httptrace.DNSStartInfo) {
			log.Println(prefix, "DNSStart", info)
		},
		DNSDone: func(info httptrace.DNSDoneInfo) {
			log.Println(prefix, "DNSDone", info)
		},
		ConnectStart: func(network, addr string) {
			log.Println(prefix, "ConnectStart", network, addr)
		},
		ConnectDone: func(network, addr string, err error) {
			log.Println(prefix, "ConnectDone", network, addr, err)
		},
		TLSHandshakeStart: func() {
			log.Println(prefix, "TLSHandshakeStart")
		},
		TLSHandshakeDone: func(state tls.ConnectionState, err error) {
			log.Println(prefix, "TLSHandshakeDone", state, err)
		},
		WroteHeaderField: func(key string, value []string) {
			if secret, ok := secretHeaders[key]; ok {
				value = secret
			}
			log.Println(prefix, "WroteHeaderField", key, value)
		},
		WroteHeaders: func() {
			log.Println(prefix, "WroteHeaders")
		},
		Wait100Continue: func() {
			log.Println(prefix, "Wait100Continue")
		},
		WroteRequest: func(info httptrace.WroteRequestInfo) {
			log.Println(prefix, "WroteRequest", info)
		},
	}

	return httptrace.WithClientTrace(ctx, &trace)
}
