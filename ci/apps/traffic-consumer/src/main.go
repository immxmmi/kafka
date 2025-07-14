package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/confluentinc/confluent-kafka-go/kafka"
)

type TrafficData struct {
	CarsPerMinute    int     `json:"carsPerMinute"`
	AverageSpeed     float64 `json:"averageSpeed"`
	TrafficDensity   float64 `json:"trafficDensity"`
	IncidentReported bool    `json:"incidentReported"`
}

var (
	dataPoints []TrafficData
	dataMutex  sync.Mutex
	maxData    = 20
)

func main() {
	go func() {
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

		http.ListenAndServe(":8081", nil)
	}()

	go startKafkaConsumer()

	http.HandleFunc("/", serveIndex)
	http.HandleFunc("/api/data", serveAPI)

	log.Println("🌐 Server running at http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func serveIndex(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "./static/index.html")
}

func serveAPI(w http.ResponseWriter, r *http.Request) {
	dataMutex.Lock()
	defer dataMutex.Unlock()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(dataPoints)
}

func startKafkaConsumer() {
	broker := os.Getenv("KAFKA_BROKER")
	topic := os.Getenv("KAFKA_TOPIC")
	if broker == "" || topic == "" {
		log.Fatal("KAFKA_BROKER and KAFKA_TOPIC must be set")
	}

	adminClient, err := kafka.NewAdminClient(&kafka.ConfigMap{"bootstrap.servers": broker})
	if err != nil {
		log.Fatalf("❌ Unable to connect to Kafka broker: %v", err)
	}
	defer adminClient.Close()

	config := &kafka.ConfigMap{
		"bootstrap.servers": broker,
		"group.id":          "web-consumer-traffic-group",
		"auto.offset.reset": "earliest",
	}

	consumer, err := kafka.NewConsumer(config)
	if err != nil {
		log.Fatalf("❌ Failed to create consumer: %v", err)
	}
	defer consumer.Close()

	err = consumer.SubscribeTopics([]string{topic}, nil)
	if err != nil {
		log.Fatalf("❌ Failed to subscribe to topic: %v", err)
	}

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

		messageStr, _ := raw["message"].(string)

		var traffic TrafficData
		if err := json.Unmarshal([]byte(messageStr), &traffic); err != nil {
			log.Printf("⚠️ Failed to unmarshal TrafficData: %v", err)
			continue
		}

		dataMutex.Lock()
		dataPoints = append(dataPoints, traffic)
		if len(dataPoints) > maxData {
			dataPoints = dataPoints[1:]
		}
		dataMutex.Unlock()

		log.Printf("🚗 Cars: %d/min | 🛣️ Speed: %.2f km/h | 🚦 Density: %.2f | 🚨 Incident: %v",
			traffic.CarsPerMinute, traffic.AverageSpeed, traffic.TrafficDensity, traffic.IncidentReported)
	}
}