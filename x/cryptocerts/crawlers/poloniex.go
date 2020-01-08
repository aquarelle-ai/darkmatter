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
	PoloniexUID             = "poloniex"
	PoloniexWebsocketAPIUrl = "wss://api2.poloniex.com"
	PoloniexUSDBTCSymbol    = "USDT_BTC"
)

// PoloniexCrawler implements the  REST API client to get data from Poloniex
type PoloniexCrawler struct {
	*GenericCrawler
}

var (
	polonixOrderBookReceived = false
)

// NewPoloniexCrawler creates a new crawler
func NewPoloniexCrawler(publication chan cctypes.QuotePriceData) PoloniexCrawler {

	c := PoloniexCrawler{
		GenericCrawler: NewGenericExchangeCrawler(PoloniexWebsocketAPIUrl, publication),
	}

	// work around to "create" the methods overriding in  struct
	c.serializeEvent = c.SerializePoloniexEvent
	c.initializeConnection = c.InitializePoloniexConnection
	c.getUID = c.GetUID

	c.Start()

	return c
}

// InitializePoloniexConnection send the subscription messages
func (crawler *PoloniexCrawler) InitializePoloniexConnection(c *websocket.Conn) {

	subscriptionMessage := fmt.Sprintf("{\"command\": \"subscribe\", \"channel\": \"%s\"}", PoloniexUSDBTCSymbol)
	if err := c.WriteMessage(websocket.TextMessage, []byte(subscriptionMessage)); err != nil {
		glog.Error("Can´t start the subscription to Poloniex API. %v", err)
	}
}

// SerializePoloniexEvent transform the message received from Poloniex
func (crawler *PoloniexCrawler) SerializePoloniexEvent(message []byte, c *websocket.Conn) error {

	var raw []interface{}
	if err := json.Unmarshal(message, &raw); err != nil {
		glog.Errorf("Cannot unserialize the Poloniex´s data message! %v", err)
		return err
	}

	if len(raw) == 1 {
		return nil // A HB. no-op
	}

	raw = raw[2].([]interface{}) // this is the data part in the message: the first are channel_id & sequence number
	// glog.Infof("DATA LIST (%s) ============> %v", raw[0].([]interface{})[0], raw)
	// the first element is the type of message: if "i" => info (snapshot of the order book), "o": for orders, "t" for trades
	if raw[0].([]interface{})[0] != "o" {
		return nil
	}

	for i := 0; i < len(raw); i++ {
		data := raw[i].([]interface{})
		// Only process  the orders
		if data[0].(string) == "o" {
			price, _ := strconv.ParseFloat(data[2].(string), 64)
			qty, _ := strconv.ParseFloat(data[3].(string), 64)

			if qty == 0 {
				continue
			}

			// ask or bid!
			if data[1].(float64) == 1 { // Bids
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
			} else { // Asks
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
	}

	return nil
}

// GetUID return the name of this crawler
func (crawler PoloniexCrawler) GetUID() string {
	return PoloniexUID
}
