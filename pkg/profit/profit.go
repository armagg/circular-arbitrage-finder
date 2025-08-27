package profit

import (
	"math"
	"strings"

	"github.com/armagg/circular-arbitrage-finder/pkg/types"
)


type Simulator interface {
	EvaluateTOB(t types.Triangle, markets []types.Market, tobBySymbol func(symbol string) (types.TopOfBook, bool), feesBySymbol func(symbol string) (types.Fee, bool), targetQuote float64) (types.Plan, bool)
}

type TOBSimulator struct {
	MinEdge    float64
	SlippageBp float64
}

func NewTOBSimulator(minEdge, slippageBp float64) *TOBSimulator {
	return &TOBSimulator{MinEdge: minEdge, SlippageBp: slippageBp}
}

func (s *TOBSimulator) EvaluateTOB(t types.Triangle, markets []types.Market, tobBySymbol func(symbol string) (types.TopOfBook, bool), feesBySymbol func(symbol string) (types.Fee, bool), targetQuote float64) (types.Plan, bool) {

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


	rate := 1.0
	for i := 0; i < 3; i++ {
		mid := t.MarketIds[i]
		m := markets[mid]
		if t.Dirs[i] > 0 {
			px := tob[i].AskPx * (1.0 + s.SlippageBp/10000.0)
			feeMul := 1.0 - fee[i].TakerBp/10000.0



			rate *= (1.0 / px) * feeMul
		} else {
			px := tob[i].BidPx * (1.0 - s.SlippageBp/10000.0)
			feeMul := 1.0 - fee[i].TakerBp/10000.0

			rate *= px * feeMul
		}
		_ = m
	}

	if rate <= s.MinEdge {
		return types.Plan{}, false
	}



	legs := [3]types.TriangleLeg{}
	value := targetQuote
	for i := 0; i < 3; i++ {
		m := markets[t.MarketIds[i]]
		if t.Dirs[i] > 0 {
			px := tob[i].AskPx * (1.0 + s.SlippageBp/10000.0)
			qty := value / px
			legs[i] = types.TriangleLeg{Market: m.Symbol, Side: types.SideBuy, Qty: qty, LimitPrice: px}

			value = qty
		} else {
			px := tob[i].BidPx * (1.0 - s.SlippageBp/10000.0)
			qty := value
			legs[i] = types.TriangleLeg{Market: m.Symbol, Side: types.SideSell, Qty: qty, LimitPrice: px}

			value = qty * px
		}
	}

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
		PlanID:              "",
	}
	return plan, true
}

func isFinite(v float64) bool {
	return !math.IsNaN(v) && !math.IsInf(v, 0)
}
