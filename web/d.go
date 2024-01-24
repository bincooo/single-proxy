package web

import (
	"github.com/single-proxy/api"
	"html/template"
	"log"
	"net/http"
	"net/http/httputil"
	"os"
	"regexp"
	"strings"
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

	var rou api.Route
	var prefix api.Proxies
	var routeAll api.Proxies

	if mapper, ok := api.ProxiesMapper[uri]; ok {
		log.Printf("proxy target: %v\n\n\n", mapper.Path())
		route := mapper.Route()
		if route.Rewrite != "" {
			rewriteRoute(r, route)
		}
		if len(rou.Action) > 0 {
			if err = execAction(r, w, route); err != nil {
				log.Printf("Error: %v\n", err)
				return
			}
		}
		mapper.ServeHTTP(w, r)
		return
	}

	for _, mapper := range api.ProxiesMapper {
		route := mapper.Route()
		if strings.HasPrefix(uri, route.Path) {
			prefix = mapper
			rou = route
			break
		}

		if route.Path == "*" {
			routeAll = mapper
			rou = route
			break
		}
	}

	if prefix != nil {
		log.Printf("proxy target * : %v\n\n\n", prefix.Path())
		if rou.Rewrite != "" {
			rewriteRoute(r, rou)
		}
		if len(rou.Action) > 0 {
			if err = execAction(r, w, rou); err != nil {
				log.Printf("Error: %v\n", err)
				return
			}
		}
		prefix.ServeHTTP(w, r)
		return
	}

	if routeAll != nil {
		log.Printf("proxy target * : %v\n\n\n", routeAll.Path())
		if rou.Rewrite != "" {
			rewriteRoute(r, rou)
		}
		if err = execAction(r, w, rou); err != nil {
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
	appen := func(v1, v2 string) string {
		return v1 + v2
	}
	funcMap := template.FuncMap{
		"rSet":     set(req.Header.Set),
		"rDel":     del(req.Header.Del),
		"rGet":     req.Header.Get,
		"wSet":     set(w.Header().Set),
		"wDel":     del(w.Header().Del),
		"wGet":     w.Header().Get,
		"split":    strings.Split,
		"contains": strings.Contains,
		"append":   appen,
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
