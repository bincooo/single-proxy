package api

import (
	"bufio"
	"bytes"
	"github.com/joho/godotenv"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strconv"
	"strings"
)

var (
	pu      *url.URL = nil
	POOL             = make(map[string]SingleProxy)
	PORT             = 8080
	VERSION          = "v1.0.0"
)

type SingleProxy interface {
	ServeHTTP(rw http.ResponseWriter, req *http.Request)
	Path() string
}

type EasyProxy struct {
	path string
	*httputil.ReverseProxy
}

func (e EasyProxy) Path() string {
	return e.path
}

func init() {
	_ = godotenv.Load()
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	PORT = LoadEnvInt("PORT", PORT)
	p := LoadEnvVar("PROXY", "")
	config := LoadEnvVar("CONFIG", "")
	if p != "" {
		proxy, err := url.Parse(p)
		if err != nil {
			log.Printf("%v\n", err)
			os.Exit(-1)
		}
		pu = proxy
	}

	b, err := os.ReadFile("config.ini")
	if err != nil {
		if config == "" {
			log.Printf("%v\n", err)
			os.Exit(-1)
		}

		var response *http.Response
		response, err = http.DefaultClient.Get(config)
		if err != nil {
			log.Printf("%v\n", err)
			os.Exit(-1)
		}
		b, err = io.ReadAll(response.Body)
		if err != nil {
			log.Printf("%v\n", err)
			os.Exit(-1)
		}
		if response.StatusCode != 200 {
			log.Printf("%v\n", "Error: "+string(b))
			os.Exit(-1)
		}
	}

	var (
		prefix          = false
		original        = make([]byte, 0)
		readLine []byte = nil
	)

	reader := bufio.NewReader(bytes.NewReader(b))
	for {
		readLine, prefix, err = reader.ReadLine()
		if err == io.EOF {
			return
		}

		if prefix {
			original = append(original, readLine...)
			continue
		}

		content := string(append(original, readLine...))
		original = make([]byte, 0)

		split := strings.Split(content, "=")
		if len(split) < 2 {
			continue
		}

		newSingle(split[0], strings.Split(split[1], ","))
	}
}

func newSingle(addr string, uri []string) {
	target, err := url.Parse(addr)
	if err != nil {
		log.Fatal(err)
	}

	// tls ja3
	if strings.HasPrefix(addr, "tls:") {
		addr = addr[4:]
		proxy := NewTlsProxy(addr)
		for _, it := range uri {
			POOL[strings.TrimSpace(it)] = proxy
		}

		log.Printf("create new Single: [ %s ] - %s\n", addr, "[ "+strings.Join(uri, ", ")+" ]")
		return
	}

	// default
	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.Director = func(req *http.Request) {
		req.Host = target.Host
		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host
	}

	if pu != nil {
		proxy.Transport = &http.Transport{
			Proxy: http.ProxyURL(pu),
		}
	}

	for _, it := range uri {
		POOL[strings.TrimSpace(it)] = &EasyProxy{
			path:         addr,
			ReverseProxy: proxy,
		}
	}

	log.Printf("create new Single: [ %s ] - %s\n", addr, "[ "+strings.Join(uri, ", ")+" ]")
}

func LoadEnvVar(key, defaultValue string) string {
	value, exists := os.LookupEnv(key)
	if !exists {
		value = defaultValue
	}
	return value
}

func LoadEnvInt(key string, defaultValue int) int {
	value, exists := os.LookupEnv(key)
	if !exists {
		return defaultValue
	}

	i, err := strconv.Atoi(value)
	if err != nil {
		log.Printf("%v\n", err)
		return defaultValue
	}

	return i
}
