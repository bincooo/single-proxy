package api

import (
	"github.com/bincooo/requests"
	_ "github.com/bincooo/requests"
	"github.com/bincooo/requests/models"
	"github.com/bincooo/requests/url"
	"io"
	"log"
	"net/http"
	"time"
)

type StaticProxies struct {
	proxies string
	path    string
	route   Route
}

func (t *StaticProxies) Path() string {
	return t.path
}

func (t *StaticProxies) Route() Route {
	return t.route
}

func (t *StaticProxies) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	b, err := io.ReadAll(req.Body)
	if err != nil {
		log.Printf("http: proxy error: %v", err)
		rw.WriteHeader(http.StatusBadGateway)
		return
	}

	request := url.NewRequest()
	request.Timeout = time.Duration(timeout) * time.Second
	request.Headers = url.NewHeaders()
	copyHeader(*request.Headers, req.Header, []string{
		//"Content-Encoding",
		"Content-Length",
	})

	request.Ja3 = JA3
	request.Body = string(b)
	if t.proxies != "" {
		if t.proxies == "auto" {
			withProxies, ferr := fetchPoolWithProxies()
			if ferr != nil {
				log.Printf("%v", ferr)
			} else {
				request.Proxies = withProxies
				goto label
			}
		} else {
			request.Proxies = t.proxies
			goto label
		}
	}

	if proxies != "" {
		request.Proxies = proxies
	}
label:

	var partialResponse *models.Response
	partialResponse, err = requests.Request(req.Method, t.path+req.RequestURI, request)
	if err != nil {
		log.Printf("http: proxy error: %v", err)
		rw.WriteHeader(http.StatusBadGateway)
		return
	}

	if len(t.route.Content) > 0 {
		text, aerr := execAction(partialResponse.Text, t.route.Content)
		if aerr != nil {
			log.Printf("http: proxy error: %v", aerr)
			rw.WriteHeader(http.StatusBadGateway)
			return
		}

		copyHeader(rw.Header(), partialResponse.Headers, []string{
			"Content-Encoding",
			"Content-Length",
		})
		rw.WriteHeader(partialResponse.StatusCode)
		_, _ = rw.Write([]byte(text))
	} else {
		copyHeader(rw.Header(), partialResponse.Headers, []string{
			"Content-Encoding",
			"Content-Length",
		})
		rw.WriteHeader(partialResponse.StatusCode)
		_, _ = rw.Write(partialResponse.Content)
	}
}

func newStaticProxies(addr string, route Route, proxies string) Proxies {
	return &StaticProxies{proxies, addr, route}
}
