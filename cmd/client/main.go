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

	connChan, err := conn.Channel()
	if err != nil {
		log.Fatal(err)
	}

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
		pubsub.SimpleQueueTransient,
		handlerPause(state),
	)

	pubsub.DeclareAndBind(
		conn,
		routing.ExchangePerilDirect,
		fmt.Sprintf("%s.%s", routing.PauseKey, username),
		routing.PauseKey,
		pubsub.SimpleQueueTransient,
	)

	pubsub.SubscribeJSON(
		conn,
		routing.ExchangePerilTopic,
		fmt.Sprintf("%s.%s", "army_moves", username),
		"army_moves.*",
		pubsub.SimpleQueueTransient,
		handlerMove(state),
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
			move, err := state.CommandMove(words)
			if err != nil {
				log.Println("error when move")
				continue
			}

			pubsub.PublishJSON(
				connChan,
				routing.ExchangePerilTopic,
				fmt.Sprintf("%s.%s", "army_moves", username),
				move,
			)
			log.Println("Move published...")
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

func handlerPause(gs *gamelogic.GameState) func(routing.PlayingState) pubsub.Acktype {
	return func(state routing.PlayingState) pubsub.Acktype {
		defer fmt.Print("> ")

		gs.HandlePause(state)
		return pubsub.Ack
	}
}

func handlerMove(gs *gamelogic.GameState) func(gamelogic.ArmyMove) pubsub.Acktype {
	return func(move gamelogic.ArmyMove) pubsub.Acktype {
		defer fmt.Print("> ")

		outcome := gs.HandleMove(move)

		switch outcome {
		case gamelogic.MoveOutComeSafe, gamelogic.MoveOutcomeMakeWar:
			return pubsub.Ack
		case gamelogic.MoveOutcomeSamePlayer:
			return pubsub.NackDiscard
		default:
			return pubsub.NackDiscard
		}
	}
}
