/**
 ** Copyright 2019 by Cratos Network, a project from Aquarelle Tech
**/
package crawlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"time"
	"unicode"

	cctypes "github.com/aquarelle-tech/darkmatter/x/cryptocerts/internal/types"
	"github.com/golang/glog"
	"github.com/gorilla/websocket"
)

const (
	BitfinexUID             = "bitfinex"
	BitfinexWebsocketAPIUrl = "wss://api-pub.bitfinex.com/ws/2"
	BitfinexUSDBTCSymbol    = "tBTCUSD"
)

// BitfinexCrawler implements the  REST API client to get data from Bitfinex
type BitfinexCrawler struct {
	*GenericCrawler
}

var (
	orderBookReceived = false
)

// NewBitfinexCrawler creates a new crawler
func NewBitfinexCrawler(publication chan cctypes.QuotePriceData) BitfinexCrawler {

	c := BitfinexCrawler{
		GenericCrawler: NewGenericExchangeCrawler(BitfinexWebsocketAPIUrl, publication),
	}

	// work around to "create" the methods overriding in  struct
	c.serializeEvent = c.SerializeBitfinexEvent
	c.initializeConnection = c.InitializeBitfinexConnection
	c.getUID = c.GetUID

	c.Start()

	return c
}

// InitializeBitfinexConnection send the subscription messages
func (crawler *BitfinexCrawler) InitializeBitfinexConnection(c *websocket.Conn) {

	subscriptionMessage := fmt.Sprintf("{\"event\": \"subscribe\", \"channel\": \"book\", \"prec\": \"R0\", \"len\": 100, \"symbol\" : \"%s\"}", BitfinexUSDBTCSymbol)
	if err := c.WriteMessage(websocket.TextMessage, []byte(subscriptionMessage)); err != nil {
		glog.Error("Can´t start the subscription to Bitfinex API. %v", err)
	}
}

// SerializeBitfinexEvent transform the message received from Bitfinex
func (crawler *BitfinexCrawler) SerializeBitfinexEvent(message []byte, c *websocket.Conn) error {

	jsonMsg := bytes.TrimLeftFunc(message, unicode.IsSpace) // remove the first spaces to ensure to get a valid char in the firs pos
	err := error(nil)

	// either a channel data array or an event object, raw json encoding
	if bytes.HasPrefix(jsonMsg, []byte("[")) {
		err = crawler.parseMessage(message)
	} else if bytes.HasPrefix(jsonMsg, []byte("{")) {
		return nil
	} else {
		errorMsg := fmt.Sprintf("Unexpected message from Bitfinex websocket: %s", message)
		glog.Error(errorMsg)
		return nil
	}

	return err
}

func (crawler *BitfinexCrawler) parseMessage(message []byte) error {

	// The first message after the connection is an snapshot of the order book
	if !orderBookReceived {
		orderBookReceived = true
		return nil
	}

	var raw []interface{}
	if err := json.Unmarshal(message, &raw); err != nil {
		glog.Errorf("Cannot unserialize the Bitfinex´s data message! %v", err)
		return err
	}

	switch data := raw[1].(type) {
	case string:
		switch data {
		case "hb":
			// no-op
			return nil
		case "cs":
			// TODO: Manage checksums! no-op for now
			return nil
			// if checksum, ok := raw[2].(float64); ok {
			// 	return c.handleChecksumChannel(chanID, int(checksum))
			// } else {
			// 	c.log.Error("Unable to parse checksum")
			// }
		default:
			glog.Warning("Bitfinex: Message type not managed! %s", message)
			return nil
		}
	case []interface{}:

		price := data[1].(float64)
		qty := data[2].(float64)

		if price > 0 {
			if qty > 0 { // bids
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

			} else { // asks
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
func (crawler BitfinexCrawler) GetUID() string {
	return BitfinexUID
}
