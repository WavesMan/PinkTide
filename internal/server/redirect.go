package server

import (
	"net"
	"net/http"
	"strings"
)

func redirectHandler(listenAddr string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		target := buildRedirectURL(r, listenAddr)
		http.Redirect(w, r, target, http.StatusPermanentRedirect)
	})
}

func buildRedirectURL(r *http.Request, listenAddr string) string {
	host := r.Host
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}

	listenHost := ""
	listenPort := ""
	if h, p, err := net.SplitHostPort(listenAddr); err == nil {
		listenHost = h
		listenPort = p
	} else if strings.HasPrefix(listenAddr, ":") {
		listenPort = strings.TrimPrefix(listenAddr, ":")
	} else if listenAddr != "" {
		listenHost = listenAddr
	}

	if listenHost != "" && listenHost != "0.0.0.0" && listenHost != "::" {
		host = listenHost
	}
	if listenPort != "" && listenPort != "443" {
		host = net.JoinHostPort(host, listenPort)
	}

	return "https://" + host + r.URL.RequestURI()
}
