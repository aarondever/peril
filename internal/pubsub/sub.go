package pubsub

import (
	"encoding/json"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Acktype string

func SubscribeJSON[T any](
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	queueType simpleQueueType, // an enum to represent "durable" or "transient"
	handler func(T) Acktype,
) error {
	connChan, queue, err := DeclareAndBind(conn, exchange, queueName, key, queueType)
	if err != nil {
		return err
	}

	deliveryChan, err := connChan.Consume(queue.Name, "", false, false, false, false, nil)
	if err != nil {
		return err
	}

	go func() Acktype {
		for c := range deliveryChan {
			var v T
			if err := json.Unmarshal(c.Body, &v); err != nil {
				continue
			}
			handler(v)
			c.Ack(false)
			log.Println("acktype: Ack")
			return "Ack"
		}

		log.Println("acktype: NackDiscard")
		return "NackDiscard"
	}()

	return nil
}
