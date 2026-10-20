package profit

import (
	"math"
	"testing"

	"github.com/armagg/circular-arbitrage-finder/pkg/types"
)

func TestNewTOBSimulator(t *testing.T) {
	minEdge := 0.001
	slippageBp := 5.0

	sim := NewTOBSimulator(minEdge, slippageBp)

	if sim == nil {
		t.Error("NewTOBSimulator should return a non-nil simulator")
	}

	if sim.MinEdge != minEdge {
		t.Errorf("Expected MinEdge %f, got %f", minEdge, sim.MinEdge)
	}

	if sim.SlippageBp != slippageBp {
		t.Errorf("Expected SlippageBp %f, got %f", slippageBp, sim.SlippageBp)
	}
}

func TestIsFinite(t *testing.T) {
	tests := []struct {
		name     string
		value    float64
		expected bool
	}{
		{"Normal number", 1.5, true},
		{"Zero", 0.0, true},
		{"Negative number", -2.5, true},
		{"Positive infinity", math.Inf(1), false},
		{"Negative infinity", math.Inf(-1), false},
		{"NaN", math.NaN(), false},
		{"Very large number", 1e308, true},
		{"Very small number", 1e-323, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isFinite(tt.value)
			if result != tt.expected {
				t.Errorf("isFinite(%f) = %v, expected %v", tt.value, result, tt.expected)
			}
		})
	}
}

func TestTOBSimulatorEvaluateTOB(t *testing.T) {
	sim := NewTOBSimulator(0.0001, 0.1)

	markets := []types.Market{
		{Exchange: "binance", Symbol: "BTCUSDT", Base: "BTC", Quote: "USDT"},
		{Exchange: "binance", Symbol: "ETHBTC", Base: "ETH", Quote: "BTC"},
		{Exchange: "binance", Symbol: "ETHUSDT", Base: "ETH", Quote: "USDT"},
	}

	triangle := types.Triangle{
		MarketIds: [3]int{0, 1, 2},
		Dirs:      [3]int8{1, -1, -1},
		QuoteCcy:  "USDT",
	}

	tobBySymbol := func(symbol string) (types.TopOfBook, bool) {
		switch symbol {
		case "BTCUSDT":
			return types.TopOfBook{BidPx: 50000.0, AskPx: 50000.0, BidSz: 2.1, AskSz: 1.8}, true
		case "ETHBTC":
			return types.TopOfBook{BidPx: 0.03, AskPx: 0.03, BidSz: 65.0, AskSz: 62.0}, true
		case "ETHUSDT":
			return types.TopOfBook{BidPx: 1500.0, AskPx: 1500.0, BidSz: 10.0, AskSz: 8.0}, true
		default:
			return types.TopOfBook{}, false
		}
	}

	feeBySymbol := func(symbol string) (types.Fee, bool) {
		return types.Fee{TakerBp: 0.1, MakerBp: 0.05}, true
	}

	targetQuote := 1000.0

	plan, found := sim.EvaluateTOB(triangle, markets, tobBySymbol, feeBySymbol, targetQuote)

	t.Logf("Arbitrage found: %v, profit: %f", found, plan.ExpectedProfitQuote)

	if found {
		if plan.ExpectedProfitQuote <= 0 {
			t.Errorf("Found arbitrage but profit is not positive: %f", plan.ExpectedProfitQuote)
		}

		if len(plan.Legs) != 3 {
			t.Errorf("Expected 3 legs, got %d", len(plan.Legs))
		}

		if plan.QuoteCurrency != "USDT" {
			t.Errorf("Expected quote currency USDT, got %s", plan.QuoteCurrency)
		}
	}
}

