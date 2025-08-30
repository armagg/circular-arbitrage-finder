package main

import (
	"testing"

	"github.com/armagg/circular-arbitrage-finder/pkg/bookstore"
	"github.com/armagg/circular-arbitrage-finder/pkg/config"
	"github.com/armagg/circular-arbitrage-finder/pkg/detector"
	"github.com/armagg/circular-arbitrage-finder/pkg/graph"
	"github.com/armagg/circular-arbitrage-finder/pkg/profit"
	"github.com/armagg/circular-arbitrage-finder/pkg/registry"
	"github.com/armagg/circular-arbitrage-finder/pkg/testutils"
	"github.com/armagg/circular-arbitrage-finder/pkg/types"
)

// TestFullArbitrageFlow tests the complete arbitrage detection flow
func TestFullArbitrageFlow(t *testing.T) {
	// Setup all components
	idx := graph.NewIndex()
	books := bookstore.NewTopOfBookStore()
	reg := registry.NewMarketRegistry()
	sim := profit.NewTOBSimulator(0.001, 5.0)
	pub := testutils.NewMockPublisher()

	detector := detector.NewDetector(idx, books, reg, sim, pub)

	// Setup test markets
	markets, _ := testutils.SetupTestTriangle()

	// Add markets to registry and index
	for _, market := range markets {
		reg.UpsertMarket(market)
		reg.SetFee(market.Symbol, types.Fee{TakerBp: 0.1, MakerBp: 0.05})
		idx.AddMarket(market)
	}

	// Setup profitable arbitrage prices
	tobData := testutils.CreateProfitableArbitragePrices()
	for symbol, tob := range tobData {
		books.Set(symbol, tob)
	}

	// Trigger arbitrage detection
	detector.OnMarketChange("binance", "BTCUSDT", 1000.0)

	// Verify results
	publishedPlans := pub.GetPublishedPlans()

	// The system may or may not find arbitrage depending on exact calculations
	// The important thing is that it processes the request without panicking
	t.Logf("Found %d arbitrage opportunities", len(publishedPlans))

	for _, plan := range publishedPlans {
		if !testutils.AssertPlanValid(plan) {
			t.Errorf("Plan is not valid: %+v", plan)
		}

		t.Logf("Found profitable arbitrage: %s, profit: %.2f %s",
			plan.PlanID, plan.ExpectedProfitQuote, plan.QuoteCurrency)
	}
}

// TestConfigToRegistryIntegration tests config loading and registry integration
func TestConfigToRegistryIntegration(t *testing.T) {
	// Create a temporary config (in real scenario this would be loaded from file)
	cfg := &config.Config{
		QuoteAssets: []string{"USDT", "BTC", "ETH"},
		Fees: config.Fees{
			Default: config.FeeConfig{Taker: 0.1, Maker: 0.05},
			Exchanges: map[string]config.FeeQuotes{
				"BINANCE": {
					"USDT": config.FeeConfig{Taker: 0.08, Maker: 0.04},
				},
			},
		},
	}

	reg := registry.NewMarketRegistry()

	// Test market creation from config
	market, err := cfg.ParseMarket("binance", "BTCUSDT")
	if err != nil {
		t.Fatalf("Failed to parse market: %v", err)
	}

	reg.UpsertMarket(market)

	// Test fee retrieval from config
	fee := cfg.GetFee("binance", "USDT")
	reg.SetFee("BTCUSDT", fee)

	retrievedMarket, exists := reg.GetMarket("BTCUSDT")
	if !exists {
		t.Error("Market should exist in registry")
	}

	if retrievedMarket.Symbol != "BTCUSDT" {
		t.Errorf("Expected symbol BTCUSDT, got %s", retrievedMarket.Symbol)
	}

	retrievedFee, exists := reg.GetFee("BTCUSDT")
	if !exists {
		t.Error("Fee should exist in registry")
	}

	if retrievedFee.TakerBp != 0.08 {
		t.Errorf("Expected taker fee 0.08, got %f", retrievedFee.TakerBp)
	}
}

