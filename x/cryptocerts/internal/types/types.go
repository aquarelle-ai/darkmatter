package types

import (
	"encoding/json"
	"fmt"
	"github.com/aquarelle-tech/darkmatter/types"
)

const (
	// ModuleName is the "official" name of the module
	ModuleName = "CryptoCerts"
)

type QuotePriceEvidence struct {
	Bids        [][]string `json:"bids"`
	Asks        [][]string `json:"asks"`
	Timestamp   uint64     `json:"timestamp"`
	ExchangeUID string     `json:"exchange"`
}

// QuotePriceData is the model used to get the data
type QuotePriceData struct {
	Symbol      string               `json:"symbol"`
	AskVolume   int64                `json:"askVolume"`
	BidVolume   int64                `json:"bidVolume"`
	Bid         float64              `json:"bidPrice"`
	Ask         float64              `json:"askPrice"`
	BidQty      float64              `json:"bidQty"`
	AskQty      float64              `json:"askQty"`
	LowBid      float64              `json:"lowBid"`
	LowAsk      float64              `json:"lowAsk"`
	HighBid     float64              `json:"highBid"`
	HighAsk     float64              `json:"highAsk"`
	Timestamp   int64                `json:"timestamp"`
	ExchangeUID string               `json:"provider"`
	Evidence    []QuotePriceEvidence `json:"evidence"`
}

type QuotePriceMessage struct {
	AveragePrice float64 `json:"averagePrice"`
	Volume       int64   `json:"volume"`
	Timestamp    int64   `json:"timestamp"`
}

// String will print the JSON output
func (info QuotePriceData) String() string {
	result, err := json.Marshal(&info)

	if err != nil {
		panic(err)
	}

	return string(result)
}

// ExchangeMarketEvent holds the order book price and quantity depth updates for any exchange. Used to unify
// the events in different exchanges
type ExchangeMarketEvent struct {
	ExchangeName  string     `json:"exchangeName"`
	Symbol        string     `json:"symbol"`
	FirstUpdateID int64      `json:"firstUpdateId"` // First update ID in event
	LastUpdateID  int64      `json:"lastUpdateId"`  // Final update ID in event
	Bids          [][]string `json:"bids"`          // Bids to be updated
	Asks          [][]string `json:"asks"`          // Asks to be updated
}

// String implements the Stringer interface
func (e ExchangeMarketEvent) String() string {
	return fmt.Sprintf("%s (%s): firstUpdateID:%d, lastUpdateID:%d ", e.ExchangeName, e.Symbol, e.FirstUpdateID, e.LastUpdateID)
}

// PriceEvidenceCrawler is the specific type to manage the evidence for stock markets
type PriceEvidenceCrawler interface {
	GetUID() string
}

// GetDataJob is the job message to insert in a queue to be processed as part of the the Mapping Stage
type GetDataJob struct {
	DataCrawler PriceEvidenceCrawler
}

// Result is the message that will receive the results from the mapped nodes in the Reduce Stage
type Result struct {
	CrawlerName string         `json:"name"`
	Data        QuotePriceData `json:"data"`
	Timestamp   int64          `json:"timestamp"`
	Ticker      string         `json:"ticker"`
	Hash        string         `json:"hash"`
}

// UpdateHash creates a double hash (sha256(sha256)) for all the content
func (result *Result) UpdateHash() error {
	// create a hash the result
	result.Hash = "" // To asure a clean hash (without an spurious value in the hash variable)
	hash, err := types.CalculateHash(result)

	if err == nil {
		result.Hash = hash
	}

	return err // No error
}
