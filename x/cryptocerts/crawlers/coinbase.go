/**
 ** Copyright 2019 by Cratos Network, a project from Aquarelle Tech
**/
package crawlers

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"time"

	cctypes "github.com/aquarelle-tech/darkmatter/x/cryptocerts/internal/types"
	"github.com/golang/glog"
	"github.com/gorilla/websocket"
)

const (
	CoinbaseUID             = "coinbase"
	CoinbaseWebsocketAPIUrl = "wss://ws-feed.pro.coinbase.com"
	CoinbaseUSDBTCSymbol    = "BTC-USD"
)

type coinbaseEventType struct {
	Type string `json:"type"`
}

type coinbaseMatchEvent struct {
	coinbaseEventType

	Side    string `json:"side"`
	Time    string `json:"time"`
	Reason  string `json:"reason"`
	Price   string `json:"price"`
	Size    string `json:"size"`
	OrderID string `json:"order_id"`
}

// CoinbaseCrawler implements the  REST API client to get data from Coinbase
type CoinbaseCrawler struct {
	*GenericCrawler
}

var (
	coinbaseOrderSizes = make(map[string]*float64)
)

// NewCoinbaseCrawler creates a new crawler
func NewCoinbaseCrawler(publication chan cctypes.QuotePriceData) CoinbaseCrawler {

	c := CoinbaseCrawler{
		GenericCrawler: NewGenericExchangeCrawler(CoinbaseWebsocketAPIUrl, publication),
	}

	// work around to "create" the methods overriding in  struct
	c.serializeEvent = c.SerializeCoinbaseEvent
	c.initializeConnection = c.InitializeCoinbaseConnection
	c.getUID = c.GetUID

	c.Start()

	return c
}

// InitializeCoinbaseConnection send the subscription messages
func (crawler *CoinbaseCrawler) InitializeCoinbaseConnection(c *websocket.Conn) {

	subscriptionMessage := fmt.Sprintf(`
	{
		"type": "subscribe", 
		"channels" : [{"name" : "full", "product_ids" : ["%s"]}]
	}`, CoinbaseUSDBTCSymbol)

	if err := c.WriteMessage(websocket.TextMessage, []byte(subscriptionMessage)); err != nil {
		glog.Error("Can´t start the subscription to Coinbase API. %v", err)
	}
}

// SerializeCoinbaseEvent transform the message received from Coinbase
func (crawler *CoinbaseCrawler) SerializeCoinbaseEvent(message []byte, c *websocket.Conn) error {

	// glog.Infof("COINBASE ====================================> %s", message)

	matchEvent := coinbaseMatchEvent{}
	if err := json.Unmarshal(message, &matchEvent); err != nil {
		glog.Errorf("Coinbase: Error unmarshaling message. %v", err)
		return err
	}

	switch matchEvent.Type {
	case "received":
		// Store the value for the  next event: "done"
		qty, _ := strconv.ParseFloat(matchEvent.Size, 64)
		coinbaseOrderSizes[matchEvent.OrderID] = &qty
		break
	case "done":
		var price, qty float64

		if coinbaseOrderSizes[matchEvent.OrderID] != nil {
			qty = *coinbaseOrderSizes[matchEvent.OrderID]
			// Remove the value because is not longer needed
			delete(coinbaseOrderSizes, matchEvent.OrderID)
		}

		if matchEvent.Reason == "canceled" || qty == 0 {
			return nil // Canceled or empty orders are not included in the algorithm
		}
		// Get the price from the "done" order
		price, _ = strconv.ParseFloat(matchEvent.Price, 64)

		if matchEvent.Side == "sell" {
			crawler.bidPrice += price
			crawler.bidQty += qty
			crawler.lowBid = math.Min(crawler.lowBid, price)
			crawler.highBid = math.Max(crawler.highBid, price)

			crawler.bidVolume++
			// Store the evidence
			crawler.evidence = append(crawler.evidence, cctypes.QuotePriceEvidence{
				Bids:        [][]string{{fmt.Sprintf("%f", price), fmt.Sprintf("%f", qty)}},
				Timestamp:   uint64(time.Now().Unix()),
				ExchangeUID: crawler.GetUID(),
			})
		} else {
			crawler.askPrice += price
			crawler.askQty += math.Abs(qty)
			crawler.lowAsk = math.Min(crawler.lowAsk, price)
			crawler.highAsk = math.Max(crawler.highAsk, price)

			crawler.askVolume++
			// Store the evidence
			crawler.evidence = append(crawler.evidence, cctypes.QuotePriceEvidence{
				Asks:        [][]string{{fmt.Sprintf("%f", price), fmt.Sprintf("%f", qty)}},
				Timestamp:   uint64(time.Now().Unix()),
				ExchangeUID: crawler.GetUID(),
			})
		}

	}

	return nil
}

// GetUID return the name of this crawler
func (crawler CoinbaseCrawler) GetUID() string {
	return CoinbaseUID
}
