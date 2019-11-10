/**
 ** Copyright 2019 by Cratos Network, a project from Aquarelle AI
**/
package types

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"

	"fmt"
	"log"
	"os"
	"time"
)

const (
	ServiceHash = "1d0684170dcf58ed2499d233be72b5dde48d8124cb617f1309bae85da2fe85cf"

	ChainDataDir       = "./chain"
	LatestHashFileName = ChainDataDir + "/latest-hash"
)

// The model used to get the data
type PriceSummary struct {
	// Symbol string `json:"symbol"`
	// PriceChange        float32 `json:"priceChange"`
	// PriceChangePercent float32 `json:"priceChangePercent"`
	// LastQty            float32 `json:"LastQty"`
	// // LastPrice          float32 `json:"lastPrice"`
	// BidPrice           float32 `json:"bidPrice"`
	// AskPrice           float32 `json:"askPrice"`
	// BidQty             float32 `json:"bidQty"`
	// AskQty             float32 `json:"askQty"`
	QuoteVolume float64 `json:"quoteVolumen"`
	Volume      float64 `json:"volume"`
	HighPrice   float64 `json:"highPrice"`
	OpenPrice   float64 `json:"openPrice"`
	// LowPrice           float64 `json:"lowPrice"`
	// OpenTime           int64  `json:"openTime"`
	// CloseTime          int64  `json:"closeTime"`
}

func (info PriceSummary) String() string {
	result, err := json.Marshal(&info)

	if err != nil {
		panic(err)
	}

	return string(result)
}

/**************   Service´s Public Messages ******************/
// Message to send to the connected clients through websocket
type PriceMessage struct {
	Height        int      `json:"height"`
	AveragePrice  float64  `json:"avgPrice"`
	AverageVolume float64  `json:"avgVolumen"`
	UID           string   `json:"uid"`
	Ticker        string   `json:"ticker"`
	Timestamp     int64    `json:"timestamp"`
	Hash          string   `json:"hash"`
	Seed          string   `json:"seed"`
	PreviousHash  string   `json:"previousHash"`
	Sources       []Result `json:"sources"`
}

var previousHash string
var height int

// Return a new message with
func NewPriceMessage(uid string, ticker string, avgPrice float64, avgVolumen float64, sources []Result) PriceMessage {

	// Create a "protomessage" in order to be hashed with the hash inside
	newSeed, _ := GenerateRandomString(64)
	msg := PriceMessage{
		Height:        height,
		AveragePrice:  avgVolumen,
		AverageVolume: avgPrice,
		UID:           uid,
		Ticker:        ticker,
		Timestamp:     time.Now().Unix(),
		Seed:          newSeed,
		PreviousHash:  previousHash,
		Sources:       sources,
	}

	bytes, err := json.Marshal(msg)
	if err != nil {
		log.Println("Error serializing message", err)
	}
	fullText := fmt.Sprintf("%s:%s", ServiceHash, string(bytes))
	log.Println("Text to hash", fullText)
	msg.Hash = fmt.Sprintf("%x", sha256.Sum256([]byte(fullText)))

	// Chain the current hash with the previous one
	previousHash = msg.Hash
	height++ // The next new height
	// and keep it safe
	storeLatestHash()

	return msg
}

// Store the latest hash of the message
func storeLatestHash() {

	if _, err := os.Stat(ChainDataDir); os.IsNotExist(err) {
		err = os.Mkdir(ChainDataDir, os.ModeDir)
		if err != nil {
			log.Fatal(err)
		}
	}

	f, err := os.Create(LatestHashFileName)
	if err != nil {
		log.Fatal(err)
	}

	defer f.Close()

	f.WriteString(fmt.Sprintf("%d:%s:%d", height, previousHash, time.Now().Unix()))
	f.Sync()

}

/**************** Map & Reduce types ***************************/
// The job message to insert in a queue to be processed as part of the the Mapping Stage
type GetDataJob struct {
	Quote       string
	UID         string
	DataCrawler PriceSourceCrawler
}

// The result message that will receive the results from the mapped nodes in the Reduce Stage
type Result struct {
	CrawlerName string       `json:"name"`
	Data        PriceSummary `json:"data"`
	UID         string       `json:"messageUid"`
	HasError    bool         `json:"hasError"`
	Timestamp   int64        `json:"timestamp"`
	Ticker      string       `json:"ticker"`
}

/*****  Crawkers types **********************/
// Interface for clients
type PriceSourceCrawler interface {
	Crawl(quotedCurrency string, done chan PriceSummary)
	GetName() string
	GetTicker() string
}

// GenerateRandomBytes returns securely generated random bytes. It will return an error if the system's secure random
// number generator fails to function correctly, in which case the caller should not continue.
func GenerateRandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	// Note that err == nil only if we read len(b) bytes.
	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	return b, nil
}

// GenerateRandomString returns a URL-safe, base64 encoded securely generated random string.
// It will return an error if the system's secure random  number generator fails to function correctly, in which
// case the caller should not continue.
func GenerateRandomString(s int) (string, error) {
	b, err := GenerateRandomBytes(s)
	if err != nil {
		log.Fatal(err)
	}
	return base64.URLEncoding.EncodeToString(b), err
}
