package main

import (
	"fmt"
	"log"
	"strconv"
	"time"

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
		handlePause(state),
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
		fmt.Sprintf("%s.%s", routing.ArmyMovesPrefix, username),
		routing.ArmyMovesPrefix+".*",
		pubsub.SimpleQueueTransient,
		handleMove(state, connChan),
	)

	pubsub.SubscribeJSON(
		conn,
		routing.ExchangePerilTopic,
		"war",
		fmt.Sprintf("%s.*", routing.WarRecognitionsPrefix),
		pubsub.SimpleQueueDurable,
		handleWar(state, connChan),
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
				fmt.Sprintf("%s.%s", routing.ArmyMovesPrefix, username),
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
			if len(words) < 1 {
				log.Println("No understand amigo")
				continue
			}

			count, err := strconv.Atoi(words[1])
			if err != nil {
				log.Println("Second command not integer")
				continue
			}

			for range count {
				logMsg := gamelogic.GetMaliciousLog()
				gl := routing.GameLog{
					CurrentTime: time.Now(),
					Message:     logMsg,
					Username:    username,
				}

				publishGameLog(&gl, connChan)
			}
		}

	}

}

func handlePause(gs *gamelogic.GameState) func(routing.PlayingState) pubsub.Acktype {
	return func(state routing.PlayingState) pubsub.Acktype {
		defer fmt.Print("> ")

		gs.HandlePause(state)
		return pubsub.Ack
	}
}

func handleMove(gs *gamelogic.GameState, connChan *amqp.Channel) func(gamelogic.ArmyMove) pubsub.Acktype {
	return func(move gamelogic.ArmyMove) pubsub.Acktype {
		defer fmt.Print("> ")

		outcome := gs.HandleMove(move)

		switch outcome {
		case gamelogic.MoveOutComeSafe:
			return pubsub.Ack
		case gamelogic.MoveOutcomeMakeWar:
			err := pubsub.PublishJSON(
				connChan,
				routing.ExchangePerilTopic,
				fmt.Sprintf("%s.%s", routing.WarRecognitionsPrefix, gs.GetUsername()),
				gamelogic.RecognitionOfWar{
					Attacker: move.Player,
					Defender: gs.GetPlayerSnap(),
				},
			)
			if err != nil {
				fmt.Printf("error: %s\n", err)
				return pubsub.NackRequeue
			}
			return pubsub.Ack
		case gamelogic.MoveOutcomeSamePlayer:
			return pubsub.NackDiscard
		default:
			return pubsub.NackDiscard
		}
	}
}

func handleWar(gs *gamelogic.GameState, connChan *amqp.Channel) func(gamelogic.RecognitionOfWar) pubsub.Acktype {
	return func(rw gamelogic.RecognitionOfWar) pubsub.Acktype {
		defer fmt.Print("> ")

		outcome, winner, loser := gs.HandleWar(rw)

		logMsg := ""

		switch outcome {
		case gamelogic.WarOutcomeNotInvolved:
			return pubsub.NackRequeue
		case gamelogic.WarOutcomeNoUnits:
			return pubsub.NackDiscard
		case gamelogic.WarOutcomeOpponentWon, gamelogic.WarOutcomeYouWon:
			logMsg = fmt.Sprintf("%s won a war against %s", winner, loser)
		case gamelogic.WarOutcomeDraw:
			logMsg = fmt.Sprintf("A war between %s and %s resulted in a draw", winner, loser)
		}

		if logMsg != "" {
			gl := routing.GameLog{
				CurrentTime: time.Now(),
				Message:     logMsg,
				Username:    gs.GetUsername(),
			}

			return publishGameLog(&gl, connChan)
		}

		log.Println("Error outcome")
		return pubsub.NackDiscard
	}
}

func publishGameLog(gl *routing.GameLog, connChan *amqp.Channel) pubsub.Acktype {
	err := pubsub.PublishGob(
		connChan,
		routing.ExchangePerilTopic,
		fmt.Sprintf("%s.%s", routing.GameLogSlug, gl.Username),
		gl,
	)
	if err != nil {
		log.Println("Error publish game log", err)
		return pubsub.NackRequeue
	}

	return pubsub.Ack
}
