/**
 ** Copyright 2019 by Cratos Network, a project from Aquarelle AI
**/
package service

import (
	"fmt"
	"log"
	"net/http"

	"aquarelle.ai/darkmatter/types"
	"github.com/gorilla/websocket"
)

const (
	PUBLIC_DIRECTORY_PATH = "./public"
)

var clients = make(map[*websocket.Conn]bool)           // connected clients
var broadcast = make(chan types.LiteIndexValueMessage) // Broadcast channel
var upgrader = websocket.Upgrader{}

type OracleServer struct {
	// Channel to se
	Published chan types.FullSignedBlock
	Broadcast chan types.LiteIndexValueMessage
	Clients   map[*websocket.Conn]bool
}

func NewOracleServer(published chan types.FullSignedBlock) OracleServer {
	return OracleServer{
		Published: published,
		Broadcast: broadcast,
		Clients:   clients,
	}
}

// Read from the broadcast channel
func (o OracleServer) broadcastMessages() {
	for {
		// Grab the next message from the broadcast channel
		msg := <-o.Broadcast

		// Send it out to every client that is currently connected
		for client := range o.Clients {
			err := client.WriteJSON(msg)

			// If client is not longer listening or any other error, the client is removed from the list
			if err != nil {
				log.Printf("Error writing to a client: %v", err)
				client.Close()
				delete(o.Clients, client)
			}
		}
	}
}

// This function will receive and register all the new listeners
func (o OracleServer) handlePriceListeners(w http.ResponseWriter, r *http.Request) {

	// Try to upgrade the connection. If it fails, the log, but not break the execution
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Fatal(err)
	}

	// Para cerrar la conexión una vez termina la función
	defer ws.Close()

	// Register a new listener
	o.Clients[ws] = true

	// An infinite loop to get the published messages from the data processors ad send to the broadcast queue
	for {
		msg := <-o.Published // Get a message from the public queue
		log.Printf("MESSAGE: Volume=%f, HighPrice=%f", msg.AverageVolume, msg.AveragePrice)

		liteMessage := types.LiteIndexValueMessage{
			Height:      msg.Height,
			PriceIndex:  msg.AveragePrice,
			Quoted:      msg.Ticker,
			NodeAddress: fmt.Sprintf("http://localhost/node/%s", msg.Hash),
			Timestamp:   msg.Timestamp,
		}

		// Send the newly received message to the broadcast channel
		o.Broadcast <- liteMessage
	}
}

// Prepare and start the main routines
func (o OracleServer) Initialize() {
	// To send back a html page by default
	fs := http.FileServer(http.Dir(PUBLIC_DIRECTORY_PATH))
	http.Handle("/", fs)

	// The main route to get the websocket path
	http.HandleFunc("/price", o.handlePriceListeners)

	// Launch subrouting to handle messages
	go o.broadcastMessages()
}
