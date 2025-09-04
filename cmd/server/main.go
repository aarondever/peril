package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	if err != nil {
		log.Fatal("Error connecting RabbitMQ", err)
	}
	defer conn.Close()

	fmt.Println("Connect to RabbitMQ successfully...")

	connChan, err := conn.Channel()
	if err != nil {
		log.Fatal("Error open channel")
	}

	pubsub.PublishJSON(connChan, routing.ExchangePerilDirect, routing.PauseKey, routing.PlayingState{
		IsPaused: true,
	})

	fmt.Println("Starting Peril server...")

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	<-signalChan

	fmt.Println("Shutting down program...")
}
