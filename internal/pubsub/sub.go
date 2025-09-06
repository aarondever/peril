package pubsub

import (
	"encoding/json"

	amqp "github.com/rabbitmq/amqp091-go"
)

func SubscribeJSON[T any](
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	queueType simpleQueueType, // an enum to represent "durable" or "transient"
	handler func(T),
) error {
	connChan, queue, err := DeclareAndBind(conn, exchange, queueName, key, queueType)
	if err != nil {
		return err
	}

	deliveryChan, err := connChan.Consume(queue.Name, "", false, false, false, false, nil)
	if err != nil {
		return err
	}

	go func() {
		for c := range deliveryChan {
			var v T
			if err := json.Unmarshal(c.Body, &v); err != nil {
				continue
			}
			handler(v)
			c.Ack(false)
		}
	}()

	return nil
}
