package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"time"
)

// Route defines a reverse proxy route.
type Route struct {
	Prefix string
	Target string
}

var routes = []Route{
	{"/dashboard/", "http://141.136.47.94:3003"},
	{"/openclaw/", "http://141.136.47.94:18789"},
	{"/n8n/", "http://141.136.47.94:5678"},
	{"/ollama/", "http://148.230.100.223:11434"},
	{"/comfyui/", "http://148.230.100.223:8188"},
	{"/api/", "http://148.230.100.223:3081"},
	{"/reels/", "http://148.230.100.223:7880"},
	{"/manametamaori/", "http://148.230.100.223:3080"},
}

func newProxy(target string, prefix string) http.Handler {
	u, err := url.Parse(target)
	if err != nil {
		log.Fatalf("invalid proxy target %s: %v", target, err)
	}
	proxy := httputil.NewSingleHostReverseProxy(u)
	proxy.ErrorLog = log.New(os.Stderr, "[proxy] ", log.LstdFlags)
	proxy.Transport = &http.Transport{
		ResponseHeaderTimeout: 30 * time.Second,
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.URL.Path = strings.TrimPrefix(r.URL.Path, strings.TrimSuffix(prefix, "/"))
		if r.URL.Path == "" {
			r.URL.Path = "/"
		}
		r.Host = u.Host
		proxy.ServeHTTP(w, r)
	})
}

func main() {
	port := os.Getenv("MANA_PROXY_PORT")
	if port == "" {
		port = "8081"
	}

	mux := http.NewServeMux()

	for _, route := range routes {
		log.Printf("proxy: %s -> %s", route.Prefix, route.Target)
		mux.Handle(route.Prefix, newProxy(route.Target, route.Prefix))
	}

	// Root handler — service index
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<!DOCTYPE html><html><head><title>Mana Proxy</title>
<style>body{font-family:monospace;background:#0a0a0a;color:#0f0;padding:2em}
a{color:#0ff;text-decoration:none}a:hover{text-decoration:underline}
h1{color:#fff}li{margin:0.5em 0}</style></head>
<body><h1>Mana Node — Service Proxy</h1><ul>`))
		for _, r := range routes {
			w.Write([]byte(`<li><a href="` + r.Prefix + `">` + r.Prefix + `</a> → ` + r.Target + `</li>`))
		}
		w.Write([]byte(`</ul></body></html>`))
	})

	log.Printf("mana-proxy listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}
