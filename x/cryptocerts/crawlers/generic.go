package crawlers

import (
	"log"
	"math"
	"reflect"
	"time"

	cctypes "github.com/aquarelle-tech/darkmatter/x/cryptocerts/internal/types"
	"github.com/golang/glog"
	"github.com/gorilla/websocket"
)

// GenericCrawler implements the  REST API client to get data from Bitfinex
type GenericCrawler struct {
	// DataCrawler *clients.WebCrawler
	APIUrl string
	Ticker string
	// Events      chan cctypes.ExchangeMarketEvent
	Publication chan cctypes.QuotePriceData

	// Counter variables to create the aggregations
	askVolume int64
	bidVolume int64
	lowAsk    float64
	lowBid    float64
	askPrice  float64
	bidPrice  float64
	highAsk   float64
	highBid   float64
	bidQty    float64
	askQty    float64
	evidence  []cctypes.QuotePriceEvidence

	// Functions to be initialized by the descendants
	getUID               func() string
	initializeConnection func(c *websocket.Conn)
	closeConnection      func(c *websocket.Conn)
	serializeEvent       func(message []byte, c *websocket.Conn) error
}

// NewGenericExchangeCrawler creates a new instance of a base crawler
func NewGenericExchangeCrawler(apiUrl string, publication chan cctypes.QuotePriceData) *GenericCrawler {
	crawler := GenericCrawler{
		APIUrl:      apiUrl,
		Publication: publication,
	}

	return &crawler
}

// Start launch the connection to the data from the exchange
func (crawler *GenericCrawler) Start() {

	crawler.resetCounters() // Starting with a clean values
	go crawler.connectToStream()
}

// getFloat is helper function to convert anoymous data to float64
func getFloat(unk interface{}) float64 {
	v := reflect.ValueOf(unk)
	floatType := reflect.TypeOf(float64(0))
	fv := v.Convert(floatType)
	return fv.Float()
}

func (crawler *GenericCrawler) connectToStream() {

	glog.Infof("%s: Connecting stream to %s", crawler.getUID(), crawler.APIUrl)
	c, response, err := websocket.DefaultDialer.Dial(crawler.APIUrl, nil)
	if err != nil {
		log.Fatal("dial:", err)
	}
	glog.Info(response)
	// crawler.Events = make(chan cctypes.ExchangeMarketEvent, 1000)

	done := make(chan bool)
	collectorTicker := time.NewTicker(time.Second) // Each second
	crawler.setSummaryCollectorTicker(done, collectorTicker)

	defer func() {
		c.Close()
		collectorTicker.Stop()
		done <- true

		if crawler.closeConnection != nil {
			crawler.closeConnection(c)
		}
	}()

	if crawler.initializeConnection != nil {
		crawler.initializeConnection(c)
	}

	crawler.handleMessages(c, done)
}

// setSummaryCollectorTicker uses a time ticker to collects all the sats and reset a new cycle of agreggations
func (crawler *GenericCrawler) setSummaryCollectorTicker(done chan bool, ticker *time.Ticker) {
	// Create aticker to calculate the aggregations and initialize the counters
	go func() {
		for {
			select {
			case <-done:
				return // Shutdwn gracefully the ticker
			case <-ticker.C:
				if crawler.askQty > 0 || crawler.bidQty > 0 {

					data := cctypes.QuotePriceData{
						AskVolume:   crawler.askVolume,
						BidVolume:   crawler.bidVolume,
						Bid:         crawler.bidPrice,
						Ask:         crawler.askPrice,
						BidQty:      crawler.bidQty,
						AskQty:      crawler.askQty,
						LowBid:      crawler.lowBid,
						LowAsk:      crawler.lowAsk,
						HighBid:     crawler.highBid,
						HighAsk:     crawler.highAsk,
						Timestamp:   time.Now().Unix(),
						ExchangeUID: crawler.getUID(),
						Evidence:    crawler.evidence,
					}

					// Add the message to the queue
					crawler.Publication <- data
					crawler.resetCounters()

				}
			}
		}
	}()
}

// resetCounters reset all the ounters and aggregators
func (crawler *GenericCrawler) resetCounters() {

	crawler.askPrice = 0.0
	crawler.bidPrice = 0.0
	crawler.bidQty = 0
	crawler.askQty = 0
	crawler.lowAsk = math.MaxFloat64
	crawler.lowBid = math.MaxFloat64
	crawler.highAsk = 0.0
	crawler.highBid = 0.0
	crawler.askVolume = 0
	crawler.bidVolume = 0
	crawler.evidence = nil

}

// handleMessages will loop infinitely reading the messages from Binance. These process will aggregates all the ticks
func (crawler *GenericCrawler) handleMessages(c *websocket.Conn, done chan bool) error {
	// Get message in an infinite loop
	glog.Infof("%s: Starting infinite loop for websocket client", crawler.getUID())
	for {
		select {
		case <-done:
			glog.Infof("Closing the stream for %s", crawler.getUID())
			return nil // Shutdown gracefully the loop
		default:
			msgType, message, err := c.ReadMessage()
			if err != nil {
				glog.Errorf("Error reading from stream: %v", err)
				return err
			}

			if msgType == websocket.TextMessage {
				// Extract the info and calculates the aggregations
				crawler.serializeEvent(message, c)
			}
		} // select
	} // for
}

// Crawl is the implementation function to convert the json from Bitfinex´s API to a QuotePriceData instance
// func (c GenericCrawler) Crawl(done chan interface{}) error {

// 	glog.Infof("%s: Crawling...", c.getName())
// 	jsonData, err := c.DataCrawler.Get()

// 	if err != nil {
// 		glog.Errorf("The crawler has failed! No data read. %v", err)
// 		return err
// 	}

// 	// Create the data block
// 	priceInfo := c.toQuotePriceData(jsonData)
// 	priceInfo.Timestamp = time.Now().Unix()
// 	priceInfo.ExchangeUID = BitfinexID

// 	// buf := bytes.Buffer{}
// 	// binary.Write(&buf, binary.BigEndian, priceInfo)
// 	// msgBytes := buf.Bytes()

// 	// var data []byte
// 	// var mh codec.MsgpackHandle
// 	// enc := codec.NewEncoderBytes(&data, &mh)
// 	// err = enc.Encode(priceInfo)

// 	// glog.Infof("Bitfinex: Extracted %d bytes, (%v)", len(data), data)

// 	done <- priceInfo

// 	return nil
// }

// keepAlive will send the pong reply to Binance when the service request a signal to no ternminate the connection
func (crawler *GenericCrawler) keepAlive(c *websocket.Conn, timeout time.Duration) {
	ticker := time.NewTicker(timeout)
	// keepAlive(c, WebsocketTimeout)

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
			if err == nil {
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
