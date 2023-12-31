package api

import (
	"bufio"
	"bytes"
	fhttp "github.com/bogdanfinn/fhttp"
	tlscli "github.com/bogdanfinn/tls-client"
	"github.com/bogdanfinn/tls-client/profiles"
	"io"
	"log"
	"net/http"
	"os"
)

var client tlscli.HttpClient

type TlsProxy struct {
	path string
}

func init() {
	options := []tlscli.HttpClientOption{
		tlscli.WithTimeoutSeconds(30),
		tlscli.WithClientProfile(profiles.Chrome_105),
		tlscli.WithNotFollowRedirects(),
	}

	c, err := tlscli.NewHttpClient(tlscli.NewNoopLogger(), options...)
	if err != nil {
		log.Fatal(err)
	}
	client = c
	p := LoadEnvVar("PROXY", "")
	if p != "" {
		if err = client.SetProxy(p); err != nil {
			log.Printf("%v\n", err)
			os.Exit(-1)
		}
	}
}

func (t *TlsProxy) Path() string {
	return t.path
}

func (t *TlsProxy) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	b, err := io.ReadAll(req.Body)
	if err != nil {
		log.Printf("http: proxy error: %v", err)
		rw.WriteHeader(http.StatusBadGateway)
		return
	}

	var request *fhttp.Request
	request, err = fhttp.NewRequest(http.MethodGet, t.path+req.RequestURI, bytes.NewReader(b))
	if err != nil {
		log.Printf("http: proxy error: %v", err)
		rw.WriteHeader(http.StatusBadGateway)
		return
	}

	request.Header = fhttp.Header{
		"accept":          {"*/*"},
		"accept-language": {"de-DE,de;q=0.9,en-US;q=0.8,en;q=0.7"},
		"user-agent":      {"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/100.0.4896.75 Safari/537.36"},
		fhttp.HeaderOrderKey: {
			"accept",
			"accept-language",
			"user-agent",
		},
	}

	var partialResponse *fhttp.Response
	partialResponse, err = client.Do(request)
	if err != nil {
		log.Printf("http: proxy error: %v", err)
		rw.WriteHeader(http.StatusBadGateway)
		return
	}

	rw.WriteHeader(partialResponse.StatusCode)
	copyHeader(rw.Header(), partialResponse.Header)
	if err = copyBody(rw, partialResponse); err != nil {
		log.Printf("http: proxy error: %v", err)
		rw.WriteHeader(http.StatusBadGateway)
		return
	}
}

func copyHeader(dst http.Header, src fhttp.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

func copyBody(rw http.ResponseWriter, response *fhttp.Response) error {
	reader := bufio.NewReader(response.Body)

	for {
		readLine, _, err := reader.ReadLine()

		if err == io.EOF {
			return nil
		}

		if err != nil {
			return err
		}

		_, err = rw.Write(readLine)
		if err != nil {
			return err
		}
	}
}

func NewTlsProxy(addr string) SingleProxy {
	return &TlsProxy{
		path: addr,
	}
}
