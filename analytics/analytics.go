package analytics

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/streadway/amqp"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

var mongoClient *mongo.Client
var analyticsCollection *mongo.Collection
var err error

func init() {
	//Conneting to mongodb
	mongoClient, err = mongo.Connect(context.TODO(), options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		log.Fatal("Error in connecting to mongodb - ", err)
	}

	// Ping MongoDB to ensure connection is established
	err = mongoClient.Ping(context.TODO(), readpref.Primary())
	if err != nil {
		log.Fatal("Error connecting to MongoDB:", err)
	}
	fmt.Println("Connected to MongoDB")

	// Get the analytics collection
	analyticsCollection = mongoClient.Database("urlshortener").Collection("analytics")
}

type AnalyticsPayload struct {
	URL        string `json:"url" bson:"url"`
	ShortURL   string `json:"short_url" bson:"short_url"`
	ClickCount int    `json:"click_count" bson:"click_count"`
}

func Analytics(channel *amqp.Channel) {
	channel.Qos(
		20,
		0,
		false,
	)
	msgs, err := channel.Consume(
		"ANALYTICS_QUEUE",
		"",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Panic("Failed to consume data from Queue - ", err)
		return
	}
	forever := make(chan bool)
	go func() {
		for msg := range msgs {
			var analyticsData AnalyticsPayload
			err = json.Unmarshal(msg.Body, &analyticsData)
			if err != nil {
				log.Panic("Error in Unmarshaling - ", err)
			}
			//Add analytics logic if any
		}
	}()
	<-forever
}
