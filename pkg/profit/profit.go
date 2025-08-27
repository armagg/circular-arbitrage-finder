package profit

import (
	"math"
	"strings"

	"github.com/armagg/circular-arbitrage-finder/pkg/types"
)

// Simulator evaluates triangles using top-of-book prices and fees.
type Simulator interface {
	EvaluateTOB(t types.Triangle, markets []types.Market, tobBySymbol func(symbol string) (types.TopOfBook, bool), feesBySymbol func(symbol string) (types.Fee, bool), targetQuote float64) (types.Plan, bool)
}

type TOBSimulator struct {
	MinEdge    float64 // minimum multiplicative edge, e.g., 1.0002
	SlippageBp float64 // haircuts on prices when forming limit prices
}

func NewTOBSimulator(minEdge, slippageBp float64) *TOBSimulator {
	return &TOBSimulator{MinEdge: minEdge, SlippageBp: slippageBp}
}

func (s *TOBSimulator) EvaluateTOB(t types.Triangle, markets []types.Market, tobBySymbol func(symbol string) (types.TopOfBook, bool), feesBySymbol func(symbol string) (types.Fee, bool), targetQuote float64) (types.Plan, bool) {
	// Collect TOB and fees
	tob := make([]types.TopOfBook, 3)
	fee := make([]types.Fee, 3)
	symbols := make([]string, 3)
	for i, mid := range t.MarketIds {
		m := markets[mid]
		symbols[i] = strings.ToUpper(m.Symbol)
		v, ok := tobBySymbol(symbols[i])
		if !ok || v.BidPx <= 0 || v.AskPx <= 0 {
			return types.Plan{}, false
		}
		tob[i] = v
		f, ok := feesBySymbol(symbols[i])
		if !ok {
			f = types.Fee{}
		}
		fee[i] = f
	}

	// Compute multiplicative rate through the cycle using TOB and fees
	rate := 1.0
	for i := 0; i < 3; i++ {
		mid := t.MarketIds[i]
		m := markets[mid]
		if t.Dirs[i] > 0 { // buy base with quote at ask
			px := tob[i].AskPx * (1.0 + s.SlippageBp/10000.0)
			feeMul := 1.0 - fee[i].TakerBp/10000.0
			// Convert quote -> base -> carried to next leg via base
			// In multiplicative terms for value in quote-space, buying reduces value by px and fees
			// Using ratio form: value_next = value_current / px * feeMul
			rate *= (1.0 / px) * feeMul
		} else { // sell base for quote at bid
			px := tob[i].BidPx * (1.0 - s.SlippageBp/10000.0)
			feeMul := 1.0 - fee[i].TakerBp/10000.0
			// value_next = value_current * px * feeMul
			rate *= px * feeMul
		}
		_ = m // reserved for precision/steps in later depth-aware implementation
	}

	if rate <= s.MinEdge {
		return types.Plan{}, false
	}

	// Size legs naively from targetQuote and TOB; ensure positive, leave exact rounding for later.
	// Start with value in quote currency of leg0's market quote.
	legs := [3]types.TriangleLeg{}
	value := targetQuote
	for i := 0; i < 3; i++ {
		m := markets[t.MarketIds[i]]
		if t.Dirs[i] > 0 { // buy base with quote
			px := tob[i].AskPx * (1.0 + s.SlippageBp/10000.0)
			qty := value / px
			legs[i] = types.TriangleLeg{Market: m.Symbol, Side: types.SideBuy, Qty: qty, LimitPrice: px}
			// after trade, carry base quantity forward as value in base units for next leg
			value = qty
		} else { // sell base for quote
			px := tob[i].BidPx * (1.0 - s.SlippageBp/10000.0)
			qty := value // sell all base value
			legs[i] = types.TriangleLeg{Market: m.Symbol, Side: types.SideSell, Qty: qty, LimitPrice: px}
			// after trade, carry quote value
			value = qty * px
		}
	}
	// Final value is in the starting quote currency units
	expectedProfit := value - targetQuote
	if !isFinite(expectedProfit) || expectedProfit <= 0 {
		return types.Plan{}, false
	}

	plan := types.Plan{
		Exchange:            markets[t.MarketIds[0]].Exchange,
		Legs:                legs,
		ExpectedProfitQuote: expectedProfit,
		QuoteCurrency:       markets[t.MarketIds[0]].Quote,
		ValidMs:             250,
		MaxSlippageBp:       s.SlippageBp,
		PlanID:              "", // can be filled by caller
	}
	return plan, true
}

func isFinite(v float64) bool {
	return !math.IsNaN(v) && !math.IsInf(v, 0)
}
