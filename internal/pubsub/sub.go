package pubsub

import (
	"encoding/json"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Acktype int

type SimpleQueueType int

const (
	SimpleQueueDurable SimpleQueueType = iota
	SimpleQueueTransient
)

const (
	Ack Acktype = iota
	NackDiscard
	NackRequeue
)

func SubscribeJSON[T any](
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	queueType SimpleQueueType,
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

	go func() {
		for c := range deliveryChan {
			var v T
			if err := json.Unmarshal(c.Body, &v); err != nil {
				continue
			}
			switch handler(v) {
			case Ack:
				c.Ack(false)
				log.Println("Ack")
			case NackRequeue:
				c.Nack(false, true)
				log.Println("NackRequeue")
			case NackDiscard:
				c.Nack(false, false)
				log.Println("NackDiscard")
			}
		}
	}()

	return nil
}