func TestTOBSimulatorEvaluateTOBNoProfit(t *testing.T) {
	sim := NewTOBSimulator(0.01, 5.0)

	markets := []types.Market{
		{Exchange: "binance", Symbol: "BTCUSDT", Base: "BTC", Quote: "USDT"},
		{Exchange: "binance", Symbol: "ETHBTC", Base: "ETH", Quote: "BTC"},
		{Exchange: "binance", Symbol: "ETHUSDT", Base: "ETH", Quote: "USDT"},
	}

	triangle := types.Triangle{
		MarketIds: [3]int{0, 1, 2},
		Dirs:      [3]int8{1, -1, -1},
		QuoteCcy:  "USDT",
	}

	tobBySymbol := func(symbol string) (types.TopOfBook, bool) {
		switch symbol {
		case "BTCUSDT":
			return types.TopOfBook{BidPx: 50000.0, AskPx: 50000.0, BidSz: 2.1, AskSz: 1.8}, true
		case "ETHBTC":
			return types.TopOfBook{BidPx: 0.03, AskPx: 0.03, BidSz: 65.0, AskSz: 62.0}, true
		case "ETHUSDT":
			return types.TopOfBook{BidPx: 1500.0, AskPx: 1500.0, BidSz: 10.0, AskSz: 8.0}, true
		default:
			return types.TopOfBook{}, false
		}
	}

	feeBySymbol := func(symbol string) (types.Fee, bool) {
		return types.Fee{TakerBp: 0.1, MakerBp: 0.05}, true
	}

	targetQuote := 1000.0

	_, found := sim.EvaluateTOB(triangle, markets, tobBySymbol, feeBySymbol, targetQuote)

	if found {
		t.Error("Expected no profitable arbitrage to be found")
	}
}

func TestTOBSimulatorEvaluateTOBMissingData(t *testing.T) {
	sim := NewTOBSimulator(0.001, 5.0)

	markets := []types.Market{
		{Exchange: "binance", Symbol: "BTCUSDT", Base: "BTC", Quote: "USDT"},
	}

	triangle := types.Triangle{
		MarketIds: [3]int{0, 1, 2},
		Dirs:      [3]int8{1, -1, -1},
		QuoteCcy:  "USDT",
	}

	tobBySymbol := func(symbol string) (types.TopOfBook, bool) {
		if symbol == "BTCUSDT" {
			return types.TopOfBook{BidPx: 0, AskPx: 0}, true
		}
		return types.TopOfBook{}, false
	}

	feeBySymbol := func(symbol string) (types.Fee, bool) {
		return types.Fee{TakerBp: 0.1, MakerBp: 0.05}, true
	}

	targetQuote := 1000.0

	_, found := sim.EvaluateTOB(triangle, markets, tobBySymbol, feeBySymbol, targetQuote)

	if found {
		t.Error("Expected no arbitrage when data is missing or invalid")
	}
}

func TestTOBSimulatorComplexArbitrage(t *testing.T) {
	sim := NewTOBSimulator(0.0001, 0.1)

	markets := []types.Market{
		{Exchange: "binance", Symbol: "BTCUSDT", Base: "BTC", Quote: "USDT"},
		{Exchange: "binance", Symbol: "ETHBTC", Base: "ETH", Quote: "BTC"},
		{Exchange: "binance", Symbol: "ETHUSDT", Base: "ETH", Quote: "USDT"},
	}

	triangle := types.Triangle{
		MarketIds: [3]int{0, 1, 2},
		Dirs:      [3]int8{1, -1, -1},
		QuoteCcy:  "USDT",
	}

	tobBySymbol := func(symbol string) (types.TopOfBook, bool) {
		switch symbol {
		case "BTCUSDT":
			return types.TopOfBook{BidPx: 49800.0, AskPx: 49850.0, BidSz: 2.1, AskSz: 1.8}, true
		case "ETHBTC":
			return types.TopOfBook{BidPx: 0.0295, AskPx: 0.0298, BidSz: 65.0, AskSz: 62.0}, true
		case "ETHUSDT":
			return types.TopOfBook{BidPx: 1480.0, AskPx: 1485.0, BidSz: 10.0, AskSz: 8.0}, true
		default:
			return types.TopOfBook{}, false
		}
	}

	feeBySymbol := func(symbol string) (types.Fee, bool) {
		return types.Fee{TakerBp: 0.08, MakerBp: 0.04}, true
	}

	targetQuote := 1000.0

	plan, found := sim.EvaluateTOB(triangle, markets, tobBySymbol, feeBySymbol, targetQuote)

	t.Logf("Complex arbitrage found: %v, profit: %f", found, plan.ExpectedProfitQuote)

	if found {
		if plan.ExpectedProfitQuote <= 0 {
			t.Errorf("Found arbitrage but profit is not positive: %f", plan.ExpectedProfitQuote)
		}

		if plan.Exchange != "binance" {
			t.Errorf("Expected exchange binance, got %s", plan.Exchange)
		}

		if plan.ValidMs != 250 {
			t.Errorf("Expected valid ms 250, got %d", plan.ValidMs)
		}

		if plan.MaxSlippageBp != 0.1 {
			t.Errorf("Expected max slippage 0.1, got %f", plan.MaxSlippageBp)
		}

		expectedSides := [3]types.Side{types.SideBuy, types.SideSell, types.SideSell}
		for i, leg := range plan.Legs {
			if leg.Side != expectedSides[i] {
				t.Errorf("Expected leg %d side %s, got %s", i, expectedSides[i], leg.Side)
			}
			if leg.Qty <= 0 {
				t.Errorf("Expected positive quantity for leg %d, got %f", i, leg.Qty)
			}
			if leg.LimitPrice <= 0 {
				t.Errorf("Expected positive limit price for leg %d, got %f", i, leg.LimitPrice)
			}
		}
	}
}