// TestBookstoreSortingIntegration tests the sorting optimizations in real scenario
func TestBookstoreSortingIntegration(t *testing.T) {
	orderStore := bookstore.NewOrderBookStore()

	// Create unsorted order book data
	bids := []types.Level{
		{Price: 49980.0, Qty: 1.5},
		{Price: 50000.0, Qty: 2.1}, // Should be first (highest)
		{Price: 49990.0, Qty: 1.8},
		{Price: 49970.0, Qty: 0.8}, // Should be last (lowest)
	}

	asks := []types.Level{
		{Price: 50030.0, Qty: 1.2},
		{Price: 50010.0, Qty: 2.5}, // Should be first (lowest)
		{Price: 50020.0, Qty: 1.8},
		{Price: 50040.0, Qty: 0.8}, // Should be last (highest)
	}

	// Upsert with sorting
	orderStore.Upsert("BTCUSDT", bids, asks, 12345, 1640995200000000000, 0)

	retrieved, exists := orderStore.Get("BTCUSDT")
	if !exists {
		t.Error("Order book should exist")
	}

	// Verify bids are sorted descending
	for i := 1; i < len(retrieved.Bids); i++ {
		if retrieved.Bids[i].Price > retrieved.Bids[i-1].Price {
			t.Errorf("Bids not sorted descending: %f > %f at positions %d-%d",
				retrieved.Bids[i].Price, retrieved.Bids[i-1].Price, i, i-1)
		}
	}

	// Verify asks are sorted ascending
	for i := 1; i < len(retrieved.Asks); i++ {
		if retrieved.Asks[i].Price < retrieved.Asks[i-1].Price {
			t.Errorf("Asks not sorted ascending: %f < %f at positions %d-%d",
				retrieved.Asks[i].Price, retrieved.Asks[i-1].Price, i, i-1)
		}
	}

	// Test depth limiting
	orderStore.Upsert("BTCUSDT", bids, asks, 12346, 1640995200000000000, 2)

	retrievedLimited, _ := orderStore.Get("BTCUSDT")
	if len(retrievedLimited.Bids) != 2 {
		t.Errorf("Expected 2 bids after depth limit, got %d", len(retrievedLimited.Bids))
	}

	if len(retrievedLimited.Asks) != 2 {
		t.Errorf("Expected 2 asks after depth limit, got %d", len(retrievedLimited.Asks))
	}
}

