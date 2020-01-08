package cryptocerts

import (
	cctypes "github.com/aquarelle-tech/darkmatter/x/cryptocerts/internal/types"

	"github.com/aquarelle-tech/darkmatter/x/cryptocerts/crawlers"
)

var (
	// ModuleName exports the public name for the module
	ModuleName = cctypes.ModuleName

	// Directory holds the list of active crawlers
	Directory []cctypes.PriceEvidenceCrawler
	Processor JobsProcessor
)

// InitModule creates all the crawlers instances
func InitModule() {

	publication := make(chan cctypes.QuotePriceData)

	Directory = []cctypes.PriceEvidenceCrawler{
		crawlers.NewBinanceCrawler(publication),
		crawlers.NewLiquidCrawler(publication),
		crawlers.NewBitfinexCrawler(publication),
		crawlers.NewPoloniexCrawler(publication),
		crawlers.NewCoinbaseCrawler(publication),
	}

	Processor = NewJobsProcessor(Directory, publication)
	Processor.Initialize() // Start the process
}
