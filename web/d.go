package web

import (
	"github.com/single-proxy/api"
	"log"
	"math/rand"
	"net/http"
	"net/http/httputil"
	"os"
	"regexp"
	"strconv"
	"strings"
	"text/template"
	"time"
)

func Handler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "" || r.URL.Path == "/" {
		index(r.Host, w)
		return
	}

	data, err := httputil.DumpRequest(r, false)
	if err != nil {
		log.Printf("Error: %v\n", err)
		return
	} else {
		log.Printf("%s\n", data)
	}

	uri := r.URL.Path
	log.Printf("proxy uri: %s, args: %s\n", uri, r.URL.RawQuery)

	if mapper, ok := api.ProxiesMapper[uri]; ok {
		log.Printf("proxy target: %v\n\n\n", mapper.Path())
		route := mapper.Route()
		if route.Rewrite != "" {
			rewriteRoute(r, route)
		}
		if len(route.Action) > 0 {
			if err = execAction(r, w, route); err != nil {
				log.Printf("Error: %v\n", err)
				return
			}
		}
		mapper.ServeHTTP(w, r)
		return
	}

	var prefix api.Proxies
	var routeAll api.Proxies

	for _, mapper := range api.ProxiesMapper {
		route := mapper.Route()
		if strings.HasPrefix(uri, route.Path) {
			prefix = mapper
			break
		}

		if route.Path == "*" {
			routeAll = mapper
			break
		}
	}

	if prefix != nil {
		log.Printf("proxy target * : %v\n\n\n", prefix.Path())
		route := prefix.Route()
		if route.Rewrite != "" {
			rewriteRoute(r, route)
		}
		if len(route.Action) > 0 {
			if err = execAction(r, w, route); err != nil {
				log.Printf("Error: %v\n", err)
				return
			}
		}
		prefix.ServeHTTP(w, r)
		return
	}

	if routeAll != nil {
		log.Printf("proxy target * : %v\n\n\n", routeAll.Path())
		route := routeAll.Route()
		if route.Rewrite != "" {
			rewriteRoute(r, route)
		}
		if err = execAction(r, w, route); err != nil {
			log.Printf("Error: %v\n", err)
			return
		}
		routeAll.ServeHTTP(w, r)
		return
	}

	for _, mapper := range api.ProxiesMapper {
		route := mapper.Route()
		compile := regexp.MustCompile(route.Path)
		if compile.MatchString(uri) {
			log.Printf("proxy target: %v\n\n\n", mapper.Path())
			if route.Rewrite != "" {
				rewriteRoute(r, route)
			}
			if len(route.Action) > 0 {
				if err = execAction(r, w, route); err != nil {
					log.Printf("Error: %v\n", err)
					return
				}
			}
			mapper.ServeHTTP(w, r)
			return
		}
	}

	// 没有匹配到地址
	w.WriteHeader(http.StatusNotFound)
	log.Printf("proxy not found: %v\n", uri)
}

func index(host string, w http.ResponseWriter) {
	_, err := w.Write([]byte("Start by http[s]://" + host + "\n\nversion: " + api.VERSION + "\nproject: https://github.com/bincooo/single-proxy"))
	if err != nil {
		log.Printf("%v\n", err)
	}
}

func rewriteRoute(r *http.Request, route api.Route) {
	c := regexp.MustCompile(route.Path)
	r.URL.Path = c.ReplaceAllString(r.URL.Path, route.Rewrite)
	r.RequestURI = r.URL.RequestURI()
	log.Printf("rewrite route '%s' to '%s'", route.Path, r.URL.Path)
}

func execAction(req *http.Request, w http.ResponseWriter, route api.Route) error {
	t := template.New("context")
	set := func(set func(k, v string)) func(key string, value string) string {
		return func(k string, v string) string {
			set(k, v)
			return ""
		}
	}
	del := func(del func(k string)) func(key string) string {
		return func(k string) string {
			del(k)
			return ""
		}
	}

	funcMap := template.FuncMap{
		"req_setHeader": set(req.Header.Set),
		"req_delHeader": del(req.Header.Del),
		"req_getHeader": req.Header.Get,
		"res_setHeader": set(w.Header().Set),
		"res_delHeader": del(w.Header().Del),
		"res_getHeader": w.Header().Get,
		"split":         strings.Split,
		"contains":      strings.Contains,
		"randomIp":      randomIp,
		"append": func(v1, v2 string) string {
			return v1 + v2
		},
		"addr": func(addr string) string {
			if addr != "" {
				req.RemoteAddr = addr + ":17890"
			}
			return ""
		},
	}
	t.Funcs(funcMap)
	for _, tmplVar := range route.Action {
		tmpl, err := t.Parse(tmplVar)
		if err != nil {
			return err
		}
		if err = tmpl.Execute(os.Stdout, nil); err != nil {
			return err
		}
	}
	return nil
}

