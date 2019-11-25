/**
 ** Copyright 2019 by Cratos Network, a project from Aquarelle AI
**/
package crawlers

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/aquarelle-tech/darkmatter/types"
	"github.com/gorilla/websocket"
)

const (
	// BinanceModuleName is the public name for the crawler
	BinanceModuleName = "Binance REST API"
	// BinanceAPIUrl is the url used to connect to the stream of Binance
	BinanceAPIUrl = "https://api.binance.com/api/v3/ticker/24hr?symbol=BTCUSDT"

	wssBaseURL   = "wss://stream.binance.com:9443/ws"
	usdbtcSymbol = "btcusdt"

	// WebsocketTimeout is an interval for sending ping/pong messages
	WebsocketTimeout = time.Second * 60
)

var (
	volume       = 0.0
	highPrice    = 0.0
	lowPrice     = math.MaxFloat64
	asksSummary  = 0.0
	bidsSummary  = 0.0
	highAsk      = 0.0
	highBid      = 0.0
	priceSummary = 0.0
	bidsCount    = 0 // How many bids in the time interval
	asksCount    = 0 // How many asks in the time interval
)

// BinanceMarketDepthEvent is the "official" message from binance
type BinanceMarketDepthEvent struct {
	EventType     string     `json:"e"` // Event type
	Timestamp     uint64     `json:"E"` // Event time
	Symbol        string     `json:"s"` // Symbol
	FirstUpdateID int64      `json:"U"` // First update ID in event
	LastUpdateID  int64      `json:"u"` // Final update ID in event
	Bids          [][]string `json:"b"` // Bids to be updated
	Asks          [][]string `json:"a"` // Asks to be updated
}

// BinanceCrawler is the REST API client to get data from Binance
type BinanceCrawler struct {
	DataCrawler Crawler
	Symbol      string

	events chan types.ExchangeMarketEvent // Channel to publish the events from the Binance´s stream
}

// NewBinanceCrawler creates a new crawler
func NewBinanceCrawler() BinanceCrawler {
	crawler := NewCrawler(BinanceAPIUrl) // Generic crawler

	binanceCrawler := BinanceCrawler{
		DataCrawler: crawler,
	}
	go binanceCrawler.connectToStream()

	return binanceCrawler
}

func (crawler *BinanceCrawler) connectToStream() {

	endpoint := fmt.Sprintf("%s/%s@depth", wssBaseURL, strings.ToLower(usdbtcSymbol))
	log.Println(fmt.Sprintf("Connecting stream to %s", endpoint))

	c, response, err := websocket.DefaultDialer.Dial(endpoint, nil)
	if err != nil {
		log.Fatal("dial:", err)
	}
	log.Println(response)
	// crawler.events = make(chan types.ExchangeMarketEvent, 1000)

	// Create a ticker to calculate the aggregations and initialize the counters
	collectorTicker := time.NewTicker(time.Second) // Each second
	collectorDone := make(chan bool)
	go func() {
		for {
			select {
			case <-collectorDone:
				return
			case <-collectorTicker.C:
				bidAskSpread := (highBid - highAsk) / highAsk

				log.Printf("\n=====\nAsks: %f, Bids: %f, Spread: %f\nHigh: %f, Low: %f\nHigh Bid: %f, High: Ask: %f \n===\n\n",
					asksSummary,
					bidsSummary,
					bidAskSpread,
					highPrice,
					lowPrice,
					highBid,
					highAsk,
				)
				// Reset all the counters and aggregators
				asksSummary = 0.0
				bidsSummary = 0.0
				bidsCount = 0
				asksCount = 0
				highPrice = 0.0
				lowPrice = math.MaxFloat64
				highAsk = 0.0
				highBid = 0.0
			}
		}
	}()

	defer func() {
		c.Close()
		collectorDone <- true
	}()

	handleMessages(c)
}

