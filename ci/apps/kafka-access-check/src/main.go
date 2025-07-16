package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	dto "github.com/prometheus/client_model/go"
)

var (
	currentBootstrap = getEnvOrDefault("KAFKA_BOOTSTRAP", "localhost:9093")
	currentTopic     = getEnvOrDefault("KAFKA_TOPIC", "test-topic")
	currentUsername  = getEnvOrDefault("KAFKA_USERNAME", "user")
	currentPassword  = getEnvOrDefault("KAFKA_PASSWORD", "pass")
	lastResult       = "❔ Kafka access not yet attempted"

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

func getEnvOrDefault(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

func init() {
	prometheus.MustRegister(kafkaWriteSuccess)
	prometheus.MustRegister(kafkaWriteFailure)
}

func getCounterValue(counter prometheus.Counter) float64 {
	pb := &dto.Metric{}
	if err := counter.Write(pb); err != nil {
		return 0
	}
	return pb.GetCounter().GetValue()
}

// --- In-memory message cache for demo consumer ---
type cachedMsg struct {
	Time    time.Time
	Topic   string
	Group   string
	Message string
}

var (
	// Very simple cache: map[group+topic] -> slice of last 10 messages
	consumerCache = make(map[string][]cachedMsg)
)

func cacheConsumerMessage(topic, group, msg string) {
	key := group + "|" + topic
	arr := consumerCache[key]
	if len(arr) > 9 {
		arr = arr[1:]
	}
	arr = append(arr, cachedMsg{Time: time.Now(), Topic: topic, Group: group, Message: msg})
	consumerCache[key] = arr
}

func getConsumerMessages(topic, group string) []cachedMsg {
	return consumerCache[group+"|"+topic]
}

func main() {
	http.Handle("/metrics", promhttp.Handler())

	// Main page: show tabs for Producer and Consumer
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		successCount := getCounterValue(kafkaWriteSuccess)
		failureCount := getCounterValue(kafkaWriteFailure)
		fmt.Fprintf(w, `
  <html>
  <head>
    <title>Kafka Access</title>
    <style>
      body { font-family: sans-serif; padding: 24px; background: #f8f9fa; }
      h1 { font-size: 32px; margin-bottom: 10px; }
      .status { margin: 12px 0; padding: 10px; background: #fff3cd; border: 1px solid #ffeeba; border-radius: 4px; }
      .tabs { margin-top: 24px; }
      .tab-btn {
        padding: 10px 20px;
        border: none;
        background: #e9ecef;
        margin-right: 6px;
        cursor: pointer;
        font-size: 16px;
        border-radius: 4px 4px 0 0;
      }
      .tab-btn.active {
        background: #ffffff;
        font-weight: bold;
        border-bottom: 2px solid #007BFF;
      }
      .tab-content {
        border: 1px solid #dee2e6;
        padding: 16px;
        background: #ffffff;
        border-radius: 0 4px 4px 4px;
      }
      textarea {
        width: 100%%;
        padding: 8px;
        border: 1px solid #ccc;
        border-radius: 4px;
      }
      input[type="submit"], button {
        margin-top: 10px;
        padding: 10px 16px;
        font-size: 16px;
        background-color: #007BFF;
        color: white;
        border: none;
        border-radius: 4px;
        cursor: pointer;
      }
      input[type="submit"]:hover, button:hover {
        background-color: #0056b3;
      }
    </style>
  </head>
  <body>
    <h1>Kafka Access Demo</h1>
    <div class="status"><b>Success Count:</b> %v<br><b>Failure Count:</b> %v<br><b>Last Result:</b> %s</div>
    <p><a href="/settings">⚙️ Settings</a> | <a href="/metrics">📈 Metrics</a></p>
    <div class="tabs">
      <button class="tab-btn active" id="producer-tab-btn" onclick="showTab('producer')">Producer</button>
      <button class="tab-btn" id="consumer-tab-btn" onclick="showTab('consumer')">Consumer</button>
    </div>
    <div class="tab-content" id="producer-tab">
      <h2>Producer</h2>
      <form id="producer-form" method="POST" action="/settings" onsubmit="return submitProducer(event)">
        <input type="hidden" name="action" value="produce">
        <label>Message:</label><br>
        <textarea name="message" id="producer-message" rows="2"></textarea><br>
        <input type="submit" value="Send">
      </form>
      <div id="producer-result"></div>
    </div>
    <div class="tab-content" id="consumer-tab" style="display:none">
      <h2>Consumer</h2>
      <form id="consumer-form" onsubmit="return startConsumer(event)">
        <label>Topic:</label> <input name="topic" id="consumer-topic" value="%s"><br>
        <label>Group:</label> <input name="group" id="consumer-group" value="test-group"><br>
        <input type="submit" value="Start">
      </form>
      <div>
        <button onclick="refreshConsumer()">Refresh</button>
      </div>
      <div id="consumer-result" style="margin-top:8px; font-family:monospace;"></div>
    </div>
    <script>
      function showTab(name) {
        document.getElementById('producer-tab').style.display = name === 'producer' ? '' : 'none';
        document.getElementById('consumer-tab').style.display = name === 'consumer' ? '' : 'none';
        document.getElementById('producer-tab-btn').classList.toggle('active', name==='producer');
        document.getElementById('consumer-tab-btn').classList.toggle('active', name==='consumer');
      }
      function submitProducer(ev) {
        ev.preventDefault();
        const msg = document.getElementById('producer-message').value;
        const form = ev.target;
        const data = new FormData(form);
        fetch(form.action, {
          method: 'POST',
          body: data
        }).then(r=>r.text()).then(txt=>{
          document.getElementById('producer-result').innerText = txt;
        });
        return false;
      }
      function startConsumer(ev) {
        ev.preventDefault();
        refreshConsumer();
        return false;
      }
      function refreshConsumer() {
        const topic = document.getElementById('consumer-topic').value;
        const group = document.getElementById('consumer-group').value;
        fetch('/consume?topic='+encodeURIComponent(topic)+'&group='+encodeURIComponent(group))
          .then(r=>r.json())
          .then(arr=>{
            let txt = '';
            for (const entry of arr) {
              txt += '['+entry.Time+'] '+entry.Message+'\\n';
            }
            document.getElementById('consumer-result').innerText = txt || '(No messages)';
          });
      }
    </script>
  </body>
  </html>
  `, successCount, failureCount, lastResult, currentTopic)
	})

	// Settings and Producer handler
	http.HandleFunc("/settings", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			action := r.FormValue("action")
			if action == "produce" {
				// Send message via producer
				msg := r.FormValue("message")
				go func(message string) {
					err := produceKafkaMessage(message)
					if err != nil {
						kafkaWriteFailure.Inc()
						lastResult = fmt.Sprintf("❌ Kafka produce failed: %v", err)
					} else {
						kafkaWriteSuccess.Inc()
						lastResult = "✅ Kafka produce succeeded"
					}
				}(msg)
				fmt.Fprintf(w, "Message sent (async)")
				return
			}
			// Settings update
			currentBootstrap = r.FormValue("bootstrap")
			currentTopic = r.FormValue("topic")
			currentUsername = r.FormValue("username")
			currentPassword = r.FormValue("password")

			go func() {
				err := testKafkaWrite()
				if err != nil {
					kafkaWriteFailure.Inc()
					lastResult = fmt.Sprintf("❌ Kafka write failed: %v", err)
				} else {
					kafkaWriteSuccess.Inc()
					lastResult = "✅ Kafka write succeeded"
				}
			}()

			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		fmt.Fprintf(w, `
  <html>
  <head>
    <title>Kafka Settings</title>
    <style>
      body { font-family: sans-serif; padding: 20px; background-color: #f5f5f5; }
      h1 { font-size: 28px; }
      label { display: block; margin-top: 12px; font-weight: bold; }
      input[type=text], input[type=password] {
        width: 300px; padding: 8px; margin-top: 4px; border: 1px solid #ccc; border-radius: 4px;
      }
      button, input[type=submit] {
        margin-top: 20px; padding: 10px 16px; font-size: 16px;
        border: none; border-radius: 4px; background-color: #007BFF; color: white; cursor: pointer;
      }
      button:hover, input[type=submit]:hover {
        background-color: #0056b3;
      }
      #statusResult {
        margin-top: 20px; font-weight: bold;
      }
    </style>
    <script>
      async function testConnection() {
        const result = document.getElementById("statusResult");
        result.innerText = "Testing...";
        result.style.color = "black";
        try {
          const form = document.querySelector("form");
          const data = new FormData(form);
          const params = new URLSearchParams(data);
          const res = await fetch("/test?" + params.toString());
          const txt = await res.text();
          if (res.ok) {
            result.innerText = "✅ " + txt;
            result.style.color = "green";
          } else {
            result.innerText = "❌ " + txt;
            result.style.color = "red";
          }
        } catch (e) {
          result.innerText = "❌ Error: " + e;
          result.style.color = "red";
        }
      }
    </script>
  </head>
  <body>
    <h1>Kafka Settings</h1>
    <form method="POST">
      <label>Bootstrap:</label>
      <input name="bootstrap" value="%s">
      <label>Topic:</label>
      <input name="topic" value="%s">
      <label>Username:</label>
      <input name="username" value="%s">
      <label>Password:</label>
      <input type="password" name="password" value="%s">
      <br>
      <input type="submit" value="Update">
      <button type="button" onclick="testConnection()">Test Connection</button>
    </form>
    <div id="statusResult"></div>
    <p><a href="/">Back</a></p>
  </body>
  </html>
  `, currentBootstrap, currentTopic, currentUsername, currentPassword)
	})

	// Consumer handler (demo, not a real Kafka consumer, just shows cached messages)
	http.HandleFunc("/consume", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		topic := r.URL.Query().Get("topic")
		group := r.URL.Query().Get("group")
		msgs := getConsumerMessages(topic, group)
		fmt.Fprintf(w, "[")
		for i, m := range msgs {
			if i > 0 {
				fmt.Fprint(w, ",")
			}
			fmt.Fprintf(w, `{"Time":"%s","Topic":%q,"Group":%q,"Message":%q}`, m.Time.Format("15:04:05"), m.Topic, m.Group, m.Message)
		}
		fmt.Fprint(w, "]")
	})

	http.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		bootstrap := r.URL.Query().Get("bootstrap")
		topic := r.URL.Query().Get("topic")
		username := r.URL.Query().Get("username")
		password := r.URL.Query().Get("password")

		// fallback to current settings if not provided
		if bootstrap == "" {
			bootstrap = currentBootstrap
		}
		if topic == "" {
			topic = currentTopic
		}
		if username == "" {
			username = currentUsername
		}
		if password == "" {
			password = currentPassword
		}

		err := produceKafkaMessageWithCreds("Test message", bootstrap, topic, username, password)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		fmt.Fprint(w, "Kafka write test succeeded")
	})

	log.Println("✅ Serving on :2112")
	log.Fatal(http.ListenAndServe(":2112", nil))
}

