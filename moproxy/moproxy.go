package main

import (
	"flag"
	"fmt"
	lumber "github.com/jcelliott/lumber"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
)

var log lumber.Logger
var port string
var origins []url.URL
var httpClient http.Client

func usage() {
	fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
	flag.PrintDefaults()
	fmt.Fprintf(os.Stderr, "  non-flag arguments: multiple origin base URLs for proxy requests\n")
}

func main() {
	flag.Usage = usage
	logLevelArg := flag.String("loglevel", "INFO", "Logging level (FATAL, ERROR, WARN, INFO, DEBUG, TRACE)")
	flag.StringVar(&port, "port", "8123", "Port to listen on")
	flag.Parse()

	logLevel := lumber.LvlInt(*logLevelArg)
	log = lumber.NewConsoleLogger(logLevel)

	origins = toUrls(flag.Args())

	httpClient = http.Client{}

	http.HandleFunc("/", requestHandler)
	http.ListenAndServe(fmt.Sprintf(":%s", port), nil)
}

func toUrls(urlvals []string) []url.URL {
	urls := make([]url.URL, len(urlvals))
	for i, urlval := range urlvals {
		url, err := url.ParseRequestURI(urlval)
		if err == nil {
			urls[i] = *url
		}
	}
	return urls
}

func requestHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/" {
		http.NotFound(w, r)
		return
	}
	for _, origin := range origins {

		u := url.URL{}
		u.Scheme = origin.Scheme
		u.Host = origin.Host
		u.Path = origin.Path + r.URL.Path
		originUrl := u.String()

		log.Debug("Requesting %s", originUrl)
		req, _ := http.NewRequest("GET", originUrl, nil)
		req.Close = true
		resp, err := httpClient.Do(req)
		if resp != nil {
			defer resp.Body.Close()
		}
		if err == nil && resp != nil && resp.StatusCode == 200 {
			contentLength := strconv.FormatInt(resp.ContentLength, 10)
			log.Info("Serving %v (%v bytes)", originUrl, contentLength)
			w.Header().Set("Content-Length", contentLength)
			w.Header().Set("Content-Location", originUrl)
			io.Copy(w, resp.Body)
			return
		}
	}
	log.Info("Not found: %v", r.URL.Path)
	http.NotFound(w, r)
}
