package book

import (
	"context"
	"hkorpo/book/pkg/errorwrapper"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type QueueRepo interface {
	PostMessage(msg string) error
}

type QueueRepoImpl struct {
	q  *amqp.Queue
	ch *amqp.Channel
}

func NewQueueRepoImpl(q *amqp.Queue, ch *amqp.Channel) *QueueRepoImpl {
	return &QueueRepoImpl{
		q:  q,
		ch: ch,
	}
}

func (qr *QueueRepoImpl) PostMessage(msg string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := qr.ch.PublishWithContext(ctx,
		"",        // exchange
		qr.q.Name, // routing key
		false,     // mandatory
		false,     // immediate
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        []byte(msg),
		})

	return errorwrapper.Wrap(err)
}
