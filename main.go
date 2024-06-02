package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/gorilla/websocket"
	"github.com/tanush-128/openzo_backend/websocket/config"
	"github.com/tanush-128/openzo_backend/websocket/internal/pb"
	"google.golang.org/grpc"
)

var (
	UserClient pb.UserServiceClient
	upgrader   = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
	clients      = make(map[string]map[*websocket.Conn]bool)
	clientsMutex = &sync.Mutex{}
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal(fmt.Errorf("failed to load config: %w", err))
	}

	// Initialize gRPC client
	conn, err := grpc.Dial(cfg.UserGrpc, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	UserClient = pb.NewUserServiceClient(conn)

	go consumeKafka()

	http.HandleFunc("/echo", echoHandler)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "websockets.html")
	})

	log.Printf("Server starting on port %s", cfg.HTTPPort)
	http.ListenAndServe(fmt.Sprintf(":%s", cfg.HTTPPort), nil)
}

func echoHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade to WebSocket: %v", err)
		http.Error(w, "Failed to upgrade to WebSocket", http.StatusInternalServerError)
		return
	}

	storeId := r.URL.Query().Get("storeId")
	log.Printf("Store ID: %s", storeId)
	if storeId == "" {
		log.Printf("Invalid storeId")
		conn.Close()
		return
	}

	clientsMutex.Lock()
	if clients[storeId] == nil {
		clients[storeId] = make(map[*websocket.Conn]bool)
	}
	clients[storeId][conn] = true
	clientsMutex.Unlock()

	defer func() {
		conn.Close()
		clientsMutex.Lock()
		delete(clients[storeId], conn)
		clientsMutex.Unlock()
	}()

	conn.SetCloseHandler(func(code int, text string) error {
		clientsMutex.Lock()
		delete(clients[storeId], conn)
		clientsMutex.Unlock()
		return nil
	})

	for {
		msgType, msg, err := conn.ReadMessage()
		if err != nil {
			log.Printf("ReadMessage error: %v", err)
			return
		}

		log.Printf("%s sent: %s", conn.RemoteAddr(), string(msg))

		clientsMutex.Lock()
		for client := range clients[storeId] {
			if err := client.WriteMessage(msgType, msg); err != nil {
				log.Printf("WriteMessage error: %v", err)
				client.Close()
				delete(clients[storeId], client)
			}
		}
		clientsMutex.Unlock()
	}
}

// func broadcastMessageByStoreID(clients map[*websocket.Conn]bool, message []byte) {
// 	log.Printf("Broadcasting message to %d clients", len(clients))
// 	for client := range clients {
// 		if err := client.WriteMessage(websocket.TextMessage, message); err != nil {
// 			return
// 		}
// 	}
// }

// type Notification struct {
// 	Message  string `json:"message"`
// 	FCMToken string `json:"fcm_token"`
// 	Data     string `json:"data,omitempty"`
// 	Topic    string `json:"topic,omitempty"`
// }

func consumeKafka() {
	conf := ReadConfig()

	topic := "sales"
	conf["group.id"] = "go-group-1"
	conf["auto.offset.reset"] = "latest"

	consumer, err := kafka.NewConsumer(&conf)
	if err != nil {
		log.Fatalf("Failed to create Kafka consumer: %v", err)
	}
	defer consumer.Close()

	err = consumer.SubscribeTopics([]string{topic}, nil)
	if err != nil {
		log.Fatalf("Failed to subscribe to topics: %v", err)
	}

	run := true
	for run {
		e := consumer.Poll(1000)
		switch ev := e.(type) {
		case *kafka.Message:
			var message map[string]interface{}
			err := json.Unmarshal(ev.Value, &message)
			if err != nil {
				log.Printf("Error unmarshalling JSON: %v", err)
				continue
			}

			storeId, ok := message["store_id"].(string)
			if !ok {
				log.Printf("store_id is missing or not a string in message: %v", message)
				continue
			}

			clientsMutex.Lock()
			if clients[storeId] != nil {
				broadcastMessageByStoreID(clients[storeId], ev.Value)
			}
			clientsMutex.Unlock()
		case kafka.Error:
			// Log Kafka errors but don't stop the consumer
			log.Printf("Kafka error: %v", ev)
		default:
			// Handle other event types if necessary
		}
	}
}

func broadcastMessageByStoreID(storeClients map[*websocket.Conn]bool, message []byte) {
	for client := range storeClients {
		err := client.WriteMessage(websocket.TextMessage, message)
		if err != nil {
			log.Printf("Error writing message to WebSocket client: %v", err)
			client.Close()
			delete(storeClients, client)
		}
	}
}
