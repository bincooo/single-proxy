package api

import (
	"bytes"
	"crypto/tls"
	"github.com/joho/godotenv"
	"github.com/spf13/viper"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
)

const VERSION = "v1.0.0"

var (
	pu            *url.URL = nil
	ProxiesMapper          = make(map[string]Proxies)
	PORT                   = 8080
)

type Proxies interface {
	ServeHTTP(rw http.ResponseWriter, req *http.Request)
	Path() string
	Route() Route
}

type EasyProxies struct {
	path string
	*httputil.ReverseProxy
	route Route
}

func (e EasyProxies) Path() string {
	return e.path
}

func (e EasyProxies) Route() Route {
	return e.route
}

func init() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	vip := loadConfig()
	PORT = vip.GetInt("port")
	proxies = vip.GetString("proxies")
	JA3 = vip.GetString("ja3")
	timeout = vip.GetInt("timeout")

	if PORT == 0 {
		PORT = 8080
	}

	if proxies != "" {
		u, err := url.Parse(proxies)
		if err != nil {
			log.Fatal(err)
		}
		pu = u
	}

	var mappers []Mapper
	if err := vip.UnmarshalKey("mappers", &mappers); err != nil {
		log.Fatal(err)
	}

	for _, mapper := range mappers {
		newSingle(mapper)
	}
}

func loadConfig() *viper.Viper {
	_ = godotenv.Load()
	config := LoadEnvVar("CONFIG", "")
	data, err := os.ReadFile("config.yaml")
	if err != nil {
		if config == "" {
			log.Fatal(err)
		}

		var response *http.Response
		response, err = http.DefaultClient.Get(config)
		if err != nil {
			log.Fatal(err)
		}
		data, err = io.ReadAll(response.Body)
		if err != nil {
			log.Fatal(err)
		}
		if response.StatusCode != 200 {
			log.Printf("Error: %s\n", data)
			os.Exit(-1)
		}
	}

	vip := viper.New()
	vip.SetConfigType("yaml")
	if err = vip.ReadConfig(bytes.NewReader(data)); err != nil {
		log.Fatal(err)
	}

	return vip
}

func newSingle(mapper Mapper) {
	target, err := url.Parse(mapper.Addr)
	if err != nil {
		log.Fatal(err)
	}

	// tls ja3
	if mapper.Ja3 {
		paths := make([]string, 0)
		for _, route := range mapper.Routes {
			ProxiesMapper[route.Path] = newJa3Proxies(mapper.Addr, route)
			paths = append(paths, route.Path)
		}
		log.Printf("create new Single: %s - %s\n", mapper.Addr, "[ "+strings.Join(paths, ", ")+" ]")
		return
	}

	// default
	defaultProxies := httputil.NewSingleHostReverseProxy(target)
	defaultProxies.Director = func(req *http.Request) {
		req.Host = target.Host
		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host
	}

	if pu != nil {
		defaultProxies.Transport = &http.Transport{
			Proxy:           http.ProxyURL(pu),
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	} else {
		defaultProxies.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	}

	paths := make([]string, 0)
	for _, route := range mapper.Routes {
		paths = append(paths, route.Path)
		ProxiesMapper[route.Path] = &EasyProxies{mapper.Addr, defaultProxies, route}
	}

	log.Printf("create new Single: %s - %s\n", mapper.Addr, "[ "+strings.Join(paths, ", ")+" ]")
}

func LoadEnvVar(key, defaultValue string) string {
	value, exists := os.LookupEnv(key)
	if !exists {
		value = defaultValue
	}
	return value
}
