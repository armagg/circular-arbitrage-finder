package types

import (
	"testing"
)

func TestSideConstants(t *testing.T) {
	tests := []struct {
		name     string
		side     Side
		expected string
	}{
		{"SideBuy constant", SideBuy, "BUY"},
		{"SideSell constant", SideSell, "SELL"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.side) != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, string(tt.side))
			}
		})
	}
}

func TestMarket(t *testing.T) {
	market := Market{
		Exchange:    "binance",
		Symbol:      "BTCUSDT",
		Base:        "BTC",
		Quote:       "USDT",
		Multiplier:  100000000,
		MinQty:      0.0001,
		StepSize:    0.0001,
		MinNotional: 10.0,
		PriceTick:   0.01,
	}

	if market.Exchange != "binance" {
		t.Errorf("Expected exchange binance, got %s", market.Exchange)
	}
	if market.Symbol != "BTCUSDT" {
		t.Errorf("Expected symbol BTCUSDT, got %s", market.Symbol)
	}
	if market.Base != "BTC" {
		t.Errorf("Expected base BTC, got %s", market.Base)
	}
	if market.Quote != "USDT" {
		t.Errorf("Expected quote USDT, got %s", market.Quote)
	}
	if market.Multiplier != 100000000 {
		t.Errorf("Expected multiplier 100000000, got %d", market.Multiplier)
	}
}

func TestFee(t *testing.T) {
	fee := Fee{
		TakerBp: 0.1,
		MakerBp: 0.05,
	}

	if fee.TakerBp != 0.1 {
		t.Errorf("Expected taker 0.1, got %f", fee.TakerBp)
	}
	if fee.MakerBp != 0.05 {
		t.Errorf("Expected maker 0.05, got %f", fee.MakerBp)
	}
}

func TestLevel(t *testing.T) {
	level := Level{
		Price: 50000.0,
		Qty:   1.5,
	}

	if level.Price != 50000.0 {
		t.Errorf("Expected price 50000.0, got %f", level.Price)
	}
	if level.Qty != 1.5 {
		t.Errorf("Expected qty 1.5, got %f", level.Qty)
	}
}

func TestTopOfBook(t *testing.T) {
	tob := TopOfBook{
		BidPx: 49990.0,
		BidSz: 2.1,
		AskPx: 50010.0,
		AskSz: 1.8,
		Seq:   12345,
		TsNs:  1640995200000000000,
	}

	if tob.BidPx != 49990.0 {
		t.Errorf("Expected bid price 49990.0, got %f", tob.BidPx)
	}
	if tob.BidSz != 2.1 {
		t.Errorf("Expected bid size 2.1, got %f", tob.BidSz)
	}
	if tob.AskPx != 50010.0 {
		t.Errorf("Expected ask price 50010.0, got %f", tob.AskPx)
	}
	if tob.AskSz != 1.8 {
		t.Errorf("Expected ask size 1.8, got %f", tob.AskSz)
	}
	if tob.Seq != 12345 {
		t.Errorf("Expected seq 12345, got %d", tob.Seq)
	}
	if tob.TsNs != 1640995200000000000 {
		t.Errorf("Expected timestamp 1640995200000000000, got %d", tob.TsNs)
	}
}

func TestOrderBook(t *testing.T) {
	bids := []Level{
		{Price: 49990.0, Qty: 2.1},
		{Price: 49980.0, Qty: 1.5},
	}
	asks := []Level{
		{Price: 50010.0, Qty: 1.8},
		{Price: 50020.0, Qty: 2.2},
	}

	orderBook := OrderBook{
		Bids: bids,
		Asks: asks,
		Seq:  12345,
		TsNs: 1640995200000000000,
	}

	if len(orderBook.Bids) != 2 {
		t.Errorf("Expected 2 bids, got %d", len(orderBook.Bids))
	}
	if len(orderBook.Asks) != 2 {
		t.Errorf("Expected 2 asks, got %d", len(orderBook.Asks))
	}
	if orderBook.Seq != 12345 {
		t.Errorf("Expected seq 12345, got %d", orderBook.Seq)
	}
	if orderBook.TsNs != 1640995200000000000 {
		t.Errorf("Expected timestamp 1640995200000000000, got %d", orderBook.TsNs)
	}

	// Test bid ordering (should be descending)
	if orderBook.Bids[0].Price <= orderBook.Bids[1].Price {
		t.Error("Bids should be ordered descending by price")
	}

	// Test ask ordering (should be ascending)
	if orderBook.Asks[0].Price >= orderBook.Asks[1].Price {
		t.Error("Asks should be ordered ascending by price")
	}
}