// 获取随机ip
func randomIp() string {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	ip2Int := func(ip string) int {
		slice := strings.Split(ip, ".")
		result := 0
		atoi, _ := strconv.Atoi(slice[0])
		result += atoi << 24
		atoi, _ = strconv.Atoi(slice[1])
		result += atoi << 16
		atoi, _ = strconv.Atoi(slice[2])
		result += atoi << 8
		atoi, _ = strconv.Atoi(slice[2])
		result += atoi
		return result
	}

	int2Ip := func(num int) (result string) {
		result += strconv.Itoa(num>>24&255) + "."
		result += strconv.Itoa(num>>16&255) + "."
		result += strconv.Itoa(num>>8&255) + "."
		result += strconv.Itoa(num & 255)
		return
	}

	randIndex := r.Intn(len(IP_RANGE))
	startIPInt := ip2Int(IP_RANGE[randIndex][0])
	endIPInt := ip2Int(IP_RANGE[randIndex][1])

	newIpInt := r.Intn(endIPInt-startIPInt) + startIPInt
	return int2Ip(newIpInt)
}

var IP_RANGE = [][]string{
	{"4.150.64.0", "4.150.127.255"},      // Azure Cloud EastUS2 16382
	{"4.152.0.0", "4.153.255.255"},       // Azure Cloud EastUS2 131070
	{"13.68.0.0", "13.68.127.255"},       // Azure Cloud EastUS2 32766
	{"13.104.216.0", "13.104.216.255"},   // Azure EastUS2 256
	{"20.1.128.0", "20.1.255.255"},       // Azure Cloud EastUS2 32766
	{"20.7.0.0", "20.7.255.255"},         // Azure Cloud EastUS2 65534
	{"20.22.0.0", "20.22.255.255"},       // Azure Cloud EastUS2 65534
	{"40.84.0.0", "40.84.127.255"},       // Azure Cloud EastUS2 32766
	{"40.123.0.0", "40.123.127.255"},     // Azure Cloud EastUS2 32766
	{"4.214.0.0", "4.215.255.255"},       // Azure Cloud JapanEast 131070
	{"4.241.0.0", "4.241.255.255"},       // Azure Cloud JapanEast 65534
	{"40.115.128.0", "40.115.255.255"},   // Azure Cloud JapanEast 32766
	{"52.140.192.0", "52.140.255.255"},   // Azure Cloud JapanEast 16382
	{"104.41.160.0", "104.41.191.255"},   // Azure Cloud JapanEast 8190
	{"138.91.0.0", "138.91.15.255"},      // Azure Cloud JapanEast 4094
	{"151.206.65.0", "151.206.79.255"},   // Azure Cloud JapanEast 256
	{"191.237.240.0", "191.237.241.255"}, // Azure Cloud JapanEast 512
	{"4.208.0.0", "4.209.255.255"},       // Azure Cloud NorthEurope 131070
	{"52.169.0.0", "52.169.255.255"},     // Azure Cloud NorthEurope 65534
	{"68.219.0.0", "68.219.127.255"},     // Azure Cloud NorthEurope 32766
	{"65.52.64.0", "65.52.79.255"},       // Azure Cloud NorthEurope 4094
	{"98.71.0.0", "98.71.127.255"},       // Azure Cloud NorthEurope 32766
	{"74.234.0.0", "74.234.127.255"},     // Azure Cloud NorthEurope 32766
	{"4.151.0.0", "4.151.255.255"},       // Azure Cloud SouthCentralUS 65534
	{"13.84.0.0", "13.85.255.255"},       // Azure Cloud SouthCentralUS 131070
	{"4.255.128.0", "4.255.255.255"},     // Azure Cloud WestCentralUS 32766
	{"13.78.128.0", "13.78.255.255"},     // Azure Cloud WestCentralUS 32766
	{"4.175.0.0", "4.175.255.255"},       // Azure Cloud WestEurope 65534
	{"13.80.0.0", "13.81.255.255"},       // Azure Cloud WestEurope 131070
	{"20.73.0.0", "20.73.255.255"},       // Azure Cloud WestEurope 65534
}