// handleMessages will loop infinitely reading the messages from Binance
func handleMessages(c *websocket.Conn) {
	// Get message in an infinite loop
	for {
		msgType, message, err := c.ReadMessage()
		if err != nil {
			log.Println("read:", err)
			return
		}
		log.Printf("MSG recv: type=%d\n", msgType)

		var wsEvent BinanceMarketDepthEvent
		err = json.Unmarshal(message, &wsEvent)
		if err != nil {
			log.Fatal(fmt.Sprintf("Cannot serialize the message: %s", string(message)))
		}

		// Sum all the prices
		for i := 0; i < len(wsEvent.Bids); i++ {
			price, _ := strconv.ParseFloat(wsEvent.Bids[i][0], 64)
			qty, _ := strconv.ParseFloat(wsEvent.Bids[i][1], 64)

			bid := price * qty
			bidsSummary += bid
			volume -= qty
			priceSummary += price
			asksCount++
			highPrice = math.Max(highPrice, price)
			lowPrice = math.Min(lowPrice, price)
			highBid = math.Max(highBid, bid)
		}
		for i := 0; i < len(wsEvent.Asks); i++ {
			price, _ := strconv.ParseFloat(wsEvent.Asks[i][0], 64)
			qty, _ := strconv.ParseFloat(wsEvent.Asks[i][1], 64)

			ask := price * qty
			asksSummary += ask
			volume += qty
			priceSummary += price
			bidsCount++
			highPrice = math.Max(highPrice, price)
			lowPrice = math.Min(lowPrice, price)
			highAsk = math.Max(highAsk, ask)
		}

		// unifiedEvent := types.ExchangeMarketEvent{
		// 	ExchangeName:  "binance",
		// 	Symbol:        usdbtcSymbol,
		// 	FirstUpdateID: wsEvent.FirstUpdateID,
		// 	LastUpdateID:  wsEvent.LastUpdateID,
		// 	Bids:          wsEvent.Bids,
		// 	Asks:          wsEvent.Asks,
		// }
		// crawler.events <- unifiedEvent
		// log.Printf("recv: %s", unifiedEvent)
	}

}

// keepAlive will send the pong reply to Binance when the service request a signal to no ternminate the connection
func keepAlive(c *websocket.Conn, timeout time.Duration) {
	ticker := time.NewTicker(timeout)
	keepAlive(c, WebsocketTimeout)

	lastResponse := time.Now()
	c.SetPongHandler(func(msg string) error {
		lastResponse = time.Now()
		return nil
	})

	go func() {
		defer ticker.Stop()
		for {
			deadline := time.Now().Add(10 * time.Second)
			err := c.WriteControl(websocket.PingMessage, []byte{}, deadline)
			if err != nil {
				return
			}
			<-ticker.C
			if time.Now().Sub(lastResponse) > timeout {
				c.Close()
				return
			}
		}
	}()
}

// Return the name of this crawler
func (c BinanceCrawler) GetName() string {
	return BinanceModuleName
}

func (c BinanceCrawler) GetTicker() string {
	return c.Symbol
}

// Serializes a json to a TickerInfo24 type
func (c BinanceCrawler) ToQuotePriceInfo(jsonData []byte) types.QuotePriceInfo {

	var result types.QuotePriceInfo
	aux := struct {
		Volume      string `json:"volume"`
		QuoteVolume string `json:"quoteVolume"`
		HighPrice   string `json:"highPrice"`
		OpenPrice   string `json:"openPrice"`
	}{}

	if err := json.Unmarshal(jsonData, &aux); err != nil {
		panic(err)
	}

	result = types.QuotePriceInfo{}
	result.Volume, _ = strconv.ParseFloat(aux.Volume, 32)
	result.QuoteVolume, _ = strconv.ParseFloat(aux.QuoteVolume, 32)
	result.HighPrice, _ = strconv.ParseFloat(aux.HighPrice, 32)
	result.OpenPrice, _ = strconv.ParseFloat(aux.OpenPrice, 32)

	return result
}

// Set the ticker name according the quoted currency requested
func (c BinanceCrawler) SetTicker(quotedCurrency string) {

	switch quotedCurrency {
	case "USD":
		c.Symbol = "BTCUSDT"
	}
}

// Helper function to convert the json from Binance´s API to a QuotePriceInfo instance
func (c BinanceCrawler) Crawl(quotedCurrency string, done chan types.QuotePriceInfo) {

	c.SetTicker(quotedCurrency)
	jsonData, err := c.DataCrawler.Get()
	if err != nil {
		return
	}

	priceInfo := c.ToQuotePriceInfo(jsonData)
	priceInfo.Timestamp = time.Now().Unix()
	priceInfo.DataURL = BinanceAPIUrl
	priceInfo.ExchangeUID = "binance"

	done <- priceInfo
}