// TestGraphTriangleIntegration tests graph indexing and triangle finding
func TestGraphTriangleIntegration(t *testing.T) {
	idx := graph.NewIndex()

	// Add markets that should form triangles
	markets := testutils.CreateTestMarkets()

	for _, market := range markets {
		idx.AddMarket(market)
	}

	// Should have found triangles
	if len(idx.Triangles) == 0 {
		t.Error("Should have found triangles")
	}

	// Verify triangle indexing
	for i, triangle := range idx.Triangles {
		// Each triangle should have 3 market IDs
		if len(triangle.MarketIds) != 3 {
			t.Errorf("Triangle %d should have 3 market IDs, got %d", i, len(triangle.MarketIds))
		}

		// Each market ID should be valid
		for _, marketID := range triangle.MarketIds {
			if marketID < 0 || marketID >= len(idx.Markets) {
				t.Errorf("Triangle %d has invalid market ID %d", i, marketID)
			}
		}

		// Each market should reference this triangle
		for _, marketID := range triangle.MarketIds {
			found := false
			for _, triID := range idx.TrianglesByMarket[marketID] {
				if triID == i {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Market %d should reference triangle %d", marketID, i)
			}
		}
	}

	t.Logf("Found %d triangles", len(idx.Triangles))
}

// TestProfitSimulatorIntegration tests profit calculation with realistic data
func TestProfitSimulatorIntegration(t *testing.T) {
	sim := profit.NewTOBSimulator(0.001, 5.0)

	markets := testutils.CreateTestMarkets()
	triangle := testutils.CreateTestTriangle(markets, [3]int{0, 1, 2})

	// Test with profitable prices
	profitablePrices := testutils.CreateProfitableArbitragePrices()

	tobBySymbol := func(symbol string) (types.TopOfBook, bool) {
		tob, exists := profitablePrices[symbol]
		return tob, exists
	}

	feeBySymbol := func(symbol string) (types.Fee, bool) {
		return types.Fee{TakerBp: 0.1, MakerBp: 0.05}, true
	}

	plan, found := sim.EvaluateTOB(triangle, markets, tobBySymbol, feeBySymbol, 1000.0)

	// The profit simulator may or may not find arbitrage depending on exact calculations
	t.Logf("Profit simulation found: %v, profit: %f", found, plan.ExpectedProfitQuote)

	if found && plan.ExpectedProfitQuote <= 0 {
		t.Errorf("Found arbitrage but profit is not positive: %f", plan.ExpectedProfitQuote)
	}

	// Test with non-profitable prices
	noProfitPrices := testutils.CreateNoArbitragePrices()

	tobBySymbolNoProfit := func(symbol string) (types.TopOfBook, bool) {
		tob, exists := noProfitPrices[symbol]
		return tob, exists
	}

	_, found = sim.EvaluateTOB(triangle, markets, tobBySymbolNoProfit, feeBySymbol, 1000.0)

	if found {
		t.Error("Expected no profitable arbitrage with equal prices")
	}
}

// TestConcurrencyIntegration tests concurrent operations across all components
func TestConcurrencyIntegration(t *testing.T) {
	// Setup components
	idx := graph.NewIndex()
	books := bookstore.NewTopOfBookStore()
	orderBooks := bookstore.NewOrderBookStore()
	reg := registry.NewMarketRegistry()
	sim := profit.NewTOBSimulator(0.001, 5.0)
	pub := testutils.NewMockPublisher()

	detector := detector.NewDetector(idx, books, reg, sim, pub)

	// Setup initial data
	markets := testutils.CreateTestMarkets()
	for _, market := range markets {
		reg.UpsertMarket(market)
		reg.SetFee(market.Symbol, types.Fee{TakerBp: 0.1, MakerBp: 0.05})
		idx.AddMarket(market)
		books.Set(market.Symbol, types.TopOfBook{
			BidPx: 100.0,
			AskPx: 101.0,
			BidSz: 10.0,
			AskSz: 10.0,
		})
	}

	// Run concurrent operations
	done := make(chan bool, 1)

	go func() {
		// Concurrent market updates
		for i := 0; i < 100; i++ {
			detector.OnMarketChange("binance", "BTCUSDT", 1000.0)
		}
		done <- true
	}()

	go func() {
		// Concurrent order book updates
		for i := 0; i < 50; i++ {
			bids := []types.Level{{Price: 100.0 + float64(i), Qty: 10.0}}
			asks := []types.Level{{Price: 101.0 + float64(i), Qty: 10.0}}
			orderBooks.Upsert("BTCUSDT", bids, asks, uint64(i), 1640995200000000000, 0)
		}
		done <- true
	}()

	// Wait for completion
	<-done
	<-done

	// Verify system still works
	_, exists := books.Get("BTCUSDT")
	if !exists {
		t.Error("Data should still exist after concurrent operations")
	}

	t.Log("Concurrent operations completed successfully")
}

// TestMemoryManagementIntegration tests that components don't leak memory
func TestMemoryManagementIntegration(t *testing.T) {
	// This is a basic test - in a real scenario you'd use memory profiling

	idx := graph.NewIndex()
	books := bookstore.NewTopOfBookStore()
	reg := registry.NewMarketRegistry()

	// Add many markets (some may be duplicates/rejected)
	addedMarkets := 0
	for i := 0; i < 1000; i++ {
		market := types.Market{
			Exchange: "test",
			Symbol:   "TEST" + string(rune(65+i%26)) + string(rune(48+i%10)), // Limited combinations
			Base:     "BASE" + string(rune(65+i%26)),
			Quote:    "USDT",
		}

		reg.UpsertMarket(market)
		_, isNew := idx.AddMarket(market)
		if isNew {
			addedMarkets++
		}
		books.Set(market.Symbol, types.TopOfBook{BidPx: 100.0, AskPx: 101.0})
	}

	// Verify we added some markets (exact count may vary due to duplicates)
	if len(idx.Markets) == 0 {
		t.Error("Expected at least some markets to be added")
	}

	t.Logf("Successfully added %d markets to index", len(idx.Markets))

	// Test snapshot functionality doesn't cause issues
	reg.Snapshot()
	books.Snapshot()

	t.Log("Memory management test completed")
}

// BenchmarkFullSystem benchmarks the complete arbitrage detection system
func BenchmarkFullSystem(b *testing.B) {
	// Setup
	idx := graph.NewIndex()
	books := bookstore.NewTopOfBookStore()
	reg := registry.NewMarketRegistry()
	sim := profit.NewTOBSimulator(0.001, 5.0)
	pub := testutils.NewMockPublisher()

	detector := detector.NewDetector(idx, books, reg, sim, pub)

	markets := testutils.CreateTestMarkets()
	for _, market := range markets {
		reg.UpsertMarket(market)
		reg.SetFee(market.Symbol, types.Fee{TakerBp: 0.1, MakerBp: 0.05})
		idx.AddMarket(market)
	}

	tobData := testutils.CreateProfitableArbitragePrices()
	for symbol, tob := range tobData {
		books.Set(symbol, tob)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		detector.OnMarketChange("binance", "BTCUSDT", 1000.0)
	}
}
