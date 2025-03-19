package main

import (
	"bytes"
	"encoding/base64"
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/phin1x/go-ipp"
	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	godotenv.Load() //dont' care if it fails

	conn, err := amqp.Dial("amqp://" + os.Getenv("MQ_USERNAME") + ":" + os.Getenv("MQ_PASSWORD") + "@" + os.Getenv("MQ_HOST") + ":5672/")
	failOnError(err, "Failed to connect to RabbitMQ")
	defer conn.Close()

	ch, err := conn.Channel()
	failOnError(err, "Failed to open a channel")
	defer ch.Close()

	q, err := ch.QueueDeclare(
		os.Getenv("QUEUE_NAME"), // name
		true,                    // durable
		false,                   // delete when unused
		false,                   // exclusive
		false,                   // no-wait
		nil,                     // arguments
	)
	failOnError(err, "Failed to declare a queue")

	msgs, err := ch.Consume(
		q.Name, // queue
		"",     // consumer
		true,   // auto-ack
		false,  // exclusive
		false,  // no-local
		false,  // no-wait
		nil,    // args
	)
	failOnError(err, "Failed to register a consumer")
	client := ipp.NewIPPClient(os.Getenv("CUPS_HOST"), 631, "user", "password", true)

	var forever chan struct{}

	go func() {
		for d := range msgs {

			// expecting a string base64 encoded
			// reader := bytes.NewReader(d.Body)

			// convert into actual base64
			reader := base64.NewDecoder(base64.StdEncoding, bytes.NewReader(d.Body))

			myDoc := ipp.Document{
				Document: reader,
				Size:     len(d.Body),
				Name:     "mydoc.pdf",
				MimeType: "application/pdf",
			}

			someCode, err := client.PrintJob(myDoc, os.Getenv("PRINTER_NAME"), map[string]interface{}{})

			log.Println("jobCode", someCode)
			if err != nil {
				log.Println("error", err)
			}

		}
	}()

	log.Printf("[*] Waiting for printable documents. To exit press CTRL+C")
	<-forever
}

func failOnError(err error, msg string) {
	if err != nil {
		log.Panicf("%s: %s", msg, err)
	}
}
