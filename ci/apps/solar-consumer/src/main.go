package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/confluentinc/confluent-kafka-go/kafka"
)

type SolarData struct {
	Timestamp        string  `json:"timestamp"`
	SolarInputPower  float64 `json:"solarInputPower"`
	OutputHomePower  float64 `json:"outputHomePower"`
	ElectricLevel    float64 `json:"electricLevel"`
	OutputPackPower  float64 `json:"outputPackPower"`
}

var (
	dataPoints []SolarData
	dataMutex  sync.Mutex
	maxData    = 100
)

func main() {
	go startKafkaConsumer()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./static/index.html")
	})

	http.HandleFunc("/api/data", func(w http.ResponseWriter, r *http.Request) {
		dataMutex.Lock()
		defer dataMutex.Unlock()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(dataPoints)
	})

	http.HandleFunc("/health/liveness", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	http.HandleFunc("/health/readiness", func(w http.ResponseWriter, r *http.Request) {
		broker := os.Getenv("KAFKA_BROKER")
		adminClient, err := kafka.NewAdminClient(&kafka.ConfigMap{"bootstrap.servers": broker})
		if err != nil {
			http.Error(w, "Kafka not ready", http.StatusServiceUnavailable)
			return
		}
		adminClient.Close()
		w.WriteHeader(http.StatusOK)
	})

	log.Println("🌐 Server running at http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func startKafkaConsumer() {
	broker := os.Getenv("KAFKA_BROKER")
	topic := os.Getenv("KAFKA_TOPIC")
	if broker == "" || topic == "" {
		log.Fatal("KAFKA_BROKER and KAFKA_TOPIC must be set")
	}

	// Check broker connectivity before starting
	adminClient, err := kafka.NewAdminClient(&kafka.ConfigMap{"bootstrap.servers": broker})
	if err != nil {
		log.Fatalf("❌ Unable to connect to Kafka broker: %v", err)
	}
	defer adminClient.Close()

	config := &kafka.ConfigMap{
		"bootstrap.servers": broker,
		"group.id":          "web-consumer-solar-group",
		"auto.offset.reset": "earliest",
	}

	consumer, err := kafka.NewConsumer(config)
	if err != nil {
		log.Fatalf("❌ Failed to create consumer: %v", err)
	}

	err = consumer.SubscribeTopics([]string{topic}, nil)
	if err != nil {
		log.Fatalf("❌ Failed to subscribe to topic: %v", err)
	}

	defer consumer.Close()

	log.Printf("✅ Kafka consumer started. Listening to topic '%s' on broker '%s'...", topic, broker)

	for {
		msg, err := consumer.ReadMessage(-1)
		if err != nil {
			log.Printf("❌ Kafka read error: %v", err)
			continue
		}

		var raw map[string]interface{}
		if err := json.Unmarshal(msg.Value, &raw); err != nil {
			log.Printf("⚠️ Failed to unmarshal outer message: %v", err)
			continue
		}

		timestamp, _ := raw["timestamp"].(string)
		messageStr, _ := raw["message"].(string)

		var solarData SolarData
		if err := json.Unmarshal([]byte(messageStr), &solarData); err != nil {
			log.Printf("⚠️ Failed to unmarshal inner message: %v", err)
			continue
		}
		solarData.Timestamp = timestamp

		dataMutex.Lock()
		dataPoints = append(dataPoints, solarData)
		if len(dataPoints) > maxData {
			dataPoints = dataPoints[1:]
		}
		dataMutex.Unlock()

		log.Printf("📥 %s | SolarInputPower: %.2f | OutputHomePower: %.2f | ElectricLevel: %.2f | OutputPackPower: %.2f",
			solarData.Timestamp, solarData.SolarInputPower, solarData.OutputHomePower, solarData.ElectricLevel, solarData.OutputPackPower)
	}
}