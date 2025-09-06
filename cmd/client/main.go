package main

import (
	"fmt"
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

	fmt.Println("Connect to RabbitMQ successfully...")

	username, err := gamelogic.ClientWelcome()
	if err != nil {
		log.Fatal(err)
	}

	state := gamelogic.NewGameState(username)

	pubsub.SubscribeJSON(
		conn,
		routing.ExchangePerilDirect,
		fmt.Sprintf("%s.%s", routing.PauseKey, username),
		routing.PauseKey,
		"transient",
		handlerPause(state),
	)

	pubsub.DeclareAndBind(
		conn,
		routing.ExchangePerilDirect,
		fmt.Sprintf("%s.%s", routing.PauseKey, username),
		routing.PauseKey,
		"transient",
	)

	for {
		words := gamelogic.GetInput()
		if len(words) <= 0 {
			continue
		}

		switch words[0] {
		case "spawn":
			state.CommandSpawn(words)
		case "move":
			state.CommandMove(words)
		case "status":
			state.CommandStatus()
		case "help":
			gamelogic.PrintClientHelp()
		case "quit":
			gamelogic.PrintQuit()
			return
		default:
			log.Println("No understand amigo")
		}

	}

}

func handlerPause(gs *gamelogic.GameState) func(routing.PlayingState) {
	defer fmt.Print("> ")

	return gs.HandlePause
}
