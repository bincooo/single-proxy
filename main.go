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
	proxyMap          = make(map[string]*httputil.ReverseProxy)
)

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
		proxyMap[strings.TrimSpace(it)] = proxy
	}
}

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		b, err := httputil.DumpRequest(r, true)
		if err != nil {
			log.Printf("%v\n", err)
		} else {
			log.Printf("%s\n", string(b))
		}

		uri := r.RequestURI
		if proxy, ok := proxyMap[uri]; ok {
			proxy.ServeHTTP(w, r)
			return
		}

		for k, proxy := range proxyMap {
			if strings.HasPrefix(k, "reg:") {
				compile := regexp.MustCompile(k[4:])
				if compile.MatchString(uri) {
					proxy.ServeHTTP(w, r)
					return
				}
			} else if strings.HasPrefix(uri, k) {
				proxy.ServeHTTP(w, r)
			}
		}
	})

	log.Printf("Starting server on port %d\n", port)
	if err := http.ListenAndServe(":"+strconv.Itoa(port), nil); err != nil {
		log.Fatal(err)
	}
}
