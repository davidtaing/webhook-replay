package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"

	log "github.com/sirupsen/logrus"
)

type RequestBody struct{}

func main() {
	var (
		dest string
		port int
	)

	flag.StringVar(&dest, "dest", "", "")
	flag.IntVar(&port, "port", 0, "")
	flag.Parse()

	if dest == "" || port == 0 {
		log.Fatal("Missing required flags: --dest and --port")
	}

	log.SetFormatter(&log.JSONFormatter{})
	log.SetOutput(os.Stdout)
	log.SetLevel(log.InfoLevel)

	target, _ := url.Parse(dest)
	proxy := httputil.NewSingleHostReverseProxy(target)

	proxy.Director = func(r *http.Request) {
		r.URL.Scheme = target.Scheme
		r.URL.Host = target.Host
		r.Host = target.Host
	}

	proxy.ModifyResponse = func(r *http.Response) error {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			log.Errorf("Error reading response body: %v", err)
			return err
		}

		r.Body = io.NopCloser(bytes.NewReader(body))

		log.WithFields(log.Fields{
			"body":   string(body),
			"status": r.Status,
		}).Infoln()

		return nil
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			log.Errorf("Error reading request body: %v", err)
			http.Error(w, "Error reading request body", http.StatusInternalServerError)
			return
		}

		r.Body = io.NopCloser(bytes.NewReader(body))

		log.WithFields(log.Fields{
			"body":   string(body),
			"method": r.Method,
			"url":    target.ResolveReference(r.URL).String(),
		}).Infoln()

		proxy.ServeHTTP(w, r)
	})

	log.WithFields(log.Fields{
		"port": port,
		"dest": dest,
	}).Info("Starting server")

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))
}
