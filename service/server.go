/**
 ** Copyright 2019 by Cratos Network, a project from Aquarelle Tech
**/
package service

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/aquarelle-tech/darkmatter/database"
	"github.com/aquarelle-tech/darkmatter/types"
	"github.com/golang/glog"
	"github.com/gorilla/websocket"
)

const (
	APIVersion = "v1"
)

var (
	// ServerInstance keeps the instance of current server
	ServerInstance *OracleServer

	clients  = make(map[*websocket.Conn]bool) // connected clients
	upgrader = websocket.Upgrader{}
)

// Error represents a handler error. It provides methods for a HTTP status
// code and embeds the built-in error interface.
type Error interface {
	error
	Status() int
}

// OracleServer is the main interface to publish the functions to access the framework
type OracleServer struct {
	// Channel to se
	Publication chan types.FullSignedBlock
	// Broadcast chan types.LiteIndexValueMessage
	Broadcast chan types.FullSignedBlock
	Clients   map[*websocket.Conn]bool
}

// NewOracleServer is the constructor
func NewOracleServer() *OracleServer {

	if ServerInstance != nil {
		return ServerInstance
	}

	ServerInstance = &OracleServer{
		Publication: make(chan types.FullSignedBlock),
		Broadcast:   make(chan types.FullSignedBlock),
		Clients:     clients,
	}

	return ServerInstance
}

// Initialize will prepare and start the main routines
func (o OracleServer) Initialize() {

	http.HandleFunc(fmt.Sprintf("/%s/socket/latest", APIVersion), o.handleWebSocketConnections)
	http.HandleFunc(fmt.Sprintf("/%s/chain", APIVersion), o.getBlock)
	http.HandleFunc(fmt.Sprintf("/%s/latest", APIVersion), o.getLatestBlocks)

	// Launch subrouting to handle messages
	go o.broadcastMessages()
}

// AddBlock is an utility function to add a new block with a payload to the blockchain and broadcast the message to
// all the active subscribers
func (o OracleServer) AddBlock(payload interface{}, evidence interface{}) error {

	if ServerInstance == nil {
		msg := "The server instance has not been initialized."
		glog.Fatal(msg)
		panic(msg)
	}

	block, err := database.PublicBlockDatabase.NewFullSignedBlock(payload, evidence, "") // TODO: Add the memo info, if any
	if err != nil {
		glog.Fatalf("The database cannot store a new block. %v", err)
		panic(err)
	}

	// Add the message to the broadcast queue
	o.Publication <- block

	return nil
}

// broadcastMessages will read from the broadcast channel and send the message to every registered client
func (o OracleServer) broadcastMessages() {
	for {
		// Grab the next message from the broadcast channel
		msg := <-o.Broadcast

		// Send it out to every client that is currently connected
		for client := range o.Clients {
			err := client.WriteJSON(msg)

			// If client is not longer listening or any other error, the client is removed from the list
			if err != nil {
				glog.Infof("Error writing to a client: %v", err)
				client.Close()
				delete(o.Clients, client)
			}
		}
	}
}

// setupResponse initialize the headers in each request
func setupResponse(w *http.ResponseWriter, req *http.Request) {
	(*w).Header().Set("Access-Control-Allow-Origin", "*")
	(*w).Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	(*w).Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
}

func checkGETOnly(w http.ResponseWriter, r *http.Request) bool {
	if r.Method != "GET" {
		errorMsg := fmt.Sprintf("HTTP Error: Invalid method '%s'", r.Method)
		log.Println(errorMsg)
		http.Error(w, errorMsg, http.StatusNotAcceptable)
		return false
	}

	return true
}

// getLatestBlocks returns the latest 10 blocks
func (o OracleServer) getLatestBlocks(w http.ResponseWriter, r *http.Request) {
	setupResponse(&w, r)
	if !checkGETOnly(w, r) {
		return
	}
	// Return the latest blocks, starting now
	blocks, err := database.PublicBlockDatabase.Store.GetLatestBlocks(uint64(time.Now().Unix()), 10)
	if err != nil {
		errorMsg := "The server can´t recover the requested blocks"
		glog.Infof("HTTP Error: %s. Reason: %s", errorMsg, err)
		http.Error(w, errorMsg, http.StatusInternalServerError)
		return
	}
	response, err := json.Marshal(blocks)

	w.Write(response)
}

// getBlock will return a block from the blockchain
func (o OracleServer) getBlock(w http.ResponseWriter, r *http.Request) {

	setupResponse(&w, r)
	if !checkGETOnly(w, r) {
		return
	}

	hash := r.URL.Query().Get("hash")
	timestampParam := r.URL.Query().Get("timestamp")
	heightParam := r.URL.Query().Get("height")

	var block *types.FullSignedBlock
	var err error

	if len(hash) == 0 && len(timestampParam) == 0 && len(heightParam) == 0 {
		errorMsg := "Can´t find an acceptable parameter to look for a block"
		glog.Infof("HTTP Error: %s", errorMsg)
		http.Error(w, errorMsg, http.StatusBadRequest)
		return
	}

	if len(hash) != 0 { // If there is a hash...
		block, err = database.PublicBlockDatabase.Store.GetBlock(hash)
		if err != nil {
			errorMsg := fmt.Sprintf("Don´t exists a block for the hash '%s'", hash)
			glog.Infof("HTTP Error: %s", errorMsg)
			http.Error(w, errorMsg, http.StatusNotFound)
			return
		}
	} else if len(timestampParam) > 0 { // if there is a timestamp
		timestamp, err := strconv.ParseUint(timestampParam, 10, 64)
		if err != nil {
			errorMsg := fmt.Sprintf("Invalid value '%s' for the timestamp.", timestampParam)
			glog.Infof("HTTP Error: %s", errorMsg)
			http.Error(w, errorMsg, http.StatusBadRequest)
			return
		}
		block, err = database.PublicBlockDatabase.Store.FindBlockByTimestamp(timestamp)
	} else if len(heightParam) > 0 { // if there is a height
		height, err := strconv.ParseUint(heightParam, 10, 64)
		if err != nil {
			errorMsg := fmt.Sprintf("Invalid value '%s' for the height", heightParam)
			glog.Infof("HTTP Error: %s", errorMsg)
			http.Error(w, errorMsg, http.StatusBadRequest)
			return
		}
		block, err = database.PublicBlockDatabase.Store.FindBlockByHeight(height)
	}

	response, err := json.MarshalIndent(block, "", " ")
	if err != nil {
		errorMsg := "The stored block information is corrupted. Please check this error with the node manager"
		glog.Infof("HTTP Error: %s", errorMsg)
		http.Error(w, errorMsg, http.StatusNotAcceptable)
		return
	}

	w.Write(response)
}

// handleWebSocketConnections will receive and register all the new listeners
func (o OracleServer) handleWebSocketConnections(w http.ResponseWriter, r *http.Request) {

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

	// An infinite loop to get the published messages from the data processors and send to the broadcast queue
	for {
		// Get a message from the public queue
		msg := <-o.Publication
		// TODO: Apply any other process to the message
		// Send the newly received message to the broadcast channel
		o.Broadcast <- msg
	}
}
