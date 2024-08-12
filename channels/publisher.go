package channels

import (
	"log"

	"github.com/streadway/amqp"
)

func Publisher(queueName string, ppublishData []byte) error {
	err := RmqConnection.Publish(
		"",
		queueName,
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         ppublishData,
			DeliveryMode: amqp.Persistent,
		},
	)
	if err != nil {
		return err
	}
	log.Println("Succesfully Published Data : ")
	return nil
}
