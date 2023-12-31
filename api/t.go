package api

import (
	"bufio"
	"bytes"
	"fmt"
	fhttp "github.com/bogdanfinn/fhttp"
	tlscli "github.com/bogdanfinn/tls-client"
	"github.com/bogdanfinn/tls-client/profiles"
	"io"
	"log"
	"net/http"
	"net/textproto"
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

	//request.Header = fhttp.Header{
	//	"accept":          {"*/*"},
	//	"accept-language": {"de-DE,de;q=0.9,en-US;q=0.8,en;q=0.7"},
	//	"user-agent":      {"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/100.0.4896.75 Safari/537.36"},
	//	fhttp.HeaderOrderKey: {
	//		"accept",
	//		"accept-language",
	//		"user-agent",
	//	},
	//}

	copyHeader(request.Header, req.Header, nil)
	var partialResponse *fhttp.Response
	partialResponse, err = client.Do(request)
	if err != nil {
		log.Printf("http: proxy error: %v", err)
		rw.WriteHeader(http.StatusBadGateway)
		return
	}

	copyHeader(rw.Header(), partialResponse.Header, []string{
		//"Content-Encoding",
		"Content-Type",
		"Accept-Language",
		"Cookie",
	})
	rw.WriteHeader(partialResponse.StatusCode)

	if err = copyBody(rw, partialResponse); err != nil {
		log.Printf("http: proxy error: %v", err)
		rw.WriteHeader(http.StatusBadGateway)
		return
	}
}

func copyHeader(dst map[string][]string, src map[string][]string, ignores []string) {
	for k, vv := range src {
		if ignores != nil && !containFor(ignores, k) {
			continue
		}
		for _, v := range vv {
			//dst.Add(k, v)
			textproto.MIMEHeader(dst).Add(k, v)
		}
	}
}

func copyBody(rw http.ResponseWriter, response *fhttp.Response) error {
	reader := bufio.NewReader(response.Body)
	cache := make([]byte, 0)
	defer func() {
		fmt.Println(string(cache))
	}()

	for {
		readLine, _, err := reader.ReadLine()

		if err == io.EOF {
			return nil
		}

		if err != nil {
			return err
		}

		cache = append(cache, readLine...)
		_, err = rw.Write(readLine)
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

func NewTlsProxy(addr string) SingleProxy {
	return &TlsProxy{
		path: addr,
	}
}
