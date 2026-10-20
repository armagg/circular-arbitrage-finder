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

func TestFullArbitrageFlow(t *testing.T) {
	idx := graph.NewIndex()
	books := bookstore.NewTopOfBookStore()
	reg := registry.NewMarketRegistry()
	sim := profit.NewTOBSimulator(0.001, 5.0)
	pub := testutils.NewMockPublisher()

	detector := detector.NewDetector(idx, books, reg, sim, pub)

	markets, _ := testutils.SetupTestTriangle()

	for _, market := range markets {
		reg.UpsertMarket(market)
		reg.SetFee(market.Symbol, types.Fee{TakerBp: 0.1, MakerBp: 0.05})
		idx.AddMarket(market)
	}

	tobData := testutils.CreateProfitableArbitragePrices()
	for symbol, tob := range tobData {
		books.Set(symbol, tob)
	}

	detector.OnMarketChange("binance", "BTCUSDT", 1000.0)

	publishedPlans := pub.GetPublishedPlans()

	t.Logf("Found %d arbitrage opportunities", len(publishedPlans))

	if len(publishedPlans) == 0 {
		t.Error("Expected to find arbitrage opportunities with profitable test data")
	}

	for _, plan := range publishedPlans {
		if !testutils.AssertPlanValid(plan) {
			t.Errorf("Plan is not valid: %+v", plan)
		}

		if plan.ExpectedProfitQuote <= 0 {
			t.Errorf("Expected positive profit, got: %.2f", plan.ExpectedProfitQuote)
		}

		t.Logf("Found profitable arbitrage: %s, profit: %.2f %s",
			plan.PlanID, plan.ExpectedProfitQuote, plan.QuoteCurrency)
	}
}