func TestTOBSimulatorTriangleDirections(t *testing.T) {
	sim := NewTOBSimulator(0.001, 5.0)

	markets := []types.Market{
		{Exchange: "binance", Symbol: "BTCUSDT", Base: "BTC", Quote: "USDT"},
		{Exchange: "binance", Symbol: "ETHBTC", Base: "ETH", Quote: "BTC"},
		{Exchange: "binance", Symbol: "ETHUSDT", Base: "ETH", Quote: "USDT"},
	}

	testCases := []struct {
		name       string
		directions [3]int8
	}{
		{"Forward arbitrage", [3]int8{1, -1, -1}},
		{"Reverse arbitrage", [3]int8{-1, 1, 1}},
		{"Mixed directions", [3]int8{1, 1, -1}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			triangle := types.Triangle{
				MarketIds: [3]int{0, 1, 2},
				Dirs:      tc.directions,
				QuoteCcy:  "USDT",
			}

			tobBySymbol := func(symbol string) (types.TopOfBook, bool) {
				switch symbol {
				case "BTCUSDT":
					return types.TopOfBook{BidPx: 50000.0, AskPx: 50010.0}, true
				case "ETHBTC":
					return types.TopOfBook{BidPx: 0.03, AskPx: 0.0301}, true
				case "ETHUSDT":
					return types.TopOfBook{BidPx: 1500.0, AskPx: 1501.0}, true
				default:
					return types.TopOfBook{}, false
				}
			}

			feeBySymbol := func(symbol string) (types.Fee, bool) {
				return types.Fee{TakerBp: 0.1, MakerBp: 0.05}, true
			}

			_, found := sim.EvaluateTOB(triangle, markets, tobBySymbol, feeBySymbol, 1000.0)

			t.Logf("Triangle with directions %v: profitable = %v", tc.directions, found)
		})
	}
}

func TestTOBSimulatorEdgeCases(t *testing.T) {
	sim := NewTOBSimulator(0.001, 5.0)

	markets := []types.Market{
		{Exchange: "binance", Symbol: "BTCUSDT", Base: "BTC", Quote: "USDT"},
		{Exchange: "binance", Symbol: "ETHBTC", Base: "ETH", Quote: "BTC"},
		{Exchange: "binance", Symbol: "ETHUSDT", Base: "ETH", Quote: "USDT"},
	}

	triangle := types.Triangle{
		MarketIds: [3]int{0, 1, 2},
		Dirs:      [3]int8{1, -1, -1},
		QuoteCcy:  "USDT",
	}

	t.Run("Zero prices", func(t *testing.T) {
		tobBySymbol := func(symbol string) (types.TopOfBook, bool) {
			switch symbol {
			case "BTCUSDT":
				return types.TopOfBook{BidPx: 0, AskPx: 0}, true
			case "ETHBTC":
				return types.TopOfBook{BidPx: 0, AskPx: 0}, true
			case "ETHUSDT":
				return types.TopOfBook{BidPx: 0, AskPx: 0}, true
			default:
				return types.TopOfBook{}, false
			}
		}
		feeBySymbol := func(symbol string) (types.Fee, bool) {
			return types.Fee{TakerBp: 0.1, MakerBp: 0.05}, true
		}

		_, found := sim.EvaluateTOB(triangle, markets, tobBySymbol, feeBySymbol, 1000.0)
		if found {
			t.Error("Should not find arbitrage with zero prices")
		}
	})

	t.Run("Negative prices", func(t *testing.T) {
		tobBySymbol := func(symbol string) (types.TopOfBook, bool) {
			switch symbol {
			case "BTCUSDT":
				return types.TopOfBook{BidPx: -100, AskPx: -90}, true
			case "ETHBTC":
				return types.TopOfBook{BidPx: -0.03, AskPx: -0.02}, true
			case "ETHUSDT":
				return types.TopOfBook{BidPx: -3000, AskPx: -2900}, true
			default:
				return types.TopOfBook{}, false
			}
		}
		feeBySymbol := func(symbol string) (types.Fee, bool) {
			return types.Fee{TakerBp: 0.1, MakerBp: 0.05}, true
		}

		_, found := sim.EvaluateTOB(triangle, markets, tobBySymbol, feeBySymbol, 1000.0)
		if found {
			t.Error("Should not find arbitrage with negative prices")
		}
	})

	t.Run("Infinite results", func(t *testing.T) {
		simHighSlippage := NewTOBSimulator(0.001, 100.0)

		tobBySymbol := func(symbol string) (types.TopOfBook, bool) {
			switch symbol {
			case "BTCUSDT":
				return types.TopOfBook{BidPx: 1e-100, AskPx: 1e100}, true
			case "ETHBTC":
				return types.TopOfBook{BidPx: 1e-100, AskPx: 1e100}, true
			case "ETHUSDT":
				return types.TopOfBook{BidPx: 1e-100, AskPx: 1e100}, true
			default:
				return types.TopOfBook{}, false
			}
		}
		feeBySymbol := func(symbol string) (types.Fee, bool) {
			return types.Fee{TakerBp: 0.1, MakerBp: 0.05}, true
		}

		_, found := simHighSlippage.EvaluateTOB(triangle, markets, tobBySymbol, feeBySymbol, 1000.0)
		t.Logf("Infinite result test: profitable = %v", found)
	})
}