func testKafkaWrite() error {
	return produceKafkaMessage("Test message")
}

func produceKafkaMessage(msgStr string) error {
	producer, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers": currentBootstrap,
		"security.protocol": "SASL_SSL",
		"sasl.mechanism":    "SCRAM-SHA-512",
		"sasl.username":     currentUsername,
		"sasl.password":     currentPassword,
	})
	if err != nil {
		return fmt.Errorf("producer creation failed: %w", err)
	}
	defer producer.Close()

	topic := currentTopic
	msg := &kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
		Value:          []byte(msgStr),
	}

	err = producer.Produce(msg, nil)
	if err != nil {
		return fmt.Errorf("produce failed: %w", err)
	}
	producer.Flush(5000)

	// For demo: push to the consumer cache (simulate as if consumer receives it)
	cacheConsumerMessage(topic, "test-group", msgStr)
	return nil
}

func produceKafkaMessageWithCreds(msgStr, bootstrap, topic, username, password string) error {
	host, port, err := net.SplitHostPort(bootstrap)
	if err != nil {
		return fmt.Errorf("invalid bootstrap format (expected host:port): %w", err)
	}

	conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, port), 3*time.Second)
	if err != nil {
		return fmt.Errorf("cannot reach broker: %w", err)
	}
	conn.Close()
	return nil
}