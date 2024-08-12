package analytics

import (
	"encoding/json"
	"log"

	"github.com/streadway/amqp"
)

type AnalyticsPayload struct {
	URL      string
	ShortURL string
	//other data
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
			//Add Mongo Logic here
		}
	}()
	<-forever
}
