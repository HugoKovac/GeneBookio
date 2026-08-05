package queue

import (
	"fmt"
	"hkorpo/book/internal/primitive"
	"hkorpo/book/pkg/errorwrapper"

	amqp "github.com/rabbitmq/amqp091-go"
)

func InitConsumer(cfg *ConfigQueue, channel primitive.QueueChannel, handler func(d amqp.Delivery)) error {
	conn, err := amqp.Dial(fmt.Sprintf("amqp://%s:%s@localhost:5672/", cfg.USER, cfg.PASSWORD))
	ch, err := conn.Channel()
	if err != nil {
		return errorwrapper.Wrap(err)
	}

	q, err := ch.QueueDeclare(
		string(channel), // name
		true,            // durability
		false,           // delete when unused
		false,           // exclusive
		false,           // no-wait
		amqp.Table{
			amqp.QueueTypeArg: amqp.QueueTypeQuorum,
		},
	)
	if err != nil {
		return errorwrapper.Wrap(err)
	}

	msgs, err := ch.Consume(
		q.Name, // queue
		"",     // consumer
		true,   // auto-ack
		false,  // exclusive
		false,  // no-local
		false,  // no-wait
		nil,    // args
	)
	if err != nil {
		return errorwrapper.Wrap(err)
	}

	go func() {
		for d := range msgs {
			handler(d)
		}
	}()

	return nil
}

func InitProducer(cfg *ConfigQueue, channel primitive.QueueChannel) (*amqp.Queue, *amqp.Channel, error) {
	conn, err := amqp.Dial(fmt.Sprintf("amqp://%s:%s@localhost:5672/", cfg.USER, cfg.PASSWORD))
	ch, err := conn.Channel()
	if err != nil {
		return nil, nil, errorwrapper.Wrap(err)
	}

	q, err := ch.QueueDeclare(
		string(channel), // name
		true,            // durability
		false,           // delete when unused
		false,           // exclusive
		false,           // no-wait
		amqp.Table{
			amqp.QueueTypeArg: amqp.QueueTypeQuorum,
		},
	)
	if err != nil {
		return nil, nil, errorwrapper.Wrap(err)
	}

	return &q, ch, nil
}
