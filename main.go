package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
	"urlShortener/channels"

	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/go-redis/redis"
	"github.com/lithammer/shortuuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Mapper struct {
	Mapping map[string]string
	Lock    sync.Mutex
}
type URLAnalytics struct {
	URL        string `bson:"url"`
	ShortURL   string `bson:"short_url"`
	ClickCount int    `bson:"click_count"`
}

var urlMapper Mapper
var redisClient *redis.Client
var mongoClient *mongo.Client
var analyticsCollection *mongo.Collection

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
	//Connect to RMQ
	err = channels.QueueConnect("ANALYTICS_QUEUE", true)
	if err != nil {
		log.Fatal("Error in creating Analytics Queue - ", err)
	}
	//initialize mapper
	urlMapper = Mapper{
		Mapping: make(map[string]string),
	}
	//Conneting to mongodb
	mongoClient, err = mongo.Connect(context.TODO(), options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		log.Fatal("Error in connecting to mongodb - ", err)
	}
	// Get the analytics collection
	analyticsCollection = mongoClient.Database("urlshortener").Collection("analytics")
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

	// Increment click count in MongoDB
	filter := bson.M{"short_url": fmt.Sprintf("http://localhost:3000/short/%s", key)}
	update := bson.M{"$inc": bson.M{"click_count": 1}}
	_, err := analyticsCollection.UpdateOne(context.TODO(), filter, update)
	if err != nil {
		log.Println("Error updating click count:", err)
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

	//Add to Analytics
	go analytics(key, u)

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

func analytics(key, u string) {
	var result URLAnalytics
	//Get the value of click count from mongodb
	filter := bson.M{"short_url": fmt.Sprintf("http://localhost:3000/short/%s", key)}
	err := analyticsCollection.FindOne(context.TODO(), filter).Decode(&result)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			// Handle the case where no document was found
			log.Println("No document found with the given short URL")
			return
		}
		// Handle other errors
		log.Println("Error finding document:", err)
		return
	}

	analyticsData := URLAnalytics{
		URL:        u,
		ShortURL:   fmt.Sprintf("http://localhost:3000/short/%s", key),
		ClickCount: result.ClickCount,
	}
	_, err = analyticsCollection.InsertOne(context.TODO(), analyticsData)
	if err != nil {
		log.Panic("Unable to insert analytics data to MongoDB:", err)
	}

	analyticsDataByte, err := json.Marshal(analyticsData)
	if err != nil {
		log.Panic("Unable to marshal analytics data - ", err)
	}
	err = channels.Publisher("ANALYTICS_QUEUE", analyticsDataByte)
	if err != nil {
		log.Panic("Unable to publish to RMQ - ", err)
	}
}