func BenchmarkTOBSimulatorEvaluateTOB(b *testing.B) {
	sim := NewTOBSimulator(0.001, 5.0)

	markets := []types.Market{
		{Exchange: "binance", Symbol: "BTCUSDT", Base: "BTC", Quote: "USDT"},
		{Exchange: "binance", Symbol: "ETHBTC", Base: "ETH", Quote: "BTC"},
		{Exchange: "binance", Symbol: "ETHUSDT", Base: "ETH", Quote: "USDT"},
	}

	triangle := types.Triangle{
		MarketIds: [3]int{0, 1, 2},
		Dirs:      [3]int8{1, -1, -1},
		QuoteCcy:  "USDT",
	}

	tobBySymbol := func(symbol string) (types.TopOfBook, bool) {
		switch symbol {
		case "BTCUSDT":
			return types.TopOfBook{BidPx: 50000.0, AskPx: 50010.0}, true
		case "ETHBTC":
			return types.TopOfBook{BidPx: 0.03, AskPx: 0.0301}, true
		case "ETHUSDT":
			return types.TopOfBook{BidPx: 1500.0, AskPx: 1501.0}, true
		default:
			return types.TopOfBook{}, false
		}
	}

	feeBySymbol := func(symbol string) (types.Fee, bool) {
		return types.Fee{TakerBp: 0.1, MakerBp: 0.05}, true
	}

	targetQuote := 1000.0

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sim.EvaluateTOB(triangle, markets, tobBySymbol, feeBySymbol, targetQuote)
	}
}

