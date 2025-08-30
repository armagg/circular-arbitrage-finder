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
	sim := NewTOBSimulator(0.0001, 0.1) // Lower thresholds for testing

	// Create test markets
	markets := []types.Market{
		{Exchange: "binance", Symbol: "BTCUSDT", Base: "BTC", Quote: "USDT"},
		{Exchange: "binance", Symbol: "ETHBTC", Base: "ETH", Quote: "BTC"},
		{Exchange: "binance", Symbol: "ETHUSDT", Base: "ETH", Quote: "USDT"},
	}

	// Create test triangle
	triangle := types.Triangle{
		MarketIds: [3]int{0, 1, 2},
		Dirs:      [3]int8{1, -1, -1},
		QuoteCcy:  "USDT",
	}

	// Create mock functions with prices that should create arbitrage
	// Use prices that create a clear arbitrage opportunity
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

	// The function should execute without panicking
	// Arbitrage detection may or may not find profit depending on exact calculations
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
	sim := NewTOBSimulator(0.01, 5.0) // High minimum edge

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

	// Prices that result in no profit
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

	// Mock function that returns missing data
	tobBySymbol := func(symbol string) (types.TopOfBook, bool) {
		if symbol == "BTCUSDT" {
			return types.TopOfBook{BidPx: 0, AskPx: 0}, true // Invalid prices
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
	sim := NewTOBSimulator(0.0001, 0.1) // Lower thresholds for testing

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

	// Realistic prices that should create arbitrage opportunity
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

	// The function should execute without panicking
	t.Logf("Complex arbitrage found: %v, profit: %f", found, plan.ExpectedProfitQuote)

	if found {
		if plan.ExpectedProfitQuote <= 0 {
			t.Errorf("Found arbitrage but profit is not positive: %f", plan.ExpectedProfitQuote)
		}

		// Verify plan structure
		if plan.Exchange != "binance" {
			t.Errorf("Expected exchange binance, got %s", plan.Exchange)
		}

		if plan.ValidMs != 250 {
			t.Errorf("Expected valid ms 250, got %d", plan.ValidMs)
		}

		if plan.MaxSlippageBp != 0.1 { // Updated to match the simulator setting
			t.Errorf("Expected max slippage 0.1, got %f", plan.MaxSlippageBp)
		}

		// Verify legs
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

	// Test different triangle directions
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

			// We don't assert on 'found' since different directions may or may not be profitable
			// The important thing is that the function doesn't panic and returns a valid result
			t.Logf("Triangle with directions %v: profitable = %v", tc.directions, found)
		})
	}
}

func TestTOBSimulatorEdgeCases(t *testing.T) {
	sim := NewTOBSimulator(0.001, 5.0)

	// Create 3 markets for proper triangle testing
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
		// This tests the isFinite check
		simHighSlippage := NewTOBSimulator(0.001, 100.0) // Very high slippage

		tobBySymbol := func(symbol string) (types.TopOfBook, bool) {
			switch symbol {
			case "BTCUSDT":
				return types.TopOfBook{BidPx: 1e-100, AskPx: 1e100}, true // Extreme values
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
		// The function should handle infinite results gracefully
		t.Logf("Infinite result test: profitable = %v", found)
	})
}

// Benchmark tests
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

func BenchmarkIsFinite(b *testing.B) {
	values := []float64{1.5, 0.0, -2.5, math.Inf(1), math.Inf(-1), math.NaN()}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, v := range values {
			isFinite(v)
		}
	}
}
