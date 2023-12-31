package main

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
	"regexp"
	"strconv"
	"strings"
)

var (
	pu       *url.URL = nil
	port              = 8080
	proxyMap          = make(map[string]*SingleProxy)
	VERSION           = "v1.0.0"
)

type SingleProxy struct {
	path string
	*httputil.ReverseProxy
}

func init() {
	_ = godotenv.Load()
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	port = LoadEnvInt("PORT", port)
	p := LoadEnvVar("PROXY", "")
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
		log.Printf("%v\n", err)
		os.Exit(-1)
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
		log.Fatal(err)
	}

	return i
}

func newSingle(addr string, uri []string) {
	target, err := url.Parse(addr)
	if err != nil {
		log.Fatal(err)
	}

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
		proxyMap[strings.TrimSpace(it)] = &SingleProxy{
			path:         addr,
			ReverseProxy: proxy,
		}
	}

	log.Printf("create new Single: [ %s ] - %s\n", addr, "[ "+strings.Join(uri, ", ")+" ]")
}

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {

		b, err := httputil.DumpRequest(r, true)
		if err != nil {
			log.Printf("%v\n", err)
			return
		} else {
			log.Printf("%s\n", string(b))
		}

		args := ""
		uri := r.RequestURI
		if index := strings.Index(uri, "?"); index >= 0 {
			uri = r.RequestURI[:index]
			args = r.RequestURI[index:]
		}
		if uri == "" || uri == "/" {
			index(r.Host, w)
			return
		}

		log.Printf("proxy uri: %s, args: %s\n", uri, args)

		if proxy, ok := proxyMap[uri]; ok {
			proxy.ServeHTTP(w, r)
			return
		}

		var routeAll *SingleProxy
		for k, proxy := range proxyMap {
			if strings.HasPrefix(k, "reg:") {
				compile := regexp.MustCompile(k[4:])
				if compile.MatchString(uri) {
					log.Printf("proxy target: %v\n", proxy.path)
					proxy.ServeHTTP(w, r)
					return
				}
			} else if strings.HasPrefix(uri, k) {
				log.Printf("proxy target: %v\n", proxy.path)
				proxy.ServeHTTP(w, r)
				return
			} else if k == "*" {
				routeAll = proxy
			}
		}

		if routeAll != nil {
			log.Printf("proxy target * : %v\n", routeAll.path)
			routeAll.ServeHTTP(w, r)
		}
	})

	log.Printf("Starting server on port %d\n", port)
	if err := http.ListenAndServe(":"+strconv.Itoa(port), nil); err != nil {
		log.Fatal(err)
	}
}

func index(host string, w http.ResponseWriter) {
	_, err := w.Write([]byte("Start by http[s]://" + host + "/v1\n\nversion: " + VERSION + "\nproject: https://github.com/bincooo/single-proxy"))
	if err != nil {
		log.Printf("%v\n", err)
	}
}
