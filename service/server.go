/**
 ** Copyright 2019 by Cratos Network, a project from Aquarelle AI
**/
package service

import (
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"path/filepath"

	"aquarelle-tech/darkmatter/types"
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

func setupResponse(w *http.ResponseWriter, req *http.Request) {
	(*w).Header().Set("Access-Control-Allow-Origin", "*")
	(*w).Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	(*w).Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
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

func serveChain(w http.ResponseWriter, r *http.Request) {
	setupResponse(&w, r)

	println("Llegando a solicitar  un fichero")
	path := filepath.Join("public", filepath.Clean(r.URL.Path))

	println(path)

	if _, err := os.Stat(path); os.IsNotExist(err) {
		println("El fichero no existe")

		//log.Fatal((err))

		http.Error(w, "The requested block doesn´t exists.", http.StatusInternalServerError)

	} else {
		println("Intentando leer el fichero")
		blockContent, err := ioutil.ReadFile(path)
		if err == nil {
			// Send the blockContent
			w.Write([]byte(blockContent))

		} else {
			// log.Fatal((err))
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
	}
}

// Prepare and start the main routines
func (o OracleServer) Initialize() {
	// To send back a html page by default
	// fs := http.FileServer(http.Dir(PUBLIC_DIRECTORY_PATH))
	http.HandleFunc("/", serveChain)

	// The main route to get the websocket path
	http.HandleFunc("/price", o.handlePriceListeners)

	// Launch subrouting to handle messages
	go o.broadcastMessages()
}
