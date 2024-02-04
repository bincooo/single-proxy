package api

import (
	"bufio"
	"github.com/bincooo/requests"
	_ "github.com/bincooo/requests"
	"github.com/bincooo/requests/models"
	"github.com/bincooo/requests/url"
	"io"
	"log"
	"net/http"
	"net/textproto"
	"strings"
	"time"
)

var (
	timeout int
	proxies string
	JA3     = "771,4865-4866-4867-49195-49199-49196-49200-52393-52392-49171-49172-156-157-47-53,0-23-65281-10-11-35-16-5-13-18-51-45-43-27-17513-21,29-23-24,0"
)

type Mapper struct {
	Addr    string
	Ja3     bool
	Static  bool
	Proxies string
	Routes  []Route
}

type Route struct {
	Path    string
	Rewrite string
	Action  []string
	Content []string
}

type Ja3Proxies struct {
	proxies string
	path    string
	route   Route
}

func (t *Ja3Proxies) Path() string {
	return t.path
}

func (t *Ja3Proxies) Route() Route {
	return t.route
}

func (t *Ja3Proxies) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
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
	partialResponse, err = requests.RequestStream(req.Method, t.path+req.RequestURI, request)
	if err != nil {
		log.Printf("http: proxy error: %v", err)
		rw.WriteHeader(http.StatusBadGateway)
		return
	}

	copyHeader(rw.Header(), partialResponse.Headers, []string{
		"Content-Encoding",
		"Content-Length",
	})

	rw.WriteHeader(partialResponse.StatusCode)
	if err = copyResponse(rw, partialResponse); err != nil {
		log.Printf("http: proxy error: %v", err)
		rw.WriteHeader(http.StatusBadGateway)
		return
	}
}

func copyHeader(dst map[string][]string, src map[string][]string, ignores []string) {
	for k, vv := range src {
		if ignores != nil && containFor(ignores, k) {
			continue
		}
		for _, v := range vv {
			textproto.MIMEHeader(dst).Add(k, v)
		}
	}
}

func hasHeader(headers map[string][]string) bool {
	for k, vv := range headers {
		if k != "Content-Type" {
			continue
		}
		for _, v := range vv {
			if strings.Contains(v, "text/html") {
				return true
			}
			if strings.Contains(v, "application/json") {
				return true
			}
		}
	}
	return false
}

func copyResponse(rw http.ResponseWriter, response *models.Response) error {
	log.Printf("response: %d\n", response.StatusCode)

	if hasHeader(response.Headers) {
		data, err := io.ReadAll(response.Body)
		if err != nil {
			return err
		}
		encoding := response.Headers.Get("Content-Encoding")
		requests.DecompressBody(&data, encoding)
		_, _ = rw.Write(data)
		return nil
	}

	reader := bufio.NewReader(response.Body)

	for {
		readLine, _, err := reader.ReadLine()

		if err == io.EOF {
			return nil
		}

		if err != nil {
			return err
		}

		_, err = rw.Write(append(readLine, '\n'))
		if err != nil {
			return err
		}
		rw.(http.Flusher).Flush()
	}
}

func containFor[T comparable](slice []T, t T) bool {
	if slice == nil {
		return false
	}
	for _, item := range slice {
		if item == t {
			return true
		}
	}
	return false
}

func newJa3Proxies(addr string, route Route, proxies string) Proxies {
	return &Ja3Proxies{proxies, addr, route}
}
