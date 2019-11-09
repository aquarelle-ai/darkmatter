/**
 ** Copyright 2019 by Cratos Network, a project from Aquarelle AI
**/
package main

import (
	"log"
	"net/http"

	"cratos.network/darkmatter/crawlers"
	"cratos.network/darkmatter/mapreduce"
	"cratos.network/darkmatter/types"
	"github.com/gorilla/websocket"
)

// List of available crawlers
var directory = []types.PriceSourceCrawler{
	crawlers.NewBinanceCrawler(),
	crawlers.NewLiquidCrawler(),
	crawlers.NewBitfinexCrawler(),
}

var clients = make(map[*websocket.Conn]bool)  // connected clients
var broadcast = make(chan types.PriceMessage) // Broadcast channel
var publishedPrices = make(chan types.PriceMessage)
var upgrader = websocket.Upgrader{}

// Read from the broadcast channel
func broadcastMessages() {
	for {
		// Grab the next message from the broadcast channel
		msg := <-broadcast

		// Send it out to every client that is currently connected
		for client := range clients {
			err := client.WriteJSON(msg)

			// If client is not longer listening or any other error, the client is removed from the list
			if err != nil {
				log.Printf("Error writing to a client: %v", err)
				client.Close()
				delete(clients, client)
			}
		}
	}
}

// Esta función manejará nuestras conexiones WebSocket entrantes
func handleListeners(w http.ResponseWriter, r *http.Request) {

	// El método Upgrade() permite cambiar nuesra solicitud GET inicial a una completa en WebSocket, si hay un error lo mostramos en consola pero no salimos.
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Fatal(err)
	}

	// Para cerrar la conexión una vez termina la función
	defer ws.Close()

	// Registramos nuestro nuevo cliente al agregarlo al mapa global de "clients" que fue creado anteriormente.
	clients[ws] = true

	// An infinite loop to get the published messages from the data processors ad send to the broadcast queue
	for {
		msg := <-publishedPrices
		log.Printf("MESSAGE: Volume=%f, HighPrice=%f", msg.AverageVolume, msg.AveragePrice)

		// Read in a new message as JSON and map it to a Message object
		// Si hay un error, registramos ese error y eliminamos ese cliente de nuestro mapa global de clients

		// Send the newly received message to the broadcast channel
		broadcast <- msg
	}
}

func main() {

	quotedCurrency := "USD"

	// Para que al entrar en la aplicación el cliente tome el index.html
	fs := http.FileServer(http.Dir("./public"))
	http.Handle("/", fs)

	// "/price" es la ruta que nos ayudará a manejar cualquier solicitud para iniciar un WebSocket.
	http.HandleFunc("/price", handleListeners)

	// A subrouting to handle messages
	go broadcastMessages()

	go mapreduce.MapReduceLoop(directory, quotedCurrency, publishedPrices)

	// Iniciamos el servidor en localhost en el puerto 8080 y un mensaje que muetre por consola
	log.Println("http server started on :8080")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}

}
