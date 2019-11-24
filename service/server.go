/**
 ** Copyright 2019 by Cratos Network, a project from Aquarelle AI
**/
package service

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/aquarelle-tech/darkmatter/database"
	"github.com/aquarelle-tech/darkmatter/types"
	"github.com/gorilla/websocket"
)

const (
	APIVersion = "v1"
)

var (
	clients   = make(map[*websocket.Conn]bool)         // connected clients
	broadcast = make(chan types.LiteIndexValueMessage) // Broadcast channel
	upgrader  = websocket.Upgrader{}
)

// Error represents a handler error. It provides methods for a HTTP status
// code and embeds the built-in error interface.
type Error interface {
	error
	Status() int
}

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

func setupResponse(w *http.ResponseWriter, req *http.Request) {
	(*w).Header().Set("Access-Control-Allow-Origin", "*")
	(*w).Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	(*w).Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
}

func (o OracleServer) getBlock(w http.ResponseWriter, r *http.Request) {

	setupResponse(&w, r)
	if r.Method != "GET" {
		errorMsg := fmt.Sprintf("HTTP Error: Invalid method '%s'", r.Method)
		log.Println(errorMsg)
		http.Error(w, errorMsg, http.StatusNotAcceptable)
		return
	}

	hash := r.URL.Query().Get("hash")
	timestampParam := r.URL.Query().Get("timestamp")
	heightParam := r.URL.Query().Get("height")

	var block *types.FullSignedBlock
	var err error

	if len(hash) == 0 && len(timestampParam) == 0 && len(heightParam) == 0 {
		errorMsg := "Can´t find an acceptable parameter to look for a block"
		log.Printf("HTTP Error: %s", errorMsg)
		http.Error(w, errorMsg, http.StatusBadRequest)
		return
	}

	if len(hash) != 0 { // If there is a hash...
		block, err = database.PublicBlockDatabase.Store.GetBlock(hash)
		if err != nil {
			errorMsg := fmt.Sprintf("Don´t exists a block for the hash '%s'", hash)
			log.Printf("HTTP Error: %s", errorMsg)
			http.Error(w, errorMsg, http.StatusNotFound)
			return
		}
	} else if len(timestampParam) > 0 { // if there is a timestamp
		timestamp, err := strconv.ParseUint(timestampParam, 10, 64)
		if err != nil {
			errorMsg := fmt.Sprintf("Invalid value '%s' for the timestamp.", timestampParam)
			log.Printf("HTTP Error: %s", errorMsg)
			http.Error(w, errorMsg, http.StatusBadRequest)
			return
		}
		block, err = database.PublicBlockDatabase.Store.FindBlockByTimestamp(timestamp)
	} else if len(heightParam) > 0 { // if there is a height
		height, err := strconv.ParseUint(heightParam, 10, 64)
		if err != nil {
			errorMsg := fmt.Sprintf("Invalid value '%s' for the height", heightParam)
			log.Printf("HTTP Error: %s", errorMsg)
			http.Error(w, errorMsg, http.StatusBadRequest)
			return
		}
		block, err = database.PublicBlockDatabase.Store.FindBlockByHeight(height)
	}

	response, err := json.MarshalIndent(block, "", " ")
	if err != nil {
		errorMsg := "The stored block information is corrupted. Please check this error with the node manager"
		log.Printf("HTTP Error: %s", errorMsg)
		http.Error(w, errorMsg, http.StatusNotAcceptable)
		return
	}

	w.Write(response)
}

// This function will receive and register all the new listeners
func (o OracleServer) handlePriceListeners(w http.ResponseWriter, r *http.Request) {

	setupResponse(&w, r)
	if (*r).Method == "OPTIONS" {
		return
	}

	// Try to upgrade the connection. If it fails, the log, but not break the execution
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }
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
			Hash:          msg.Hash,
			Height:        msg.Height,
			PriceIndex:    msg.AveragePrice,
			Quoted:        msg.Ticker,
			NodeAddress:   msg.Address,
			Timestamp:     msg.Timestamp,
			Confirmations: len(msg.Evidence),
		}

		// Send the newly received message to the broadcast channel
		o.Broadcast <- liteMessage
	}
}

// Prepare and start the main routines
func (o OracleServer) Initialize() {

	// The main route to get the websocket path
	http.HandleFunc(fmt.Sprintf("/%s/socket/latest", APIVersion), o.handlePriceListeners)

	// The main route to get the websocket path
	http.HandleFunc(fmt.Sprintf("/%s/chain", APIVersion), o.getBlock)

	// Launch subrouting to handle messages
	go o.broadcastMessages()
}
