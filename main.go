package main

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/lithammer/shortuuid"
)

type Mapper struct {
	Mapping map[string]string
	Lock    sync.Mutex
}

var urlMapper Mapper

func init() {
	//initialize mapper
	urlMapper = Mapper{
		Mapping: make(map[string]string),
	}
}

func main() {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Server is running"))
	})
	r.Get("/short/{key}", redirectHandler)
	r.Post("/short-url", createShortURLHandler)
	http.ListenAndServe(":3000", r)
}

func createShortURLHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	url := r.Form.Get("URL")
	if url == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("URL field is empty"))
		return
	}
	//generate key
	key := shortuuid.New()
	//insert mapping
	insertMapping(key, url)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf("http://localhost:3000/short/%s", key)))
}

func redirectHandler(w http.ResponseWriter, r *http.Request) {
	key := chi.URLParam(r, "key")
	if key == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("URL field is empty"))
		return
	}

	//fetch mapping
	url := fetchMapping(key)
	if url == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("URL does not exists"))
		return
	}
	http.Redirect(w, r, url, http.StatusFound)
}

func insertMapping(key, u string) {
	urlMapper.Lock.Lock()
	defer urlMapper.Lock.Unlock()

	urlMapper.Mapping[key] = u
}

func fetchMapping(key string) string {
	urlMapper.Lock.Lock()
	defer urlMapper.Lock.Unlock()

	return urlMapper.Mapping[key]
}