func TestTriangle(t *testing.T) {
	triangle := Triangle{
		MarketIds: [3]int{0, 1, 2},
		Dirs:      [3]int8{1, -1, -1},
		QuoteCcy:  "USDT",
	}

	expectedMarketIds := [3]int{0, 1, 2}
	expectedDirs := [3]int8{1, -1, -1}

	for i, id := range triangle.MarketIds {
		if id != expectedMarketIds[i] {
			t.Errorf("Expected market ID %d at position %d, got %d", expectedMarketIds[i], i, id)
		}
	}

	for i, dir := range triangle.Dirs {
		if dir != expectedDirs[i] {
			t.Errorf("Expected direction %d at position %d, got %d", expectedDirs[i], i, dir)
		}
	}

	if triangle.QuoteCcy != "USDT" {
		t.Errorf("Expected quote currency USDT, got %s", triangle.QuoteCcy)
	}
}

func TestTriangleLeg(t *testing.T) {
	leg := TriangleLeg{
		Market:     "BTCUSDT",
		Side:       SideBuy,
		Qty:        1.5,
		LimitPrice: 50000.0,
	}

	if leg.Market != "BTCUSDT" {
		t.Errorf("Expected market BTCUSDT, got %s", leg.Market)
	}
	if leg.Side != SideBuy {
		t.Errorf("Expected side BUY, got %s", string(leg.Side))
	}
	if leg.Qty != 1.5 {
		t.Errorf("Expected qty 1.5, got %f", leg.Qty)
	}
	if leg.LimitPrice != 50000.0 {
		t.Errorf("Expected limit price 50000.0, got %f", leg.LimitPrice)
	}
}

func TestPlan(t *testing.T) {
	legs := [3]TriangleLeg{
		{Market: "BTCUSDT", Side: SideBuy, Qty: 1.5, LimitPrice: 50000.0},
		{Market: "ETHBTC", Side: SideSell, Qty: 1.5, LimitPrice: 0.03},
		{Market: "ETHUSDT", Side: SideSell, Qty: 50.0, LimitPrice: 1500.0},
	}

	plan := Plan{
		Exchange:            "binance",
		Legs:                legs,
		ExpectedProfitQuote: 25.5,
		QuoteCurrency:       "USDT",
		ValidMs:             250,
		MaxSlippageBp:       5.0,
		PlanID:              "test-plan-123",
	}

	if plan.Exchange != "binance" {
		t.Errorf("Expected exchange binance, got %s", plan.Exchange)
	}
	if len(plan.Legs) != 3 {
		t.Errorf("Expected 3 legs, got %d", len(plan.Legs))
	}
	if plan.ExpectedProfitQuote != 25.5 {
		t.Errorf("Expected profit 25.5, got %f", plan.ExpectedProfitQuote)
	}
	if plan.QuoteCurrency != "USDT" {
		t.Errorf("Expected quote currency USDT, got %s", plan.QuoteCurrency)
	}
	if plan.ValidMs != 250 {
		t.Errorf("Expected valid ms 250, got %d", plan.ValidMs)
	}
	if plan.MaxSlippageBp != 5.0 {
		t.Errorf("Expected max slippage 5.0, got %f", plan.MaxSlippageBp)
	}
	if plan.PlanID != "test-plan-123" {
		t.Errorf("Expected plan ID test-plan-123, got %s", plan.PlanID)
	}
}

// Test data integrity and serialization compatibility
func TestTypeCompatibility(t *testing.T) {
	// Test that all types can be created and compared
	market1 := Market{Exchange: "test", Symbol: "TEST"}
	market2 := Market{Exchange: "test", Symbol: "TEST"}

	if market1 != market2 {
		t.Error("Identical markets should be equal")
	}

	fee1 := Fee{TakerBp: 0.1, MakerBp: 0.05}
	fee2 := Fee{TakerBp: 0.1, MakerBp: 0.05}

	if fee1 != fee2 {
		t.Error("Identical fees should be equal")
	}

	level1 := Level{Price: 100.0, Qty: 1.0}
	level2 := Level{Price: 100.0, Qty: 1.0}

	if level1 != level2 {
		t.Error("Identical levels should be equal")
	}
}
