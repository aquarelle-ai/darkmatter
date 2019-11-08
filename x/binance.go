package clients

import (
	"fmt"

	"github.com/cratos.network/darkmatter/crawlers"
	"github.com/cratos.network/darkmatter/types"
)

const (
	APIURL = "https://api.binance.com/api/v3/ticker/24hr?symbol=BTCUSDT"
)

type BinanceClient struct {
	DataCrawler types.Crawler
}

func NewBinanceClient() BinanceClient {
	crawler := crawlers.NewCrawler(APIURL)

	return BinanceClient{
		DataCrawler: crawler,
	}
}

func (client BinanceClient) Crawl24() []byte {

	fmt.Printf("The HTTP request failed with error %s\n", err)

}
