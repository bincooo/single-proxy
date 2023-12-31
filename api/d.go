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
	"regexp"
	"strconv"
	"strings"
)

var (
	pu      *url.URL = nil
	pMapper          = make(map[string]*SingleProxy)
	PORT             = 8080
	VERSION          = "v1.0.0"
)

type SingleProxy struct {
	path string
	*httputil.ReverseProxy
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
		pMapper[strings.TrimSpace(it)] = &SingleProxy{
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
		log.Fatal(err)
	}

	return i
}

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

	var routeAll *SingleProxy
	var prefix *SingleProxy
	for k, proxy := range pMapper {
		if strings.HasPrefix(k, "reg:") {
			compile := regexp.MustCompile(k[4:])
			if compile.MatchString(uri) {
				log.Printf("proxy target: %v\n\n\n", proxy.path)
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
		log.Printf("proxy target * : %v\n\n\n", routeAll.path)
		prefix.ServeHTTP(w, r)
		return
	}

	if routeAll != nil {
		log.Printf("proxy target * : %v\n\n\n", routeAll.path)
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
