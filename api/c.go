package api

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
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
	proxiesFetch  string
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
	proxiesFetch = vip.GetString("proxies-pool")
	if PORT == 0 {
		PORT = 8080
	}

	if proxies != "" {
		log.Printf("golbal proxies: %s", proxies)
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

	var sp string
	var spPu *url.URL
	if mapper.Proxies != "" {
		// log.Printf("single proxies: %s", proxies)
		u, perr := url.Parse(mapper.Proxies)
		if perr != nil {
			log.Fatal(perr)
		}

		sp = mapper.Proxies
		spPu = u
	}

	// tls ja3
	if mapper.Ja3 {
		paths := make([]string, 0)
		for _, route := range mapper.Routes {
			ProxiesMapper[route.Path] = newJa3Proxies(mapper.Addr, route, sp)
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
		req.RemoteAddr = ""
	}

	if sp == "auto" {
		withProxies, ferr := fetchPoolWithProxies()
		if ferr != nil {
			log.Printf("%v", ferr)
		} else {
			u, _ := url.Parse(withProxies)
			defaultProxies.Transport = &http.Transport{
				Proxy:           http.ProxyURL(u),
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			}
			goto label
		}
	}

	if spPu != nil {
		defaultProxies.Transport = &http.Transport{
			Proxy:           http.ProxyURL(spPu),
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	} else if pu != nil {
		defaultProxies.Transport = &http.Transport{
			Proxy:           http.ProxyURL(pu),
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	} else {
		defaultProxies.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	}
label:

	paths := make([]string, 0)
	for _, route := range mapper.Routes {
		paths = append(paths, route.Path)
		ProxiesMapper[route.Path] = &EasyProxies{mapper.Addr, defaultProxies, route}
	}

	log.Printf("create new Single: %s - %s\n", mapper.Addr, "[ "+strings.Join(paths, ", ")+" ]")
}

func fetchPoolWithProxies() (string, error) {
	if proxiesFetch == "" {
		return "", errors.New(fmt.Sprintf("fetch proxies error: `proxiesFetch` is empty"))
	}
	response, err := http.Get(proxiesFetch)
	if err != nil {
		return "", errors.New(fmt.Sprintf("fetch proxies error: %v", err))
	}

	if response.StatusCode != http.StatusOK {
		return "", errors.New(fmt.Sprintf("fetch proxies error: %s", response.Status))
	}

	data, err := io.ReadAll(response.Body)
	if err != nil {
		return "", errors.New(fmt.Sprintf("fetch proxies error: %v", err))
	}

	log.Printf("fetch proxies success: \n %s", data)

	dict := make(map[string]interface{})
	err = json.Unmarshal(data, &dict)
	if err != nil {
		return "", err
	}

	tempError := errors.New(fmt.Sprintf("fetch proxies [%s] is nil result", proxiesFetch))
	if prox, ok := dict["proxy"]; ok {
		if prox == "" {
			return "", tempError
		}

		if https, _ok := dict["https"].(bool); _ok && https {
			return fmt.Sprintf("https://%s", prox), nil
		} else {
			return fmt.Sprintf("http://%s", prox), nil
		}
	}

	return "", tempError
}

func LoadEnvVar(key, defaultValue string) string {
	value, exists := os.LookupEnv(key)
	if !exists {
		value = defaultValue
	}
	return value
}
