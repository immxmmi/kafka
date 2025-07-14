package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/confluentinc/confluent-kafka-go/kafka"
)

// Configurable settings
var (
	mqttBroker  = os.Getenv("MQTT_BROKER")
	mqttTopic   = os.Getenv("MQTT_TOPIC")
	kafkaBroker = os.Getenv("KAFKA_BROKER")
	kafkaTopic  = os.Getenv("KAFKA_TOPIC")
)

// MQTT Message Handler
func messageHandler(client mqtt.Client, msg mqtt.Message) {
	timestamp := time.Now().Format("02012006150405") // Format: DDMMYYYYHHMMSS
	fmt.Printf("[%s] Received MQTT message on topic: %s\n", timestamp, msg.Topic())

	// Mapping logic: Convert MQTT topic to Kafka topic dynamically
	mappedTopic := strings.ReplaceAll(msg.Topic(), "/", "-")
	if kafkaTopic != "" {
		mappedTopic = kafkaTopic
	}

	// Publish to Kafka
	publishToKafka(mappedTopic, msg.Payload(), timestamp)
}

// Kafka Producer Function
func publishToKafka(topic string, message []byte, timestamp string) {
	data := map[string]interface{}{
		"timestamp": timestamp,
		"message":   string(message),
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		log.Printf("[%s] Failed to marshal message to JSON: %v", timestamp, err)
		return
	}

	p, err := kafka.NewProducer(&kafka.ConfigMap{"bootstrap.servers": kafkaBroker})
	if err != nil {
		log.Fatalf("[%s] ❌ Kafka producer creation failed: %v", timestamp, err)
	}
	defer p.Close()

	err = p.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
		Value:          jsonData,
	}, nil)

	if err != nil {
		log.Printf("[%s] ❌ Failed to produce Kafka message: %v", timestamp, err)
	} else {
		log.Printf("[%s] ✅ Kafka message sent to topic: %s", timestamp, topic)
	}

	// Wait for all messages to be delivered
	p.Flush(15 * 1000)
}

func main() {
	// Display configuration before starting
	log.Println("Starting MQTT-Kafka Bridge...")
	log.Printf("MQTT Broker: %s", mqttBroker)
	log.Printf("MQTT Topic: %s", mqttTopic)
	log.Printf("Kafka Broker: %s", kafkaBroker)
	log.Printf("Kafka Topic: %s", kafkaTopic)

	adminClient, err := kafka.NewAdminClient(&kafka.ConfigMap{"bootstrap.servers": kafkaBroker})
	if err != nil {
		log.Fatalf("❌ Kafka connection check failed: %v", err)
	}
	adminClient.Close()

	opts := mqtt.NewClientOptions().AddBroker(mqttBroker)
	opts.SetDefaultPublishHandler(messageHandler)

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatalf("MQTT Connection failed: %v", token.Error())
	}

	if token := client.Subscribe(mqttTopic, 0, messageHandler); token.Wait() && token.Error() != nil {
		log.Fatalf("MQTT Subscription failed: %v", token.Error())
	}

	http.HandleFunc("/health/liveness", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	http.HandleFunc("/health/readiness", func(w http.ResponseWriter, r *http.Request) {
		kClient, err := kafka.NewAdminClient(&kafka.ConfigMap{"bootstrap.servers": kafkaBroker})
		if err != nil {
			http.Error(w, "Kafka not ready", http.StatusServiceUnavailable)
			return
		}
		kClient.Close()

		if !client.IsConnected() {
			http.Error(w, "MQTT not ready", http.StatusServiceUnavailable)
			return
		}

		w.WriteHeader(http.StatusOK)
	})

	go http.ListenAndServe(":8080", nil)

	log.Println("MQTT-Kafka Bridge is running...")
	select {}
}
