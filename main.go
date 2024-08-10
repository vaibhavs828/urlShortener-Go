package main

import (
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/go-redis/redis"
	"github.com/lithammer/shortuuid"
)

type Mapper struct {
	Mapping map[string]string
	Lock    sync.Mutex
}

var urlMapper Mapper
var redisClient *redis.Client

func init() {
	// Create a new client
	redisClient = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379", // Use default Addr
		Password: "",               // No password set
		DB:       0,                // Use default DB
	})

	// Ping the Redis server
	pong, err := redisClient.Ping().Result()
	if err != nil {
		panic(err)
	}
	fmt.Println(pong)
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

	//Add a route for a custom made url
	r.Post("/custom-short-url", createCustomShortURLHanler)

	http.ListenAndServe(":3000", r)
}

func createShortURLHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	url := r.Form.Get("url")
	if url == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("URL field is empty"))
		return
	}
	//check if key already present
	if redisClient.Exists(url).Val() == 1 {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(fmt.Sprintf("http://localhost:3000/short/%s", redisClient.Get(url).Val())))
		return
	}
	//generate key
	key := shortuuid.New()
	//insert mapping
	insertMapping(key, url)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf("http://localhost:3000/short/%s", key)))
}

func createCustomShortURLHanler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	url := r.Form.Get("url")
	if url == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("URL field is empty"))
		return
	}
	customPath := r.Form.Get("custom_url")
	if url == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("custom_path field is empty"))
		return
	}
	// Check if customPath is already used
	if redisClient.Exists(customPath).Val() == 1 || urlMapper.Mapping[customPath] != "" {
		w.WriteHeader(http.StatusConflict)
		w.Write([]byte("Custom path already in use"))
		return
	}

	// Insert custom mapping
	insertMapping(customPath, url)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf("http://localhost:3000/short/%s", customPath)))
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
	//Storing in cache
	log.Println("Storing the url in cache")
	redisClient.Set(u, key, 24*time.Hour)
}

func fetchMapping(key string) string {
	urlMapper.Lock.Lock()
	defer urlMapper.Lock.Unlock()
	if redisClient.Exists(key).Val() == 1 {
		log.Println("Fetched the url from cache")
		return redisClient.Get(key).Val()
	}

	return urlMapper.Mapping[key]
}