func TestTOBSimulatorEvaluateOrderBook(t *testing.T) {
	sim := NewTOBSimulator(0.0001, 0.1)

	markets := []types.Market{
		{Exchange: "binance", Symbol: "BTCUSDT", Base: "BTC", Quote: "USDT"},
		{Exchange: "binance", Symbol: "ETHBTC", Base: "ETH", Quote: "BTC"},
		{Exchange: "binance", Symbol: "ETHUSDT", Base: "ETH", Quote: "USDT"},
	}

	triangle := types.Triangle{
		MarketIds: [3]int{0, 1, 2},
		Dirs:      [3]int8{1, -1, -1},
		QuoteCcy:  "USDT",
	}

	orderBooks := map[string]types.OrderBook{
		"BTCUSDT": {
			Bids: []types.Level{
				{Price: 49900.0, Qty: 2.5}, {Price: 49850.0, Qty: 3.1}, {Price: 49800.0, Qty: 4.2},
				{Price: 49750.0, Qty: 2.8}, {Price: 49700.0, Qty: 5.5},
			},
			Asks: []types.Level{
				{Price: 50000.0, Qty: 2.1}, {Price: 50050.0, Qty: 2.8}, {Price: 50100.0, Qty: 3.7},
				{Price: 50150.0, Qty: 2.4}, {Price: 50200.0, Qty: 4.9},
			},
		},
		"ETHBTC": {
			Bids: []types.Level{
				{Price: 0.0295, Qty: 65.0}, {Price: 0.0290, Qty: 70.0}, {Price: 0.0285, Qty: 75.0},
				{Price: 0.0280, Qty: 80.0}, {Price: 0.0275, Qty: 85.0},
			},
			Asks: []types.Level{
				{Price: 0.0300, Qty: 62.0}, {Price: 0.0305, Qty: 67.0}, {Price: 0.0310, Qty: 72.0},
				{Price: 0.0315, Qty: 77.0}, {Price: 0.0320, Qty: 82.0},
			},
		},
		"ETHUSDT": {
			Bids: []types.Level{
				{Price: 1480.0, Qty: 10.0}, {Price: 1475.0, Qty: 12.0}, {Price: 1470.0, Qty: 14.0},
				{Price: 1465.0, Qty: 11.0}, {Price: 1460.0, Qty: 16.0},
			},
			Asks: []types.Level{
				{Price: 1490.0, Qty: 8.0}, {Price: 1495.0, Qty: 11.0}, {Price: 1500.0, Qty: 13.0},
				{Price: 1505.0, Qty: 10.0}, {Price: 1510.0, Qty: 15.0},
			},
		},
	}

	obBySymbol := func(symbol string) (types.OrderBook, bool) {
		ob, exists := orderBooks[symbol]
		return ob, exists
	}

	feeBySymbol := func(symbol string) (types.Fee, bool) {
		return types.Fee{TakerBp: 0.08, MakerBp: 0.04}, true
	}

	targetQuote := 1000.0
	maxDepth := 3

	plan, found := sim.EvaluateOrderBook(triangle, markets, obBySymbol, feeBySymbol, targetQuote, maxDepth)

	t.Logf("OrderBook arbitrage found: %v, profit: %f", found, plan.ExpectedProfitQuote)

	if found {
		if plan.ExpectedProfitQuote <= 0 {
			t.Errorf("Found arbitrage but profit is not positive: %f", plan.ExpectedProfitQuote)
		}

		if len(plan.Legs) != 3 {
			t.Errorf("Expected 3 legs, got %d", len(plan.Legs))
		}

		if plan.QuoteCurrency != "USDT" {
			t.Errorf("Expected quote currency USDT, got %s", plan.QuoteCurrency)
		}

		if plan.Exchange != "binance" {
			t.Errorf("Expected exchange binance, got %s", plan.Exchange)
		}

		for i, leg := range plan.Legs {
			if leg.Qty <= 0 {
				t.Errorf("Expected positive quantity for leg %d, got %f", i, leg.Qty)
			}
			if leg.LimitPrice <= 0 {
				t.Errorf("Expected positive limit price for leg %d, got %f", i, leg.LimitPrice)
			}
			if leg.Market == "" {
				t.Errorf("Expected non-empty market for leg %d", i)
			}
		}
	}
}

