

package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	kafkaWriteSuccess = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "kafka_write_success_total",
			Help: "Total number of successful Kafka write attempts",
		},
	)
	kafkaWriteFailure = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "kafka_write_failure_total",
			Help: "Total number of failed Kafka write attempts",
		},
	)
)

func init() {
	prometheus.MustRegister(kafkaWriteSuccess)
	prometheus.MustRegister(kafkaWriteFailure)
}

func main() {
	go startMetricsServer()

	topic := os.Getenv("KAFKA_TOPIC")
	if topic == "" {
		topic = "test-topic"
	}

	producer, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers": os.Getenv("KAFKA_BOOTSTRAP"),
		"security.protocol": "SASL_SSL",
		"sasl.mechanism":    "SCRAM-SHA-512",
		"sasl.username":     os.Getenv("KAFKA_USERNAME"),
		"sasl.password":     os.Getenv("KAFKA_PASSWORD"),
	})
	if err != nil {
		log.Fatalf("Failed to create producer: %v", err)
	}
	defer producer.Close()

	msg := &kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
		Value:          []byte("Test message"),
	}

	err = producer.Produce(msg, nil)
	if err != nil {
		kafkaWriteFailure.Inc()
		log.Printf("❌ Kafka write failed: %v\n", err)
	} else {
		kafkaWriteSuccess.Inc()
		log.Println("✅ Kafka write succeeded")
	}

	producer.Flush(5000)
	time.Sleep(5 * time.Second)
}

func startMetricsServer() {
	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `
			<html><body>
				<h1>Kafka Access Status</h1>
				<p><b>Success Count:</b> %v</p>
				<p><b>Failure Count:</b> %v</p>
				<p>See <a href="/metrics">/metrics</a> for Prometheus output</p>
			</body></html>
		`, kafkaWriteSuccess, kafkaWriteFailure)
	})
	log.Println("Serving metrics on :2112/metrics")
	log.Fatal(http.ListenAndServe(":2112", nil))
}