package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/confluentinc/confluent-kafka-go/kafka"
)

type WeatherData struct {
	Timestamp      string  `json:"timestamp"`
	CloudCover     float64 `json:"cloudcover"`
	Humidity       float64 `json:"humidity"`
	Precipitation  float64 `json:"precipitation"`
	Temperature    float64 `json:"temperature"`
	WindDirection  float64 `json:"winddirection"`
	WindGusts      float64 `json:"windgusts"`
	WindSpeed      float64 `json:"windspeed"`
}

var (
	dataPoints []WeatherData
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

	adminClient, err := kafka.NewAdminClient(&kafka.ConfigMap{"bootstrap.servers": broker})
	if err != nil {
		log.Fatalf("❌ Unable to connect to Kafka broker: %v", err)
	}
	defer adminClient.Close()

	config := &kafka.ConfigMap{
		"bootstrap.servers": broker,
		"group.id":          "web-consumer-weather-group",
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

		timestamp, _ := raw["timestamp"].(string)
		messageStr, _ := raw["message"].(string)

		var payload map[string]float64
		if err := json.Unmarshal([]byte(messageStr), &payload); err != nil {
			log.Printf("⚠️ Failed to unmarshal inner message: %v", err)
			continue
		}

		dp := WeatherData{
			Timestamp:      timestamp,
			CloudCover:     payload["cloudcover"],
			Humidity:       payload["humidity"],
			Precipitation:  payload["precipitation"],
			Temperature:    payload["temperature"],
			WindDirection:  payload["winddirection"],
			WindGusts:      payload["windgusts"],
			WindSpeed:      payload["windspeed"],
		}

		dataMutex.Lock()
		dataPoints = append(dataPoints, dp)
		if len(dataPoints) > maxData {
			dataPoints = dataPoints[1:]
		}
		dataMutex.Unlock()

		log.Printf("📥 %s | Temp: %.2f°C | Humidity: %.2f%% | Wind: %.2f km/h | Gusts: %.2f | Precip: %.2f mm | CloudCover: %.2f%% | Direction: %.2f°",
			dp.Timestamp, dp.Temperature, dp.Humidity, dp.WindSpeed, dp.WindGusts, dp.Precipitation, dp.CloudCover, dp.WindDirection)
	}
}