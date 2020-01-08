/**
 ** Copyright 2019 by Cratos Network, a project from Aquarelle Tech
**/

// Package crawlers contains the functions to get the data from Liquid Tap Services
package crawlers

import (
	"encoding/json"
	"math"
	"strconv"
	"time"

	cctypes "github.com/aquarelle-tech/darkmatter/x/cryptocerts/internal/types"
	"github.com/golang/glog"
	"github.com/gorilla/websocket"
)

const (
	// LiquidUID is the unique name for Liquid Exchange used in all the messages
	LiquidUID = "liquid"
	// LiquidWebsocketAPIUrl base URL address for the Liquid Tap Services
	LiquidWebsocketAPIUrl = "wss://tap.liquid.com/app/LiquidTapClient"
)

// LiquidWSEvent holds the data serialized from the websocket messages
type LiquidWSEvent struct {
	Channel string      `json:"channel"`
	Data    interface{} `json:"data"`
	Event   string      `json:"event"`
}

var (
	keepAliveTicker *time.Ticker
	keepAliveDone   chan bool
)

// LiquidCrawler is the REST API client to get data from Liquid
type LiquidCrawler struct {
	*GenericCrawler
}

// NewLiquidCrawler creates a new crawler
func NewLiquidCrawler(publication chan cctypes.QuotePriceData) LiquidCrawler {

	c := LiquidCrawler{
		GenericCrawler: NewGenericExchangeCrawler(LiquidWebsocketAPIUrl, publication),
	}
	// work around to "create" the methods overriding in  struct
	c.serializeEvent = c.SerializeLiquidEvent
	c.closeConnection = c.CloseConnection
	c.initializeConnection = c.InitializeLiquidConnection
	c.getUID = c.GetUID

	c.Start()

	return c
}

// CloseConnection stops the timer
func (crawler *LiquidCrawler) CloseConnection(c *websocket.Conn) {

	keepAliveTicker.Stop()
	keepAliveDone <- true
}

// InitializeLiquidConnection creates the timer to send a Ping every 60 seconds to the Liquid services to show activity
func (crawler *LiquidCrawler) InitializeLiquidConnection(c *websocket.Conn) {

	keepAliveTicker := time.NewTicker(60 * time.Second) // Each 60 seconds
	keepAliveDone := make(chan bool)

	go func() {
		for {
			select {
			case <-keepAliveDone:
				return
			case <-keepAliveTicker.C:
				pingData := []byte("{\"event\":\"pusher:ping\",\"data\":{}}")
				err := c.WriteControl(websocket.PingMessage, pingData, time.Now().Add(time.Second*30))
				if err != nil {
					glog.Errorf("Error sending PING message to the Liquid Tap Services. %v", err)
				}
				glog.Info("Sent a PING message to Liquid Tap Services")
			}
		}
	}()
}

// SerializeLiquidEvent extracts the data from the message sent by the base (generic) class
func (crawler *LiquidCrawler) SerializeLiquidEvent(message []byte, c *websocket.Conn) error {
	// {"data":"{\"activity_timeout\":120,\"socket_id\":\"1697574627.7741218362\"}","event":"pusher:connection_established"}

	var ev LiquidWSEvent
	if err := json.Unmarshal(message, &ev); err != nil {
		glog.Errorf("Can´t read messages from Liquid´s data stream. %v", err)
		return err
	}
	switch ev.Event {
	case "pusher:connection_established":
		// Subscription to both channel: buy and send
		c.WriteMessage(websocket.TextMessage, []byte("{\"event\":\"pusher:subscribe\",\"data\":{\"channel\":\"price_ladders_cash_btcusd_sell\"}}"))
		c.WriteMessage(websocket.TextMessage, []byte("{\"event\":\"pusher:subscribe\",\"data\":{\"channel\":\"price_ladders_cash_btcusd_buy\"}}"))
		break
	case "updated":

		switch ev.Channel {
		case "price_ladders_cash_btcusd_sell":
			var bidData [][]string
			var validBids [][]string

			if err := json.Unmarshal([]byte(ev.Data.(string)), &bidData); err != nil {
				glog.Errorf("Can´t get the data from the Liquid message. %v", err)
				return err
			}

			for i := 0; i < len(bidData); i++ {
				price, _ := strconv.ParseFloat(bidData[i][0], 64)
				qty, _ := strconv.ParseFloat(bidData[i][1], 64)

				if qty > 0 {
					crawler.bidPrice += price
					crawler.bidQty += qty
					crawler.lowBid = math.Min(crawler.lowBid, price)
					crawler.highBid = math.Max(crawler.highBid, price)

					validBids = append(validBids, bidData[i])
				}
			}
			crawler.bidVolume += int64(len(validBids))
			// Store the evidence
			crawler.evidence = append(crawler.evidence, cctypes.QuotePriceEvidence{
				Bids:        validBids,
				Timestamp:   uint64(time.Now().Unix()),
				ExchangeUID: crawler.GetUID(),
			})

			break
		case "price_ladders_cash_btcusd_buy":
			var askData [][]string
			var validAsks [][]string

			if err := json.Unmarshal([]byte(ev.Data.(string)), &askData); err != nil {
				glog.Errorf("Can´t get the data from the Liquid message. %v", err)
				return err
			}

			for i := 0; i < len(askData); i++ {
				price, _ := strconv.ParseFloat(askData[i][0], 64)
				qty, _ := strconv.ParseFloat(askData[i][1], 64)

				if qty > 0 { // Avoid to inclide non-valid data
					crawler.askPrice += price
					crawler.askQty += qty
					crawler.lowAsk = math.Min(crawler.lowAsk, price)
					crawler.highAsk = math.Max(crawler.highAsk, price)

					validAsks = append(validAsks, askData[i])
				}
			}
			crawler.askVolume += int64(len(validAsks))
			// Store the evidence

			crawler.evidence = append(crawler.evidence, cctypes.QuotePriceEvidence{
				Bids:        validAsks,
				Timestamp:   uint64(time.Now().Unix()),
				ExchangeUID: crawler.GetUID(),
			})

			break
		}

	}

	return nil
}

// GetUID return the name of this crawler
func (crawler LiquidCrawler) GetUID() string {
	return LiquidUID
}
