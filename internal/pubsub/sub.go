package pubsub

import (
	"bytes"
	"encoding/gob"
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
	unmarshal := func(data []byte) (T, error) {
		var v T
		err := json.Unmarshal(data, &v)
		if err != nil {
			return v, err
		}

		return v, nil
	}

	return subscribe(
		conn,
		exchange,
		queueName,
		key,
		queueType,
		handler,
		unmarshal,
	)
}

func SubscribeGob[T any](
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	queueType SimpleQueueType,
	handler func(T) Acktype,
) error {
	unmarshal := func(data []byte) (T, error) {
		buf := bytes.NewBuffer(data)
		dec := gob.NewDecoder(buf)

		var v T
		err := dec.Decode(&v)
		if err != nil {
			return v, err
		}

		return v, nil
	}

	return subscribe(
		conn,
		exchange,
		queueName,
		key,
		queueType,
		handler,
		unmarshal,
	)
}

func subscribe[T any](
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	queueType SimpleQueueType,
	handler func(T) Acktype,
	unmarshaller func([]byte) (T, error),
) error {
	connChan, queue, err := DeclareAndBind(conn, exchange, queueName, key, queueType)
	if err != nil {
		log.Println("Error declaring", err)
		return err
	}

	deliveryChan, err := connChan.Consume(queue.Name, "", false, false, false, false, nil)
	if err != nil {
		return err
	}

	log.Println("Consume started")

	go func() {
		for c := range deliveryChan {
			log.Printf("rk=%s ct=%s", c.RoutingKey, c.ContentType)
			v, err := unmarshaller(c.Body)
			if err != nil {
				log.Println("Error when unmarshal", err)
				c.Nack(false, false)
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
