package registry

import (
	"sync"
	"testing"

	"github.com/armagg/circular-arbitrage-finder/pkg/types"
)

func TestNewMarketRegistry(t *testing.T) {
	reg := NewMarketRegistry()

	if reg == nil {
		t.Error("NewMarketRegistry should return a non-nil registry")
	}

	if reg.markets == nil {
		t.Error("Markets map should be initialized")
	}

	if len(reg.markets) != 0 {
		t.Errorf("New registry should have empty markets map, got %d items", len(reg.markets))
	}

	if reg.fees == nil {
		t.Error("Fees map should be initialized")
	}

	if len(reg.fees) != 0 {
		t.Errorf("New registry should have empty fees map, got %d items", len(reg.fees))
	}
}

func TestMarketRegistryUpsertMarket(t *testing.T) {
	reg := NewMarketRegistry()

	market := types.Market{
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

	reg.UpsertMarket(market)

	// Verify the market was stored
	retrieved, exists := reg.GetMarket("BTCUSDT")
	if !exists {
		t.Error("Expected market to exist after upsert")
	}

	if retrieved != market {
		t.Error("Retrieved market should match stored market")
	}
}

func TestMarketRegistryUpsertMarketUpdate(t *testing.T) {
	reg := NewMarketRegistry()

	symbol := "BTCUSDT"

	// Insert initial market
	market1 := types.Market{
		Exchange: "binance",
		Symbol:   symbol,
		Base:     "BTC",
		Quote:    "USDT",
		MinQty:   0.001,
	}

	reg.UpsertMarket(market1)

	// Update the market
	market2 := types.Market{
		Exchange: "binance",
		Symbol:   symbol,
		Base:     "BTC",
		Quote:    "USDT",
		MinQty:   0.0001, // Different value
	}

	reg.UpsertMarket(market2)

	// Verify the update
	retrieved, exists := reg.GetMarket(symbol)
	if !exists {
		t.Error("Expected market to exist after update")
	}

	if retrieved.MinQty != market2.MinQty {
		t.Error("Market should be updated with new values")
	}
}

func TestMarketRegistryGetMarket(t *testing.T) {
	reg := NewMarketRegistry()

	// Test getting non-existent market
	_, exists := reg.GetMarket("NONEXISTENT")
	if exists {
		t.Error("GetMarket should return false for non-existent market")
	}

	// Test getting existing market
	market := types.Market{
		Exchange: "binance",
		Symbol:   "BTCUSDT",
		Base:     "BTC",
		Quote:    "USDT",
	}

	reg.UpsertMarket(market)

	retrieved, exists := reg.GetMarket("BTCUSDT")
	if !exists {
		t.Error("GetMarket should return true for existing market")
	}

	if retrieved != market {
		t.Error("Retrieved market should match stored market")
	}
}

func TestMarketRegistrySetFee(t *testing.T) {
	reg := NewMarketRegistry()

	fee := types.Fee{
		TakerBp: 0.1,
		MakerBp: 0.05,
	}

	reg.SetFee("BTCUSDT", fee)

	// Verify the fee was stored
	retrieved, exists := reg.GetFee("BTCUSDT")
	if !exists {
		t.Error("Expected fee to exist after SetFee")
	}

	if retrieved != fee {
		t.Error("Retrieved fee should match stored fee")
	}
}

func TestMarketRegistryGetFee(t *testing.T) {
	reg := NewMarketRegistry()

	// Test getting non-existent fee
	_, exists := reg.GetFee("NONEXISTENT")
	if exists {
		t.Error("GetFee should return false for non-existent fee")
	}

	// Test getting existing fee
	fee := types.Fee{
		TakerBp: 0.08,
		MakerBp: 0.04,
	}

	reg.SetFee("BTCUSDT", fee)

	retrieved, exists := reg.GetFee("BTCUSDT")
	if !exists {
		t.Error("GetFee should return true for existing fee")
	}

	if retrieved != fee {
		t.Error("Retrieved fee should match stored fee")
	}
}

func TestMarketRegistrySnapshot(t *testing.T) {
	reg := NewMarketRegistry()

	// Add some test data
	markets := map[string]types.Market{
		"BTCUSDT": {Exchange: "binance", Symbol: "BTCUSDT", Base: "BTC", Quote: "USDT"},
		"ETHUSDT": {Exchange: "binance", Symbol: "ETHUSDT", Base: "ETH", Quote: "USDT"},
	}

	fees := map[string]types.Fee{
		"BTCUSDT": {TakerBp: 0.1, MakerBp: 0.05},
		"ETHUSDT": {TakerBp: 0.08, MakerBp: 0.04},
	}

	for _, market := range markets {
		reg.UpsertMarket(market)
	}

	for symbol, fee := range fees {
		reg.SetFee(symbol, fee)
	}

	marketSnapshot, feeSnapshot := reg.Snapshot()

	if len(marketSnapshot) != len(markets) {
		t.Errorf("Market snapshot should have %d items, got %d", len(markets), len(marketSnapshot))
	}

	if len(feeSnapshot) != len(fees) {
		t.Errorf("Fee snapshot should have %d items, got %d", len(fees), len(feeSnapshot))
	}

	for symbol, expectedMarket := range markets {
		actualMarket, exists := marketSnapshot[symbol]
		if !exists {
			t.Errorf("Market snapshot should contain symbol %s", symbol)
		}
		if actualMarket != expectedMarket {
			t.Errorf("Market snapshot data for %s should match original", symbol)
		}
	}

	for symbol, expectedFee := range fees {
		actualFee, exists := feeSnapshot[symbol]
		if !exists {
			t.Errorf("Fee snapshot should contain symbol %s", symbol)
		}
		if actualFee != expectedFee {
			t.Errorf("Fee snapshot data for %s should match original", symbol)
		}
	}
}

func TestMarketRegistrySnapshotEmpty(t *testing.T) {
	reg := NewMarketRegistry()

	marketSnapshot, feeSnapshot := reg.Snapshot()

	if len(marketSnapshot) != 0 {
		t.Errorf("Empty registry market snapshot should have 0 items, got %d", len(marketSnapshot))
	}

	if len(feeSnapshot) != 0 {
		t.Errorf("Empty registry fee snapshot should have 0 items, got %d", len(feeSnapshot))
	}
}

func TestMarketRegistryConcurrency(t *testing.T) {
	reg := NewMarketRegistry()

	// Test concurrent operations
	var wg sync.WaitGroup
	numGoroutines := 10
	numOperations := 100

	// Test concurrent market operations
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				symbol := "TEST" + string(rune(id*numOperations+j+65))
				market := types.Market{
					Exchange: "binance",
					Symbol:   symbol,
					Base:     "BASE" + string(rune(id*numOperations+j+65)),
					Quote:    "USDT",
				}
				reg.UpsertMarket(market)

				_, _ = reg.GetMarket(symbol)
			}
		}(i)
	}

	// Test concurrent fee operations
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				symbol := "FEE" + string(rune(id*numOperations+j+65))
				fee := types.Fee{
					TakerBp: float64(id*j) * 0.001,
					MakerBp: float64(id*j) * 0.0005,
				}
				reg.SetFee(symbol, fee)

				_, _ = reg.GetFee(symbol)
			}
		}(i)
	}

	wg.Wait()

	// Verify final state
	marketSnapshot, feeSnapshot := reg.Snapshot()
	if len(marketSnapshot) == 0 {
		t.Error("Should have markets after concurrent operations")
	}

	if len(feeSnapshot) == 0 {
		t.Error("Should have fees after concurrent operations")
	}
}

