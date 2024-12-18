package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/streadway/amqp"
)

func main() {
	serverAddress := "127.0.0.1:5672"
	if len(os.Args) > 1 {
		serverAddress = os.Args[1]
	}

  conn, err := amqp.Dial("amqp://guest:guest@" + serverAddress)
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %s", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("failed to open a channel: %s", err)
	}
	defer ch.Close()

	err = ch.ExchangeDeclare(
		"chat",   // name
		"direct", // type
		true,     // durable
		false,    // auto-deleted
		false,    // internal
		false,    // no-wait
		nil,      // arguments
	)

	if err != nil {
		log.Fatalf("failed to open a exchanger: %s", err)
	}
	currentChannel := "default" // начальный канал

	q, err := ch.QueueDeclare(currentChannel, false, true, false, false, nil)
	if err != nil {
		log.Fatalf("Failed to declare queue: %s", err)
	}

	msgs, err := ch.Consume(
		q.Name, // queue
		"",     // consumer
		false,  // auto-ack
		false,  // exclusive
		false,  // no-local
		false,  // no-wait
		nil,    // args
	)
	if err != nil {
		log.Fatalf("Failed to start consumer: %s", err)
	}
	to_close := make(chan struct{})
	go func() {
		for {
			select {
			case <-to_close:
				break
			case msg := <-msgs:
				log.Printf("Received a message: %s", msg.Body)
			}
		}
	}()

	for {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Enter message (or !switch channel_name): ")
		text, _ := reader.ReadString('\n')
		text = strings.TrimSpace(text)

		if strings.HasPrefix(text, "!switch ") {
			newChannel := strings.TrimPrefix(text, "!switch ")
			currentChannel = newChannel
			q, err = ch.QueueDeclare(currentChannel, false, true, false, false, nil)
			if err != nil {
				log.Fatalf("Failed to declare queue: %s", err)
			}
			fmt.Printf("Switched to channel: %s\n", currentChannel)
			to_close <- struct{}{}
			msgs, err := ch.Consume(
				q.Name, // queue
				"",     // consumer
				false,  // auto-ack
				false,  // exclusive
				false,  // no-local
				false,  // no-wait
				nil,    // args
			)
			if err != nil {
				log.Fatalf("Failed to start consumer: %s", err)
			}
			go func() {
				for {
					select {
					case <-to_close:
						break
					case msg := <-msgs:
						log.Printf("Received a message: %s", msg.Body)
					}
				}
			}()
			continue
		}

		err = ch.Publish(
			"",     // exchange
			q.Name, // routing key (имя канала)
			false,  // mandatory
			false,  // immediate
			amqp.Publishing{
				ContentType: "text/plain",
				Body:        []byte(text),
			})
		if err != nil {
			log.Fatalf("Failed to publish a message: %s", err)
		}

		fmt.Printf("Sent message to %s: %s\n", currentChannel, text)
	}
}