func TestTOBSimulatorEvaluateOrderBookDepth(t *testing.T) {
	sim := NewTOBSimulator(0.0001, 0.1)

	markets := []types.Market{
		{Exchange: "binance", Symbol: "BTCUSDT", Base: "BTC", Quote: "USDT"},
		{Exchange: "binance", Symbol: "ETHBTC", Base: "ETH", Quote: "BTC"},
		{Exchange: "binance", Symbol: "ETHUSDT", Base: "ETH", Quote: "USDT"},
	}

	triangle := types.Triangle{
		MarketIds: [3]int{0, 1, 2},
		Dirs:      [3]int8{1, -1, -1},
		QuoteCcy:  "USDT",
	}

	orderBooks := map[string]types.OrderBook{
		"BTCUSDT": {
			Asks: []types.Level{
				{Price: 50000.0, Qty: 1.0}, {Price: 50010.0, Qty: 2.0}, {Price: 50020.0, Qty: 1.5},
				{Price: 50030.0, Qty: 2.5}, {Price: 50040.0, Qty: 1.8},
			},
			Bids: []types.Level{
				{Price: 49990.0, Qty: 1.2}, {Price: 49980.0, Qty: 2.1}, {Price: 49970.0, Qty: 1.6},
				{Price: 49960.0, Qty: 2.3}, {Price: 49950.0, Qty: 1.9},
			},
		},
		"ETHBTC": {
			Asks: []types.Level{
				{Price: 0.0300, Qty: 30.0}, {Price: 0.0301, Qty: 35.0}, {Price: 0.0302, Qty: 32.0},
				{Price: 0.0303, Qty: 38.0}, {Price: 0.0304, Qty: 33.0},
			},
			Bids: []types.Level{
				{Price: 0.0299, Qty: 31.0}, {Price: 0.0298, Qty: 36.0}, {Price: 0.0297, Qty: 33.0},
				{Price: 0.0296, Qty: 39.0}, {Price: 0.0295, Qty: 34.0},
			},
		},
		"ETHUSDT": {
			Asks: []types.Level{
				{Price: 1490.0, Qty: 5.0}, {Price: 1491.0, Qty: 6.0}, {Price: 1492.0, Qty: 5.5},
				{Price: 1493.0, Qty: 7.0}, {Price: 1494.0, Qty: 6.2},
			},
			Bids: []types.Level{
				{Price: 1489.0, Qty: 5.2}, {Price: 1488.0, Qty: 6.1}, {Price: 1487.0, Qty: 5.6},
				{Price: 1486.0, Qty: 7.1}, {Price: 1485.0, Qty: 6.3},
			},
		},
	}

	obBySymbol := func(symbol string) (types.OrderBook, bool) {
		ob, exists := orderBooks[symbol]
		return ob, exists
	}

	feeBySymbol := func(symbol string) (types.Fee, bool) {
		return types.Fee{TakerBp: 0.1, MakerBp: 0.05}, true
	}

	targetQuote := 1000.0

	testCases := []struct {
		name     string
		maxDepth int
	}{
		{"Depth 1", 1},
		{"Depth 3", 3},
		{"Depth 5", 5},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			plan, found := sim.EvaluateOrderBook(triangle, markets, obBySymbol, feeBySymbol, targetQuote, tc.maxDepth)

			t.Logf("%s: Arbitrage found: %v, profit: %f", tc.name, found, plan.ExpectedProfitQuote)

			if found {
				if plan.ExpectedProfitQuote <= 0 {
					t.Errorf("%s: Found arbitrage but profit is not positive: %f", tc.name, plan.ExpectedProfitQuote)
				}

				if len(plan.Legs) != 3 {
					t.Errorf("%s: Expected 3 legs, got %d", tc.name, len(plan.Legs))
				}
			}
		})
	}
}

func TestTOBSimulatorEvaluateOrderBookMissingData(t *testing.T) {
	sim := NewTOBSimulator(0.001, 5.0)

	markets := []types.Market{
		{Exchange: "binance", Symbol: "BTCUSDT", Base: "BTC", Quote: "USDT"},
		{Exchange: "binance", Symbol: "ETHBTC", Base: "ETH", Quote: "BTC"},
		{Exchange: "binance", Symbol: "ETHUSDT", Base: "ETH", Quote: "USDT"},
	}

	triangle := types.Triangle{
		MarketIds: [3]int{0, 1, 2},
		Dirs:      [3]int8{1, -1, -1},
		QuoteCcy:  "USDT",
	}

	obBySymbol := func(symbol string) (types.OrderBook, bool) {
		switch symbol {
		case "BTCUSDT":
			return types.OrderBook{}, true
		case "ETHBTC":
			return types.OrderBook{
				Asks: []types.Level{{Price: 0.03, Qty: 10.0}},
				Bids: []types.Level{{Price: 0.029, Qty: 10.0}},
			}, true
		case "ETHUSDT":
			return types.OrderBook{
				Asks: []types.Level{{Price: 1500.0, Qty: 10.0}},
				Bids: []types.Level{{Price: 1490.0, Qty: 10.0}},
			}, true
		default:
			return types.OrderBook{}, false
		}
	}

	feeBySymbol := func(symbol string) (types.Fee, bool) {
		return types.Fee{TakerBp: 0.1, MakerBp: 0.05}, true
	}

	targetQuote := 1000.0
	maxDepth := 5

	_, found := sim.EvaluateOrderBook(triangle, markets, obBySymbol, feeBySymbol, targetQuote, maxDepth)

	if found {
		t.Error("Expected no arbitrage when order book data is missing or empty")
	}
}

