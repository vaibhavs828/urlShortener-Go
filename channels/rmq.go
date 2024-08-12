package channels

import (
	"fmt"
	"log"

	"github.com/streadway/amqp"
)

var RmqConnection *amqp.Channel

func RabbitMQConnection() (*amqp.Channel, error) {
	connectionString := fmt.Sprintf("amqp://guest:guest@localhost:5672/")
	conn, err := amqp.Dial(connectionString)
	if err != nil {
		log.Panic("Failed to dial in to RMQ - ", err)
		return nil, err
	}
	ch, err := conn.Channel()
	if err != nil {
		log.Panic("Failed to create channel - ", err)
		return nil, err
	}
	log.Println("RMQ succesfully connected")
	return ch, nil

}