func TestMarketRegistryDataIntegrity(t *testing.T) {
	reg := NewMarketRegistry()

	// Test that data remains consistent across operations
	symbol := "BTCUSDT"

	// Set initial data
	market := types.Market{
		Exchange: "binance",
		Symbol:   symbol,
		Base:     "BTC",
		Quote:    "USDT",
		MinQty:   0.001,
	}

	fee := types.Fee{
		TakerBp: 0.1,
		MakerBp: 0.05,
	}

	reg.UpsertMarket(market)
	reg.SetFee(symbol, fee)

	// Perform multiple reads
	for i := 0; i < 10; i++ {
		retrievedMarket, marketExists := reg.GetMarket(symbol)
		retrievedFee, feeExists := reg.GetFee(symbol)

		if !marketExists {
			t.Errorf("Market should exist in iteration %d", i)
		}

		if !feeExists {
			t.Errorf("Fee should exist in iteration %d", i)
		}

		if retrievedMarket != market {
			t.Errorf("Market data should be consistent in iteration %d", i)
		}

		if retrievedFee != fee {
			t.Errorf("Fee data should be consistent in iteration %d", i)
		}
	}

	// Test snapshot consistency
	for i := 0; i < 5; i++ {
		marketSnapshot, feeSnapshot := reg.Snapshot()

		if len(marketSnapshot) != 1 {
			t.Errorf("Market snapshot should have 1 item in iteration %d, got %d", i, len(marketSnapshot))
		}

		if len(feeSnapshot) != 1 {
			t.Errorf("Fee snapshot should have 1 item in iteration %d, got %d", i, len(feeSnapshot))
		}
	}
}