func TestTOBSimulatorEvaluateOrderBookEdgeCases(t *testing.T) {
	sim := NewTOBSimulator(0.001, 5.0)

	markets := []types.Market{
		{Exchange: "binance", Symbol: "BTCUSDT", Base: "BTC", Quote: "USDT"},
		{Exchange: "binance", Symbol: "ETHBTC", Base: "ETH", Quote: "BTC"},
		{Exchange: "binance", Symbol: "ETHUSDT", Base: "ETH", Quote: "USDT"},
	}

	triangle := types.Triangle{
		MarketIds: [3]int{0, 1, 2},
		Dirs:      [3]int8{1, -1, -1},
		QuoteCcy:  "USDT",
	}

	t.Run("Zero quantity levels", func(t *testing.T) {
		orderBooks := map[string]types.OrderBook{
			"BTCUSDT": {Asks: []types.Level{{Price: 50000.0, Qty: 0}}, Bids: []types.Level{{Price: 49900.0, Qty: 0}}},
			"ETHBTC":  {Asks: []types.Level{{Price: 0.03, Qty: 0}}, Bids: []types.Level{{Price: 0.029, Qty: 0}}},
			"ETHUSDT": {Asks: []types.Level{{Price: 1500.0, Qty: 0}}, Bids: []types.Level{{Price: 1490.0, Qty: 0}}},
		}

		obBySymbol := func(symbol string) (types.OrderBook, bool) {
			ob, exists := orderBooks[symbol]
			return ob, exists
		}

		feeBySymbol := func(symbol string) (types.Fee, bool) {
			return types.Fee{TakerBp: 0.1, MakerBp: 0.05}, true
		}

		_, found := sim.EvaluateOrderBook(triangle, markets, obBySymbol, feeBySymbol, 1000.0, 1)

		if found {
			t.Error("Should not find arbitrage with zero quantity levels")
		}
	})

	t.Run("Insufficient depth", func(t *testing.T) {
		orderBooks := map[string]types.OrderBook{
			"BTCUSDT": {Asks: []types.Level{}, Bids: []types.Level{}},
			"ETHBTC":  {Asks: []types.Level{{Price: 0.03, Qty: 10.0}}, Bids: []types.Level{{Price: 0.029, Qty: 10.0}}},
			"ETHUSDT": {Asks: []types.Level{{Price: 1500.0, Qty: 10.0}}, Bids: []types.Level{{Price: 1490.0, Qty: 10.0}}},
		}

		obBySymbol := func(symbol string) (types.OrderBook, bool) {
			ob, exists := orderBooks[symbol]
			return ob, exists
		}

		feeBySymbol := func(symbol string) (types.Fee, bool) {
			return types.Fee{TakerBp: 0.1, MakerBp: 0.05}, true
		}

		_, found := sim.EvaluateOrderBook(triangle, markets, obBySymbol, feeBySymbol, 1000.0, 5)

		if found {
			t.Error("Should not find arbitrage with insufficient order book depth")
		}
	})
}

func TestTOBSimulatorEvaluateOrderBookIRTArbitrage(t *testing.T) {
	sim := NewTOBSimulator(0.001, 5.0)

	markets := []types.Market{
		{Exchange: "nobitex", Symbol: "BTCUSDT", Base: "BTC", Quote: "USDT"},
		{Exchange: "nobitex", Symbol: "BTCIRT", Base: "BTC", Quote: "IRT"},
		{Exchange: "nobitex", Symbol: "USDTIRT", Base: "USDT", Quote: "IRT"},
	}

	triangle := types.Triangle{
		MarketIds: [3]int{0, 1, 2},
		Dirs:      [3]int8{1, -1, -1},
		QuoteCcy:  "USDT",
	}

	orderBooks := map[string]types.OrderBook{
		"BTCUSDT": {
			Asks: []types.Level{
				{Price: 45000.0, Qty: 0.1}, {Price: 45100.0, Qty: 0.2}, {Price: 45200.0, Qty: 0.15},
			},
			Bids: []types.Level{
				{Price: 44900.0, Qty: 0.12}, {Price: 44800.0, Qty: 0.18}, {Price: 44700.0, Qty: 0.14},
			},
		},
		"BTCIRT": {
			Asks: []types.Level{
				{Price: 2000000000.0, Qty: 0.05}, {Price: 2010000000.0, Qty: 0.08}, {Price: 2020000000.0, Qty: 0.06},
			},
			Bids: []types.Level{
				{Price: 1990000000.0, Qty: 0.07}, {Price: 1980000000.0, Qty: 0.09}, {Price: 1970000000.0, Qty: 0.08},
			},
		},
		"USDTIRT": {
			Asks: []types.Level{
				{Price: 400000.0, Qty: 50.0}, {Price: 401000.0, Qty: 75.0}, {Price: 402000.0, Qty: 60.0},
			},
			Bids: []types.Level{
				{Price: 399000.0, Qty: 55.0}, {Price: 398000.0, Qty: 80.0}, {Price: 397000.0, Qty: 65.0},
			},
		},
	}

	obBySymbol := func(symbol string) (types.OrderBook, bool) {
		ob, exists := orderBooks[symbol]
		return ob, exists
	}

		feeBySymbol := func(symbol string) (types.Fee, bool) {
			if symbol == "BTCIRT" || symbol == "USDTIRT" {
				return types.Fee{TakerBp: 25.0, MakerBp: 15.0}, true
			}
			return types.Fee{TakerBp: 8.0, MakerBp: 4.0}, true
		}

	targetQuote := 1000.0
	maxDepth := 3

	plan, found := sim.EvaluateOrderBook(triangle, markets, obBySymbol, feeBySymbol, targetQuote, maxDepth)

	t.Logf("IRT Arbitrage found: %v, profit: %f", found, plan.ExpectedProfitQuote)

	if found {
		if plan.ExpectedProfitQuote <= 0 {
			t.Errorf("Found IRT arbitrage but profit is not positive: %f", plan.ExpectedProfitQuote)
		}

		hasIRT := false
		for _, leg := range plan.Legs {
			if leg.Market == "BTCIRT" || leg.Market == "USDTIRT" {
				hasIRT = true
				break
			}
		}

		if !hasIRT {
			t.Error("Expected IRT arbitrage to involve IRT markets")
		}

		t.Logf("IRT Arbitrage legs:")
		for i, leg := range plan.Legs {
			t.Logf("  Leg %d: %s %s %.6f @ %.2f", i+1, leg.Side, leg.Market, leg.Qty, leg.LimitPrice)
		}
	}
}