func TestConfigToRegistryIntegration(t *testing.T) {
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

	market, err := cfg.ParseMarket("binance", "BTCUSDT")
	if err != nil {
		t.Fatalf("Failed to parse market: %v", err)
	}

	reg.UpsertMarket(market)

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

func TestBookstoreSortingIntegration(t *testing.T) {
	orderStore := bookstore.NewOrderBookStore()

	bids := []types.Level{
		{Price: 49980.0, Qty: 1.5},
		{Price: 50000.0, Qty: 2.1},
		{Price: 49990.0, Qty: 1.8},
		{Price: 49970.0, Qty: 0.8},
	}

	asks := []types.Level{
		{Price: 50030.0, Qty: 1.2},
		{Price: 50010.0, Qty: 2.5},
		{Price: 50020.0, Qty: 1.8},
		{Price: 50040.0, Qty: 0.8},
	}

	orderStore.Upsert("BTCUSDT", bids, asks, 12345, 1640995200000000000, 0)

	retrieved, exists := orderStore.Get("BTCUSDT")
	if !exists {
		t.Error("Order book should exist")
	}

	for i := 1; i < len(retrieved.Bids); i++ {
		if retrieved.Bids[i].Price > retrieved.Bids[i-1].Price {
			t.Errorf("Bids not sorted descending: %f > %f at positions %d-%d",
				retrieved.Bids[i].Price, retrieved.Bids[i-1].Price, i, i-1)
		}
	}

	for i := 1; i < len(retrieved.Asks); i++ {
		if retrieved.Asks[i].Price < retrieved.Asks[i-1].Price {
			t.Errorf("Asks not sorted ascending: %f < %f at positions %d-%d",
				retrieved.Asks[i].Price, retrieved.Asks[i-1].Price, i, i-1)
		}
	}

	orderStore.Upsert("BTCUSDT", bids, asks, 12346, 1640995200000000000, 2)

	retrievedLimited, _ := orderStore.Get("BTCUSDT")
	if len(retrievedLimited.Bids) != 2 {
		t.Errorf("Expected 2 bids after depth limit, got %d", len(retrievedLimited.Bids))
	}

	if len(retrievedLimited.Asks) != 2 {
		t.Errorf("Expected 2 asks after depth limit, got %d", len(retrievedLimited.Asks))
	}
}

func TestGraphTriangleIntegration(t *testing.T) {
	idx := graph.NewIndex()

	markets := testutils.CreateTestMarkets()

	for _, market := range markets {
		idx.AddMarket(market)
	}

	if len(idx.Triangles) == 0 {
		t.Error("Should have found triangles")
	}

	for i, triangle := range idx.Triangles {
		if len(triangle.MarketIds) != 3 {
			t.Errorf("Triangle %d should have 3 market IDs, got %d", i, len(triangle.MarketIds))
		}

		for _, marketID := range triangle.MarketIds {
			if marketID < 0 || marketID >= len(idx.Markets) {
				t.Errorf("Triangle %d has invalid market ID %d", i, marketID)
			}
		}

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

func TestProfitSimulatorIntegration(t *testing.T) {
	sim := profit.NewTOBSimulator(0.001, 5.0)

	markets := testutils.CreateTestMarkets()
	triangle := testutils.CreateTestTriangle(markets, [3]int{0, 1, 2})

	graphTriangle := types.Triangle{
		MarketIds: [3]int{1, 2, 0},
		Dirs:      [3]int8{1, -1, -1},
		QuoteCcy:  "USDT",
	}

	profitablePrices := testutils.CreateProfitableArbitragePrices()

	tobBySymbol := func(symbol string) (types.TopOfBook, bool) {
		tob, exists := profitablePrices[symbol]
		return tob, exists
	}

	feeBySymbol := func(symbol string) (types.Fee, bool) {
		return types.Fee{TakerBp: 0.1, MakerBp: 0.05}, true
	}

	plan, found := sim.EvaluateTOB(graphTriangle, markets, tobBySymbol, feeBySymbol, 1000.0)

	t.Logf("Profit simulation found: %v, profit: %f", found, plan.ExpectedProfitQuote)

	if !found {
		t.Error("Expected to find profitable arbitrage with test data")
	}

	if found && plan.ExpectedProfitQuote <= 0 {
		t.Errorf("Found arbitrage but profit is not positive: %f", plan.ExpectedProfitQuote)
	}

	if found {
		t.Logf("Successfully found arbitrage with profit: %.2f %s", plan.ExpectedProfitQuote, plan.QuoteCurrency)
	}

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

func TestConcurrencyIntegration(t *testing.T) {
	idx := graph.NewIndex()
	books := bookstore.NewTopOfBookStore()
	orderBooks := bookstore.NewOrderBookStore()
	reg := registry.NewMarketRegistry()
	sim := profit.NewTOBSimulator(0.001, 5.0)
	pub := testutils.NewMockPublisher()

	detector := detector.NewDetector(idx, books, reg, sim, pub)

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

	done := make(chan bool, 1)

	go func() {
		for i := 0; i < 100; i++ {
			detector.OnMarketChange("binance", "BTCUSDT", 1000.0)
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 50; i++ {
			bids := []types.Level{{Price: 100.0 + float64(i), Qty: 10.0}}
			asks := []types.Level{{Price: 101.0 + float64(i), Qty: 10.0}}
			orderBooks.Upsert("BTCUSDT", bids, asks, uint64(i), 1640995200000000000, 0)
		}
		done <- true
	}()

	<-done
	<-done

	_, exists := books.Get("BTCUSDT")
	if !exists {
		t.Error("Data should still exist after concurrent operations")
	}

	t.Log("Concurrent operations completed successfully")
}

func TestMemoryManagementIntegration(t *testing.T) {

	idx := graph.NewIndex()
	books := bookstore.NewTopOfBookStore()
	reg := registry.NewMarketRegistry()

	addedMarkets := 0
	for i := 0; i < 1000; i++ {
		market := types.Market{
			Exchange: "test",
			Symbol:   "TEST" + string(rune(65+i%26)) + string(rune(48+i%10)),
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

	if len(idx.Markets) == 0 {
		t.Error("Expected at least some markets to be added")
	}

	t.Logf("Successfully added %d markets to index", len(idx.Markets))

	reg.Snapshot()
	books.Snapshot()

	t.Log("Memory management test completed")
}

func TestIRTArbitrageWithOrderBookDepth(t *testing.T) {
	markets := []types.Market{
		{
			Exchange: "nobitex", Symbol: "BTCUSDT", Base: "BTC", Quote: "USDT",
			Multiplier: 100000000, MinQty: 0.0001, StepSize: 0.0001, MinNotional: 10.0, PriceTick: 0.01,
		},
		{
			Exchange: "nobitex", Symbol: "BTCIRT", Base: "BTC", Quote: "IRT",
			Multiplier: 100000000, MinQty: 0.0001, StepSize: 0.0001, MinNotional: 1000000.0, PriceTick: 1000.0,
		},
		{
			Exchange: "nobitex", Symbol: "USDTIRT", Base: "USDT", Quote: "IRT",
			Multiplier: 100000000, MinQty: 0.1, StepSize: 0.1, MinNotional: 1000000.0, PriceTick: 1000.0,
		},
	}

	triangle := types.Triangle{
		MarketIds: [3]int{0, 1, 2},
		Dirs:      [3]int8{1, -1, -1},
		QuoteCcy:  "USDT",
	}

	cfg, err := config.Load("config.yaml")
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	orderBooks := map[string]types.OrderBook{
		"BTCUSDT": {
			Bids: []types.Level{
				{Price: 45000.0, Qty: 2.5}, {Price: 44950.0, Qty: 3.1}, {Price: 44900.0, Qty: 4.2},
				{Price: 44850.0, Qty: 2.8}, {Price: 44800.0, Qty: 5.5}, {Price: 44750.0, Qty: 3.9},
				{Price: 44700.0, Qty: 6.1}, {Price: 44650.0, Qty: 4.7}, {Price: 44600.0, Qty: 7.3},
				{Price: 44550.0, Qty: 5.2}, {Price: 44500.0, Qty: 8.4}, {Price: 44450.0, Qty: 6.8},
				{Price: 44400.0, Qty: 9.1}, {Price: 44350.0, Qty: 7.5}, {Price: 44300.0, Qty: 10.2},
				{Price: 44250.0, Qty: 8.9}, {Price: 44200.0, Qty: 11.7}, {Price: 44150.0, Qty: 9.6},
				{Price: 44100.0, Qty: 12.8}, {Price: 44050.0, Qty: 10.3},
			},
			Asks: []types.Level{
				{Price: 45100.0, Qty: 2.1}, {Price: 45150.0, Qty: 2.8}, {Price: 45200.0, Qty: 3.7},
				{Price: 45250.0, Qty: 2.4}, {Price: 45300.0, Qty: 4.9}, {Price: 45350.0, Qty: 3.2},
				{Price: 45400.0, Qty: 5.6}, {Price: 45450.0, Qty: 3.8}, {Price: 45500.0, Qty: 6.7},
				{Price: 45550.0, Qty: 4.3}, {Price: 45600.0, Qty: 7.1}, {Price: 45650.0, Qty: 4.9},
				{Price: 45700.0, Qty: 8.2}, {Price: 45750.0, Qty: 5.4}, {Price: 45800.0, Qty: 9.3},
				{Price: 45850.0, Qty: 6.1}, {Price: 45900.0, Qty: 10.5}, {Price: 45950.0, Qty: 6.8},
				{Price: 46000.0, Qty: 11.9}, {Price: 46050.0, Qty: 7.2},
			},
		},
		"BTCIRT": {
			Bids: []types.Level{
				{Price: 2100000000.0, Qty: 0.5}, {Price: 2090000000.0, Qty: 0.7}, {Price: 2080000000.0, Qty: 0.9},
				{Price: 2070000000.0, Qty: 0.6}, {Price: 2060000000.0, Qty: 1.1}, {Price: 2050000000.0, Qty: 0.8},
				{Price: 2040000000.0, Qty: 1.2}, {Price: 2030000000.0, Qty: 0.9}, {Price: 2020000000.0, Qty: 1.4},
				{Price: 2010000000.0, Qty: 1.0}, {Price: 2000000000.0, Qty: 1.6}, {Price: 1990000000.0, Qty: 1.1},
				{Price: 1980000000.0, Qty: 1.8}, {Price: 1970000000.0, Qty: 1.3}, {Price: 1960000000.0, Qty: 2.0},
				{Price: 1950000000.0, Qty: 1.4}, {Price: 1940000000.0, Qty: 2.2}, {Price: 1930000000.0, Qty: 1.6},
				{Price: 1920000000.0, Qty: 2.4}, {Price: 1910000000.0, Qty: 1.7},
			},
			Asks: []types.Level{
				{Price: 2120000000.0, Qty: 0.4}, {Price: 2130000000.0, Qty: 0.6}, {Price: 2140000000.0, Qty: 0.8},
				{Price: 2150000000.0, Qty: 0.5}, {Price: 2160000000.0, Qty: 0.9}, {Price: 2170000000.0, Qty: 0.7},
				{Price: 2180000000.0, Qty: 1.0}, {Price: 2190000000.0, Qty: 0.8}, {Price: 2200000000.0, Qty: 1.2},
				{Price: 2210000000.0, Qty: 0.9}, {Price: 2220000000.0, Qty: 1.3}, {Price: 2230000000.0, Qty: 1.0},
				{Price: 2240000000.0, Qty: 1.5}, {Price: 2250000000.0, Qty: 1.1}, {Price: 2260000000.0, Qty: 1.7},
				{Price: 2270000000.0, Qty: 1.2}, {Price: 2280000000.0, Qty: 1.9}, {Price: 2290000000.0, Qty: 1.4},
				{Price: 2300000000.0, Qty: 2.1}, {Price: 2310000000.0, Qty: 1.5},
			},
		},
		"USDTIRT": {
			Bids: []types.Level{
				{Price: 420000.0, Qty: 100.0}, {Price: 419000.0, Qty: 120.0}, {Price: 418000.0, Qty: 140.0},
				{Price: 417000.0, Qty: 110.0}, {Price: 416000.0, Qty: 160.0}, {Price: 415000.0, Qty: 130.0},
				{Price: 414000.0, Qty: 180.0}, {Price: 413000.0, Qty: 150.0}, {Price: 412000.0, Qty: 200.0},
				{Price: 411000.0, Qty: 170.0}, {Price: 410000.0, Qty: 220.0}, {Price: 409000.0, Qty: 190.0},
				{Price: 408000.0, Qty: 240.0}, {Price: 407000.0, Qty: 210.0}, {Price: 406000.0, Qty: 260.0},
				{Price: 405000.0, Qty: 230.0}, {Price: 404000.0, Qty: 280.0}, {Price: 403000.0, Qty: 250.0},
				{Price: 402000.0, Qty: 300.0}, {Price: 401000.0, Qty: 270.0},
			},
			Asks: []types.Level{
				{Price: 421000.0, Qty: 90.0}, {Price: 422000.0, Qty: 110.0}, {Price: 423000.0, Qty: 130.0},
				{Price: 424000.0, Qty: 100.0}, {Price: 425000.0, Qty: 150.0}, {Price: 426000.0, Qty: 120.0},
				{Price: 427000.0, Qty: 170.0}, {Price: 428000.0, Qty: 140.0}, {Price: 429000.0, Qty: 190.0},
				{Price: 430000.0, Qty: 160.0}, {Price: 431000.0, Qty: 210.0}, {Price: 432000.0, Qty: 180.0},
				{Price: 433000.0, Qty: 230.0}, {Price: 434000.0, Qty: 200.0}, {Price: 435000.0, Qty: 250.0},
				{Price: 436000.0, Qty: 220.0}, {Price: 437000.0, Qty: 270.0}, {Price: 438000.0, Qty: 240.0},
				{Price: 439000.0, Qty: 290.0}, {Price: 440000.0, Qty: 260.0},
			},
		},
	}

	simulator := profit.NewTOBSimulator(0.001, 5.0)

	for _, maxDepth := range []int{1, 5, 10, 20} {
		obBySymbol := func(symbol string) (types.OrderBook, bool) {
			ob, exists := orderBooks[symbol]
			return ob, exists
		}

		feeBySymbol := func(symbol string) (types.Fee, bool) {
			for _, market := range markets {
				if market.Symbol == symbol {
					fee := cfg.GetFee(market.Exchange, market.Quote)
					return fee, true
				}
			}
			return cfg.GetFee("nobitex", "IRT"), true
		}

		plan, found := simulator.EvaluateOrderBook(triangle, markets, obBySymbol, feeBySymbol, 1000.0, maxDepth)

		if found {
			t.Logf("Depth %d: Found arbitrage profit: %.2f USDT", maxDepth, plan.ExpectedProfitQuote)
			for i, leg := range plan.Legs {
				t.Logf("  Leg %d: %s %s %.6f @ %.2f", i+1, leg.Side, leg.Market, leg.Qty, leg.LimitPrice)
			}
		} else {
			t.Logf("Depth %d: No arbitrage found", maxDepth)
		}
	}
}

func BenchmarkFullSystem(b *testing.B) {
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
