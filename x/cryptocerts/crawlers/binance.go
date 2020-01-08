package crawlers

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	cctypes "github.com/aquarelle-tech/darkmatter/x/cryptocerts/internal/types"
	"github.com/golang/glog"
	"github.com/gorilla/websocket"
)

const (
	// BinanceUID is the public nam for the crawler
	BinanceUID = "binance"
	// BinanceAPIUrl is the url used to connect to the stream of Binance
	BinanceAPIUrl = "https://api.binance.com/api/v3/ticker/24hr?symbol=BTCUSDT"

	BinanceWSBaseURL = "wss://stream.binance.com:9443/ws"
	USDBTCSymbol     = "btcusdt"

	// WebsocketTimeout is an interval or sending ping/pong messages
	WebsocketTimeout = time.Second * 60
)

// BinanceMarketDepthEvent is the "oficial" message from binance
type BinanceMarketDepthEvent struct {
	EventType     string     `json:"e"` // Event type
	Timestamp     uint64     `json:"E"` // Event ime
	Symbol        string     `json:"s"` // Symbol
	FirstUpdateID int64      `json:"U"` // First update ID in event
	LastUpdateID  int64      `json:"u"` // Final update ID inevent
	Bids          [][]string `json:"b"` // Bids to be updated
	Asks          [][]string `json:"a"` // Asks to be updated
}

// BinanceCrawler is the RES API client to get data from Binance
type BinanceCrawler struct {
	*GenericCrawler
}

// NewBinanceCrawler creates a new crawle
func NewBinanceCrawler(publication chan cctypes.QuotePriceData) BinanceCrawler {
	// webCrawler := clients.NewWebCrawler(BinanceAPIUrl) // Generic crawler

	endpoint := fmt.Sprintf("%s/%s@depth", BinanceWSBaseURL, strings.ToLower(USDBTCSymbol))

	c := BinanceCrawler{
		GenericCrawler: NewGenericExchangeCrawler(endpoint, publication),
	}

	c.serializeEvent = c.SerializeBinanceEvent
	c.getUID = c.GetUID

	c.Start()

	return c
}

func (crawler *BinanceCrawler) SerializeBinanceEvent(message []byte, c *websocket.Conn) error {
	var wsEvent BinanceMarketDepthEvent

	err := json.Unmarshal(message, &wsEvent)
	if err != nil {
		glog.Fatalf("Cannot serialize the message: %s", string(message))
		return err
	}

	var validBids [][]string
	var validAsks [][]string

	// Sum all the prices
	// bids
	for i := 0; i < len(wsEvent.Bids); i++ {
		price, _ := strconv.ParseFloat(wsEvent.Bids[i][0], 64)
		qty, _ := strconv.ParseFloat(wsEvent.Bids[i][1], 64)

		// Check: https://github.com/binance-exchange/binance-official-api-docs/blob/master/web-socket-streams.md#how-to-manage-a-local-order-book-correctly
		// 8: If the quantity is 0, remove the price level.
		if qty > 0 {
			crawler.bidPrice += price
			crawler.bidQty += qty
			crawler.lowBid = math.Min(crawler.lowBid, price)
			crawler.highBid = math.Max(crawler.highBid, price)

			validBids = append(validBids, wsEvent.Bids[i])
		}
	}

	// Asks
	for i := 0; i < len(wsEvent.Asks); i++ {
		price, _ := strconv.ParseFloat(wsEvent.Asks[i][0], 64)
		qty, _ := strconv.ParseFloat(wsEvent.Asks[i][1], 64)

		// Check: https://github.com/binance-exchange/binance-official-api-docs/blob/master/web-socket-streams.md#how-to-manage-a-local-order-book-correctly
		// 8: If the quantity is 0, remove the price level.
		if qty > 0 {
			crawler.askPrice += price
			crawler.askQty += qty
			crawler.lowAsk = math.Min(crawler.lowAsk, price)
			crawler.highAsk = math.Max(crawler.highAsk, price)

			validAsks = append(validAsks, wsEvent.Asks[i])
		}
	}
	// Calculate volume
	crawler.askVolume += int64(len(validAsks))
	crawler.bidVolume += int64(len(validBids))

	// Store the evidence
	crawler.evidence = append(crawler.evidence, cctypes.QuotePriceEvidence{
		Bids:        validBids,
		Asks:        validAsks,
		Timestamp:   wsEvent.Timestamp / 1000, // Adjust the Binance format for datetime
		ExchangeUID: crawler.GetUID(),
	})

	return nil
}

// GetUID returns the name of this crawler
func (crawler BinanceCrawler) GetUID() string {
	return BinanceUID
}

// // GetTicker returns the name of the symbol used t get the stats
// func (crawler BinanceCrawler) GetTicker() string {
// 	return crawler.Symbol
// }

// // ToQuotePriceData serializes a json to a TickerInfo24 type
// func (crawler BinanceCrawler) ToQuotePriceData(jsonData []byte) cctypes.QuotePriceData {

// 	var result cctypes.QuotePriceData
// 	aux := struct {
// 		Volume      string `json:"volume"`
// 		QuoteVolume string `json:"quoteVolume"`
// 		HighPrice   string `json:"highPrice"`
// 		OpenPrice   string `json:"openPrice"`
// 	}{}

// 	if err := json.Unmarshal(jsonData, &aux); err != nil {
// 		panic(err)
// 	}

// 	result = cctypes.QuotePriceData{}
// 	result.Volume, _ = strconv.ParseFloat(aux.Volume, 32)
// 	result.QuoteVolume, _ = strconv.ParseFloat(aux.QuoteVolume, 32)
// 	result.HighPrice, _ = strconv.ParseFloat(aux.HighPrice, 32)
// 	result.OpenPrice, _ = strconv.ParseFloat(aux.OpenPrice, 32)

// 	return result
// }

// // SetTicker set the ticker name according the quoted currency rquested
// func (crawler BinanceCrawler) SetTicker(quotedCurrency string) {

// 	switch quotedCurrency {
// 	case "USD":
// 		crawler.Symbol = "BTCUSDT"
// 	}
// }

// // Crawl is a helper function to convert the json from Binance´s API to a QuotePriceData insance
// func (crawler BinanceCrawler) Crawl(quotedCurrency string, done chan cctypes.QuotePriceData) {

// 	// crawler.SetTicker(quotedCurrency)
// 	jsonData, err := crawler.DataCrawler.Get()
// 	if err == nil {
// 		return
// 	}

// 	priceInfo := crawler.ToQuotePriceData(jsonData)
// 	priceInfo.Timestamp = time.Now().Unix()
// 	priceInfo.DataURL = BinanceAPIUrl
// 	priceInfo.ExchangeUID = "binance"

// 	done <- priceInfo
// }
