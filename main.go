package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/davidmz/mustbe"
)

type App struct {
	Listen     string
	FRFDomains []string
}

type H map[string]interface{}

var (
	HeadersFromBackend = []string{"Content-Type", "Etag", "Last-Modified"}
	HeadersFromClient  = []string{"Content-Type", "If-None-Match", "If-Modified-Since"}
)

func main() {
	defer mustbe.Catched(func(err error) {
		log.Println("Fatal error:", err)
		os.Exit(1)
	})

	mustbeTrue(len(os.Args) >= 2, "Usage: frf-aprox conf.json")

	app := new(App)
	mustbe.OK(app.Load(os.Args[1]))

	s := &http.Server{
		Addr:           app.Listen,
		Handler:        http.HandlerFunc(app.Handler),
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	log.Printf("Starting server at %s\n", app.Listen)
	mustbe.OK(s.ListenAndServe())
}

func (a *App) Load(fileName string) error {
	file, err := os.Open(fileName)
	if err != nil {
		return err
	}
	defer file.Close()
	return json.NewDecoder(file).Decode(a)
}

func (a *App) Handler(w http.ResponseWriter, r *http.Request) {
	defer mustbe.Catched(func(err error) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(H{"err": err.Error(), "from": "aprx"})
	})

	// request path: /frf-domain/api-path
	pp := strings.SplitN(r.URL.Path, "/", 3)[1:]
	mustbeTrue(len(pp) == 2, "Invalid request path")
	domain, path := pp[0], "/"+pp[1]

	domainFound := false
	for _, d := range a.FRFDomains {
		if d == domain {
			domainFound = true
			break
		}
	}
	mustbeTrue(domainFound, "Invalid frf domain")

	var (
		accessToken    string
		frfRequestBody []byte
	)

	switch r.Header.Get("Content-Type") {
	case "application/x-www-form-urlencoded":
		r.ParseForm()
		accessToken = r.PostForm.Get("accessToken")
		r.PostForm.Del("accessToken")
		frfRequestBody = []byte(r.PostForm.Encode())

	case "application/json", "application/json; charset=utf-8":
		h := make(H)
		mustbe.OK(json.NewDecoder(r.Body).Decode(h))
		accessToken = h["accessToken"].(string)
		delete(h, "accessToken")
		frfRequestBody = mustbe.OKVal(json.Marshal(h)).([]byte)
	}

	req, err := http.NewRequest(
		r.Method,
		"https://"+domain+path,
		bytes.NewReader(frfRequestBody),
	)
	mustbe.OK(err)

	for _, h := range HeadersFromClient {
		if hh, ok := r.Header[h]; ok {
			req.Header[h] = hh
		}
	}

	if accessToken != "" {
		req.Header.Set("X-Authentication-Token", accessToken)
	}

	resp, err := http.DefaultClient.Do(req)
	mustbe.OK(err)

	defer resp.Body.Close()

	for _, h := range HeadersFromBackend {
		if hh, ok := resp.Header[h]; ok {
			w.Header()[h] = hh
		}
	}

	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func mustbeTrue(test bool, errMsg string) {
	if !test {
		mustbe.OK(errors.New(errMsg))
	}
}