func BenchmarkTOBSimulatorEvaluateOrderBook(b *testing.B) {
	sim := NewTOBSimulator(0.001, 5.0)

	markets := []types.Market{
		{Exchange: "binance", Symbol: "BTCUSDT", Base: "BTC", Quote: "USDT"},
		{Exchange: "binance", Symbol: "ETHBTC", Base: "ETH", Quote: "BTC"},
		{Exchange: "binance", Symbol: "ETHUSDT", Base: "ETH", Quote: "USDT"},
	}

	triangle := types.Triangle{
		MarketIds: [3]int{0, 1, 2},
		Dirs:      [3]int8{1, -1, -1},
		QuoteCcy:  "USDT",
	}

	orderBooks := map[string]types.OrderBook{
		"BTCUSDT": {
			Asks: []types.Level{{Price: 50000.0, Qty: 2.0}, {Price: 50010.0, Qty: 1.5}, {Price: 50020.0, Qty: 3.0}},
			Bids: []types.Level{{Price: 49990.0, Qty: 2.1}, {Price: 49980.0, Qty: 1.8}, {Price: 49970.0, Qty: 2.5}},
		},
		"ETHBTC": {
			Asks: []types.Level{{Price: 0.0300, Qty: 50.0}, {Price: 0.0301, Qty: 45.0}, {Price: 0.0302, Qty: 55.0}},
			Bids: []types.Level{{Price: 0.0299, Qty: 52.0}, {Price: 0.0298, Qty: 48.0}, {Price: 0.0297, Qty: 58.0}},
		},
		"ETHUSDT": {
			Asks: []types.Level{{Price: 1490.0, Qty: 10.0}, {Price: 1491.0, Qty: 8.0}, {Price: 1492.0, Qty: 12.0}},
			Bids: []types.Level{{Price: 1489.0, Qty: 9.0}, {Price: 1488.0, Qty: 11.0}, {Price: 1487.0, Qty: 13.0}},
		},
	}

	obBySymbol := func(symbol string) (types.OrderBook, bool) {
		ob, exists := orderBooks[symbol]
		return ob, exists
	}

	feeBySymbol := func(symbol string) (types.Fee, bool) {
		return types.Fee{TakerBp: 0.1, MakerBp: 0.05}, true
	}

	targetQuote := 1000.0
	maxDepth := 3

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sim.EvaluateOrderBook(triangle, markets, obBySymbol, feeBySymbol, targetQuote, maxDepth)
	}
}

func BenchmarkIsFinite(b *testing.B) {
	values := []float64{1.5, 0.0, -2.5, math.Inf(1), math.Inf(-1), math.NaN()}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, v := range values {
			isFinite(v)
		}
	}
}
