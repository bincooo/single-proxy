package api

import (
	"log"
	"net/http"
	"net/http/httputil"
	"regexp"
	"strings"
)

func ProxyAPI(w http.ResponseWriter, r *http.Request) {
	b, err := httputil.DumpRequest(r, true)
	if err != nil {
		log.Printf("%v\n", err)
		return
	} else {
		log.Printf("%s\n", string(b))
	}

	args := ""
	uri := r.RequestURI
	if idx := strings.Index(uri, "?"); idx >= 0 {
		uri = r.RequestURI[:idx]
		args = r.RequestURI[idx:]
	}
	if uri == "" || uri == "/" {
		index(r.Host, w)
		return
	}

	log.Printf("proxy uri: %s, args: %s\n", uri, args)

	if proxy, ok := pMapper[uri]; ok {
		proxy.ServeHTTP(w, r)
		return
	}

	var prefix SingleProxy
	var routeAll SingleProxy

	for k, proxy := range pMapper {
		if strings.HasPrefix(k, "reg:") {
			compile := regexp.MustCompile(k[4:])
			if compile.MatchString(uri) {
				log.Printf("proxy target: %v\n\n\n", proxy.Path())
				proxy.ServeHTTP(w, r)
				return
			}
		} else if prefix == nil && strings.HasPrefix(uri, k) {
			prefix = proxy
		} else if routeAll == nil && k == "*" {
			routeAll = proxy
		}
	}

	if prefix != nil {
		log.Printf("proxy target * : %v\n\n\n", routeAll.Path())
		prefix.ServeHTTP(w, r)
		return
	}

	if routeAll != nil {
		log.Printf("proxy target * : %v\n\n\n", routeAll.Path())
		routeAll.ServeHTTP(w, r)
		return
	}
}

func index(host string, w http.ResponseWriter) {
	_, err := w.Write([]byte("Start by http[s]://" + host + "\n\nversion: " + VERSION + "\nproject: https://github.com/bincooo/single-proxy"))
	if err != nil {
		log.Printf("%v\n", err)
	}
}
