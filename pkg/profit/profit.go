package profit

import (
	"math"
	"strings"

	"github.com/armagg/circular-arbitrage-finder/pkg/types"
)


type Simulator interface {
	EvaluateTOB(t types.Triangle, markets []types.Market, tobBySymbol func(symbol string) (types.TopOfBook, bool), feesBySymbol func(symbol string) (types.Fee, bool), targetQuote float64) (types.Plan, bool)
	EvaluateOrderBook(t types.Triangle, markets []types.Market, obBySymbol func(symbol string) (types.OrderBook, bool), feesBySymbol func(symbol string) (types.Fee, bool), targetQuote float64, maxDepth int) (types.Plan, bool)
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

// EvaluateOrderBook calculates arbitrage using full order book depth for maximum profit
func (s *TOBSimulator) EvaluateOrderBook(t types.Triangle, markets []types.Market, obBySymbol func(symbol string) (types.OrderBook, bool), feesBySymbol func(symbol string) (types.Fee, bool), targetQuote float64, maxDepth int) (types.Plan, bool) {
	orderBooks := make([]types.OrderBook, 3)
	fees := make([]types.Fee, 3)
	symbols := make([]string, 3)

	// Get order books and fees for each market
	for i, mid := range t.MarketIds {
		m := markets[mid]
		symbols[i] = strings.ToUpper(m.Symbol)

		ob, ok := obBySymbol(symbols[i])
		if !ok {
			return types.Plan{}, false
		}
		orderBooks[i] = ob

		fee, ok := feesBySymbol(symbols[i])
		if !ok {
			fee = types.Fee{}
		}
		fees[i] = fee
	}

	// Calculate maximum profit using order book depth
	bestRate, bestPlan := s.calculateOptimalPath(t, markets, orderBooks, fees, targetQuote, maxDepth)

	if bestRate <= s.MinEdge {
		return types.Plan{}, false
	}

	return bestPlan, true
}

// calculateOptimalPath finds the optimal execution path across order book depth
func (s *TOBSimulator) calculateOptimalPath(t types.Triangle, markets []types.Market, orderBooks []types.OrderBook, fees []types.Fee, targetQuote float64, maxDepth int) (float64, types.Plan) {
	bestRate := 0.0
	var bestPlan types.Plan

	// For each leg, try different execution strategies
	for leg0Levels := 1; leg0Levels <= maxDepth; leg0Levels++ {
		for leg1Levels := 1; leg1Levels <= maxDepth; leg1Levels++ {
			for leg2Levels := 1; leg2Levels <= maxDepth; leg2Levels++ {
				rate, plan := s.simulateMultiLevelExecution(t, markets, orderBooks, fees, targetQuote,
					leg0Levels, leg1Levels, leg2Levels)

				if rate > bestRate {
					bestRate = rate
					bestPlan = plan
				}
			}
		}
	}

	return bestRate, bestPlan
}

// simulateMultiLevelExecution simulates trading across multiple order book levels
func (s *TOBSimulator) simulateMultiLevelExecution(t types.Triangle, markets []types.Market, orderBooks []types.OrderBook, fees []types.Fee, targetQuote float64, levels0, levels1, levels2 int) (float64, types.Plan) {
	remainingValue := targetQuote
	rate := 1.0
	legs := [3]types.TriangleLeg{}

	// Process each leg
	for i := 0; i < 3; i++ {
		market := markets[t.MarketIds[i]]
		fee := fees[i] // Use the fee from the config

		var levels []types.Level

		if t.Dirs[i] > 0 {
			// Buy order - use ask side
			levels = orderBooks[i].Asks
		} else {
			// Sell order - use bid side
			levels = orderBooks[i].Bids
		}

		// Limit to requested depth
		maxLevels := len(levels)
		switch i {
		case 0:
			if levels0 < maxLevels { maxLevels = levels0 }
		case 1:
			if levels1 < maxLevels { maxLevels = levels1 }
		case 2:
			if levels2 < maxLevels { maxLevels = levels2 }
		}

		if maxLevels == 0 {
			return 0.0, types.Plan{}
		}
		levels = levels[:maxLevels]

		// Calculate execution price by working through order book levels
		executedValue := 0.0
		executedQty := 0.0
		remainingToExecute := remainingValue

		for _, level := range levels {
			if remainingToExecute <= 0 {
				break
			}

			price := level.Price
			availableQty := level.Qty

			// Calculate how much we can execute at this level
			if t.Dirs[i] > 0 {
				// Buying - we want to buy 'remainingToExecute' worth of base currency
				maxValueAtLevel := availableQty * price
				executeValue := math.Min(remainingToExecute, maxValueAtLevel)
				executeQty := executeValue / price

				// Apply slippage and fees from config
				effectiveExecuteValue := executeValue * (1.0 + s.SlippageBp/10000.0) * (1.0 + fee.TakerBp/10000.0)

				executedValue += effectiveExecuteValue
				executedQty += executeQty
				remainingToExecute -= executeValue
			} else {
				// Selling - we have 'remainingToExecute' units to sell
				maxQtyAtLevel := availableQty
				executeQty := math.Min(remainingToExecute, maxQtyAtLevel)
				executeValue := executeQty * price

				// Apply slippage and fees from config
				effectiveExecuteValue := executeValue * (1.0 - s.SlippageBp/10000.0) * (1.0 - fee.TakerBp/10000.0)

				executedValue += effectiveExecuteValue
				executedQty += executeQty
				remainingToExecute -= executeQty
			}
		}

		if executedQty == 0 {
			return 0.0, types.Plan{}
		}

		// Calculate effective price
		effectivePrice := executedValue / executedQty

		if t.Dirs[i] > 0 {
			// Buy: rate *= (1/price)
			rate *= (1.0 / effectivePrice)
			legs[i] = types.TriangleLeg{
				Market:     market.Symbol,
				Side:       types.SideBuy,
				Qty:        executedQty,
				LimitPrice: effectivePrice,
			}
			remainingValue = executedQty
		} else {
			// Sell: rate *= price
			rate *= effectivePrice
			legs[i] = types.TriangleLeg{
				Market:     market.Symbol,
				Side:       types.SideSell,
				Qty:        executedQty,
				LimitPrice: effectivePrice,
			}
			remainingValue = executedValue
		}
	}

	expectedProfit := remainingValue - targetQuote

	if !isFinite(expectedProfit) || expectedProfit <= 0 {
		return 0.0, types.Plan{}
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

	return rate, plan
}

func isFinite(v float64) bool {
	return !math.IsNaN(v) && !math.IsInf(v, 0)
}
