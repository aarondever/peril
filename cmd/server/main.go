package main

import (
	"log"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/gamelogic"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	pubsub.DeclareAndBind(
		conn,
		routing.ExchangePerilTopic,
		"game_logs",
		"game_logs.*",
		"durable",
	)

	log.Println("Connect to RabbitMQ successfully...")

	gamelogic.PrintClientHelp()

	for {
		words := gamelogic.GetInput()
		if len(words) <= 0 {
			continue
		}

		switch words[0] {
		case "pause":
			log.Println("Sending a pause message")
			publishMessage(conn, true)
		case "resume":
			log.Println("Sending a resume message")
			publishMessage(conn, false)
		case "quit":
			log.Println("Exiting")
			return
		default:
			log.Println("No understand amigo")
		}

	}
}

func publishMessage(conn *amqp.Connection, isPaused bool) {
	connChan, err := conn.Channel()
	if err != nil {
		log.Fatal(err)
	}

	pubsub.PublishJSON(
		connChan,
		routing.ExchangePerilDirect,
		routing.PauseKey,
		routing.PlayingState{
			IsPaused: isPaused,
		},
	)
}
