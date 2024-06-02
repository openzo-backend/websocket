package main

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"

	"net/http"

	"github.com/tanush-128/openzo_backend/websocket/config"
	"github.com/tanush-128/openzo_backend/websocket/internal/pb"
	"google.golang.org/grpc"

	"github.com/gorilla/websocket"
)

var UserClient pb.UserServiceClient

type User2 struct {
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

//group connections by store id

func main() {

	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal(fmt.Errorf("failed to load config: %w", err))
	}

	//Initialize gRPC client
	conn, err := grpc.Dial(cfg.UserGrpc, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := pb.NewUserServiceClient(conn)
	UserClient = c

	var clients = make(map[string]map[*websocket.Conn]bool)
	go consumeKafka(clients)

	http.HandleFunc("/echo", func(w http.ResponseWriter, r *http.Request) {
		conn, _ := upgrader.Upgrade(w, r, nil) // error ignored for sake of simplicity
		storeId := r.URL.Query().Get("storeId")
		log.Printf("Store ID: %s", storeId)
		if clients[storeId] == nil {
			clients[storeId] = make(map[*websocket.Conn]bool)
		}
		clients[storeId][conn] = true
		defer conn.Close()

		for {

			msgType, msg, err := conn.ReadMessage()
			if err != nil {
				return
			}

			// Print the message to the console
			fmt.Printf("%s sent: %s\n", conn.RemoteAddr(), string(msg))

			for client := range clients[storeId] {
				if err = client.WriteMessage(msgType, msg); err != nil {
					return
				}
			}
			// remove on close
			conn.SetCloseHandler(
				func(code int, text string) error {
					delete(clients[storeId], conn)
					return nil
				})
		}
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "websockets.html")
	})

	http.ListenAndServe(fmt.Sprintf(":%s", cfg.HTTPPort), nil)

}

func broadcastMessageByStoreID(clients map[*websocket.Conn]bool, message []byte) {
	log.Printf("Broadcasting message to %d clients", len(clients))
	for client := range clients {
		if err := client.WriteMessage(websocket.TextMessage, message); err != nil {
			return
		}
	}
}

// type Notification struct {
// 	Message  string `json:"message"`
// 	FCMToken string `json:"fcm_token"`
// 	Data     string `json:"data,omitempty"`
// 	Topic    string `json:"topic,omitempty"`
// }

func consumeKafka(clients map[string]map[*websocket.Conn]bool) {
	conf := ReadConfig()

	topic := "sales"

	// sets the consumer group ID and offset
	conf["group.id"] = "go-group-1"
	// conf["auto.offset.reset"] = "earliest"
	conf["auto.offset.reset"] = "latest"

	// creates a new consumer and subscribes to your topic
	consumer, _ := kafka.NewConsumer(&conf)
	consumer.SubscribeTopics([]string{topic}, nil)
	// var websocket Notification
	run := true
	for run {
		// consumes messages from the subscribed topic and prints them to the console
		e := consumer.Poll(1000)
		switch ev := e.(type) {
		case *kafka.Message:
			// application-specific processing

			// err := json.Unmarshal(ev.Value, &websocket)
			// if err != nil {
			// 	fmt.Println("Error unmarshalling JSON: ", err)
			// }

			// fmt.Fprintf(os.Stderr, "%% Error: %v\n", ev)
			// run = false

			//check for store id in the message
			//broadcast message to all clients connected to that store id

			var message map[string]interface{}
			err := json.Unmarshal(ev.Value, &message)
			log.Printf("Message: %s", message)
			if err != nil {
				fmt.Println("Error unmarshalling JSON: ", err)
			}

			storeId := message["store_id"].(string)
			// clients := make(map[*websocket.Conn]bool)
			if clients[storeId] != nil {
				broadcastMessageByStoreID(clients[storeId], ev.Value)
			}

		}
	}

	// closes the consumer connection
	consumer.Close()

}
