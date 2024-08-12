package channels

func QueueConnect(name string, durable bool) error {
	RmqConnection, err := RabbitMQConnection()
	if err != nil {
		return err
	}
	_, err = RmqConnection.QueueDeclare(
		name,
		durable,
		false,
		false,
		false,
		nil,
	)

	return err
}
