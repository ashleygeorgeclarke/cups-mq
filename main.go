package main

import (
	"bytes"
	"encoding/base64"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/phin1x/go-ipp"
	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	godotenv.Load() //dont' care if it fails

	for {
		conn, err := amqp.Dial("amqp://" + os.Getenv("MQ_USERNAME") + ":" + os.Getenv("MQ_PASSWORD") + "@" + os.Getenv("MQ_HOST") + ":5672/")
		if err != nil {
			log.Println("[!] Failed to connect to RabbitMQ, retrying in 5 seconds:", err)
			time.Sleep(5 * time.Second)
			continue
		}
		log.Println("[*] Connected to RabbitMQ")
		ch, err := conn.Channel()
		if err != nil {
			log.Println("[!] Failed to open a channel, retrying in 5 seconds:", err)
			conn.Close()
			time.Sleep(5 * time.Second)
			continue
		}
		q, err := ch.QueueDeclare(
			os.Getenv("QUEUE_NAME"), // name
			true,                    // durable
			false,                   // delete when unused
			false,                   // exclusive
			false,                   // no-wait
			nil,                     // arguments
		)
		if err != nil {
			log.Println("[!] Failed to declare a queue, retrying in 5 seconds:", err)
			ch.Close()
			conn.Close()
			time.Sleep(5 * time.Second)
			continue
		}
		msgs, err := ch.Consume(
			q.Name, // queue
			"",     // consumer
			true,   // auto-ack
			false,  // exclusive
			false,  // no-local
			false,  // no-wait
			nil,    // args
		)
		if err != nil {
			log.Println("[!] Failed to register a consumer, retrying in 5 seconds:", err)
			ch.Close()
			conn.Close()
			time.Sleep(5 * time.Second)
			continue
		}
		client := ipp.NewIPPClient(os.Getenv("CUPS_HOST"), 631, "user", "password", true)

		// Start consumer goroutine
		consumerDone := make(chan struct{})
		go func() {
			for d := range msgs {
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
			log.Println("[!] Message channel closed. Consumer goroutine exiting.")
			close(consumerDone)
		}()

		log.Printf("[*] Waiting for printable documents. To exit press CTRL+C")
		<-consumerDone // Wait for consumer to exit (msgs channel closed)
		log.Println("[!] Attempting to reconnect to RabbitMQ in 5 seconds...")
		ch.Close()
		conn.Close()
		time.Sleep(5 * time.Second)
	}
}

func failOnError(err error, msg string) {
	if err != nil {
		log.Panicf("%s: %s", msg, err)
	}
}