func TestMarketRegistryCaseSensitivity(t *testing.T) {
	reg := NewMarketRegistry()

	// Test that symbol lookup is case-sensitive
	reg.UpsertMarket(types.Market{
		Exchange: "binance",
		Symbol:   "BTCUSDT",
		Base:     "BTC",
		Quote:    "USDT",
	})

	reg.SetFee("btcusdt", types.Fee{TakerBp: 0.1, MakerBp: 0.05})

	// Should find with exact case
	_, marketExists := reg.GetMarket("BTCUSDT")
	if !marketExists {
		t.Error("Should find market with exact case")
	}

	// Should not find with different case (assuming case-sensitive)
	_, marketExistsLower := reg.GetMarket("btcusdt")
	if marketExistsLower {
		t.Error("Should not find market with different case")
	}

	_, feeExists := reg.GetFee("btcusdt")
	if !feeExists {
		t.Error("Should find fee with exact case")
	}

	_, feeExistsUpper := reg.GetFee("BTCUSDT")
	if feeExistsUpper {
		t.Error("Should not find fee with different case")
	}
}

func TestMarketRegistryZeroValues(t *testing.T) {
	reg := NewMarketRegistry()

	// Test with zero values
	market := types.Market{} // All fields zero
	fee := types.Fee{}       // All fields zero

	reg.UpsertMarket(market)
	reg.SetFee("ZERO", fee)

	retrievedMarket, marketExists := reg.GetMarket("")
	if !marketExists {
		t.Error("Should be able to store and retrieve market with zero values")
	}

	if retrievedMarket != market {
		t.Error("Zero value market should be stored and retrieved correctly")
	}

	retrievedFee, feeExists := reg.GetFee("ZERO")
	if !feeExists {
		t.Error("Should be able to store and retrieve fee with zero values")
	}

	if retrievedFee != fee {
		t.Error("Zero value fee should be stored and retrieved correctly")
	}
}

// Benchmark tests
func BenchmarkMarketRegistryUpsertMarket(b *testing.B) {
	reg := NewMarketRegistry()

	market := types.Market{
		Exchange: "binance",
		Symbol:   "BTCUSDT",
		Base:     "BTC",
		Quote:    "USDT",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reg.UpsertMarket(market)
	}
}

func BenchmarkMarketRegistryGetMarket(b *testing.B) {
	reg := NewMarketRegistry()

	market := types.Market{
		Exchange: "binance",
		Symbol:   "BTCUSDT",
		Base:     "BTC",
		Quote:    "USDT",
	}

	reg.UpsertMarket(market)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reg.GetMarket("BTCUSDT")
	}
}

func BenchmarkMarketRegistrySetFee(b *testing.B) {
	reg := NewMarketRegistry()

	fee := types.Fee{
		TakerBp: 0.1,
		MakerBp: 0.05,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reg.SetFee("BTCUSDT", fee)
	}
}

func BenchmarkMarketRegistrySnapshot(b *testing.B) {
	reg := NewMarketRegistry()

	// Add some test data
	for i := 0; i < 100; i++ {
		symbol := "TEST" + string(rune(i+65))
		reg.UpsertMarket(types.Market{
			Exchange: "binance",
			Symbol:   symbol,
			Base:     "BASE" + string(rune(i+65)),
			Quote:    "USDT",
		})
		reg.SetFee(symbol, types.Fee{TakerBp: 0.1, MakerBp: 0.05})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reg.Snapshot()
	}
}
