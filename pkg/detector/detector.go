package detector

import (
	"fmt"
	"strings"

	"github.com/armagg/circular-arbitrage-finder/pkg/apiout"
	"github.com/armagg/circular-arbitrage-finder/pkg/bookstore"
	"github.com/armagg/circular-arbitrage-finder/pkg/graph"
	"github.com/armagg/circular-arbitrage-finder/pkg/logger"
	"github.com/armagg/circular-arbitrage-finder/pkg/profit"
	"github.com/armagg/circular-arbitrage-finder/pkg/registry"
	"github.com/armagg/circular-arbitrage-finder/pkg/types"

	"github.com/sirupsen/logrus"
)

type Detector struct {
	Index     *graph.Index
	Books     *bookstore.TopOfBookStore
	Registry  *registry.MarketRegistry
	Sim       profit.Simulator
	Publisher apiout.Publisher
}

func NewDetector(idx *graph.Index, books *bookstore.TopOfBookStore, reg *registry.MarketRegistry, sim profit.Simulator, pub apiout.Publisher) *Detector {
	return &Detector{Index: idx, Books: books, Registry: reg, Sim: sim, Publisher: pub}
}

func (d *Detector) OnMarketChange(exchange, symbol string, targetQuote float64) {
	key := fmt.Sprintf("%s:%s", strings.ToUpper(exchange), strings.ToUpper(symbol))
	mid, ok := d.Index.MarketIndexBySymbol[key]
	if !ok {
		// This should not happen if the ingest layer is correctly adding markets first.
		logger.Log.WithField("market", key).Warn("detector: received update for unknown market")
		return
	}

	tris := d.Index.TrianglesByMarket[mid]
	if len(tris) == 0 {
		return
	}
	tobFn := func(sym string) (types.TopOfBook, bool) { return d.Books.Get(sym) }
	feeFn := func(sym string) (types.Fee, bool) { return d.Registry.GetFee(sym) }
	for _, ti := range tris {
		t := d.Index.Triangles[ti]
		plan, ok := d.Sim.EvaluateTOB(t, d.Index.Markets, tobFn, feeFn, targetQuote)
		if ok {
			logger.Log.WithFields(logrus.Fields{
				"symbol":         symbol,
				"triangle":       t.MarketIds,
				"profit_quote":   plan.ExpectedProfitQuote,
				"quote_currency": plan.QuoteCurrency,
			}).Info("detector: found profitable arbitrage")
			_ = d.Publisher.Publish(plan)
		} else {
			logger.Log.WithFields(logrus.Fields{
				"symbol":   symbol,
				"triangle": t.MarketIds,
			}).Debug("detector: arbitrage not profitable")
		}
	}
}
