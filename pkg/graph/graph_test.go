package graph

import (
	"strings"
	"sync"
	"testing"

	"github.com/armagg/circular-arbitrage-finder/pkg/types"
)

func TestNewIndex(t *testing.T) {
	idx := NewIndex()

	if idx == nil {
		t.Error("NewIndex should return a non-nil index")
	}

	if idx.Markets == nil {
		t.Error("Markets slice should be initialized")
	}

	if len(idx.Markets) != 0 {
		t.Errorf("New index should have empty markets slice, got %d", len(idx.Markets))
	}

	if idx.MarketIndexBySymbol == nil {
		t.Error("MarketIndexBySymbol map should be initialized")
	}

	if idx.Triangles == nil {
		t.Error("Triangles slice should be initialized")
	}

	if idx.TrianglesByMarket == nil {
		t.Error("TrianglesByMarket map should be initialized")
	}
}

func TestMakeTriangle(t *testing.T) {
	triangle := makeTriangle(0, 1, 2, "USDT")

	expected := types.Triangle{
		MarketIds: [3]int{0, 1, 2},
		Dirs:      [3]int8{1, -1, -1},
		QuoteCcy:  "USDT",
	}

	if triangle != expected {
		t.Errorf("makeTriangle returned %+v, expected %+v", triangle, expected)
	}
}

func TestIndexAddMarket(t *testing.T) {
	idx := NewIndex()

	// Add first market
	market1 := types.Market{
		Exchange: "binance",
		Symbol:   "BTCUSDT",
		Base:     "BTC",
		Quote:    "USDT",
	}

	newTriangles, isNew := idx.AddMarket(market1)

	if !isNew {
		t.Error("First market should be new")
	}

	if len(newTriangles) != 0 {
		t.Errorf("First market should not create triangles, got %d", len(newTriangles))
	}

	if len(idx.Markets) != 1 {
		t.Errorf("Expected 1 market, got %d", len(idx.Markets))
	}

	if idx.Markets[0] != market1 {
		t.Error("Market should be stored correctly")
	}

	// Add second market
	market2 := types.Market{
		Exchange: "binance",
		Symbol:   "ETHUSDT",
		Base:     "ETH",
		Quote:    "USDT",
	}

	newTriangles2, isNew2 := idx.AddMarket(market2)

	if !isNew2 {
		t.Error("Second market should be new")
	}

	if len(newTriangles2) != 0 {
		t.Errorf("Two markets should not create triangles yet, got %d", len(newTriangles2))
	}

	if len(idx.Markets) != 2 {
		t.Errorf("Expected 2 markets, got %d", len(idx.Markets))
	}

	// Add third market that creates a triangle
	market3 := types.Market{
		Exchange: "binance",
		Symbol:   "ETHBTC",
		Base:     "ETH",
		Quote:    "BTC",
	}

	newTriangles3, isNew3 := idx.AddMarket(market3)

	if !isNew3 {
		t.Error("Third market should be new")
	}

	if len(newTriangles3) == 0 {
		t.Error("Third market should create triangles")
	}

	if len(idx.Markets) != 3 {
		t.Errorf("Expected 3 markets, got %d", len(idx.Markets))
	}

	if len(idx.Triangles) == 0 {
		t.Error("Triangles should be created")
	}
}

func TestIndexAddMarketDuplicate(t *testing.T) {
	idx := NewIndex()

	market := types.Market{
		Exchange: "binance",
		Symbol:   "BTCUSDT",
		Base:     "BTC",
		Quote:    "USDT",
	}

	// Add market first time
	_, isNew1 := idx.AddMarket(market)
	if !isNew1 {
		t.Error("First addition should be new")
	}

	// Add same market again
	_, isNew2 := idx.AddMarket(market)
	if isNew2 {
		t.Error("Second addition should not be new")
	}

	if len(idx.Markets) != 1 {
		t.Errorf("Expected 1 market after duplicate, got %d", len(idx.Markets))
	}
}

func TestIndexTriangleCreation(t *testing.T) {
	idx := NewIndex()

	// Create a complete triangle: BTC/USDT -> ETH/BTC -> ETH/USDT
	markets := []types.Market{
		{Exchange: "binance", Symbol: "BTCUSDT", Base: "BTC", Quote: "USDT"},
		{Exchange: "binance", Symbol: "ETHBTC", Base: "ETH", Quote: "BTC"},
		{Exchange: "binance", Symbol: "ETHUSDT", Base: "ETH", Quote: "USDT"},
	}

	for i, market := range markets {
		triangles, isNew := idx.AddMarket(market)
		if !isNew {
			t.Errorf("Market %d should be new", i)
		}

		// Third market should create triangles
		if i == 2 {
			if len(triangles) == 0 {
				t.Error("Third market should create triangles")
			}

			// Verify triangle structure
			for _, triangle := range triangles {
				if len(triangle.MarketIds) != 3 {
					t.Errorf("Triangle should have 3 market IDs, got %d", len(triangle.MarketIds))
				}

				if triangle.QuoteCcy != "USDT" {
					t.Errorf("Triangle should have USDT as quote currency, got %s", triangle.QuoteCcy)
				}

				// Verify market IDs are valid
				for _, marketID := range triangle.MarketIds {
					if marketID < 0 || marketID >= len(idx.Markets) {
						t.Errorf("Invalid market ID %d", marketID)
					}
				}
			}
		}
	}

	if len(idx.Triangles) == 0 {
		t.Error("Triangles should be created")
	}

	// Verify triangles are indexed by market
	for marketID, triangles := range idx.TrianglesByMarket {
		if len(triangles) == 0 {
			t.Errorf("Market %d should have triangles", marketID)
		}

		for _, triangleID := range triangles {
			if triangleID < 0 || triangleID >= len(idx.Triangles) {
				t.Errorf("Invalid triangle ID %d for market %d", triangleID, marketID)
			}
		}
	}
}

func TestIndexComplexTriangleFinding(t *testing.T) {
	idx := NewIndex()

	// Create a more complex market graph
	markets := []types.Market{
		{Exchange: "binance", Symbol: "BTCUSDT", Base: "BTC", Quote: "USDT"},
		{Exchange: "binance", Symbol: "ETHUSDT", Base: "ETH", Quote: "USDT"},
		{Exchange: "binance", Symbol: "ADAUSDT", Base: "ADA", Quote: "USDT"},
		{Exchange: "binance", Symbol: "ETHBTC", Base: "ETH", Quote: "BTC"},
		{Exchange: "binance", Symbol: "ADABTC", Base: "ADA", Quote: "BTC"},
		{Exchange: "binance", Symbol: "ADAETH", Base: "ADA", Quote: "ETH"},
	}

	for _, market := range markets {
		idx.AddMarket(market)
	}

	// Should find multiple triangles
	if len(idx.Triangles) == 0 {
		t.Error("Should find triangles in complex graph")
	}

	t.Logf("Found %d triangles", len(idx.Triangles))
	for i, triangle := range idx.Triangles {
		t.Logf("Triangle %d: Markets [%d,%d,%d], Quote: %s",
			i, triangle.MarketIds[0], triangle.MarketIds[1], triangle.MarketIds[2], triangle.QuoteCcy)
	}
}

func TestIndexConcurrency(t *testing.T) {
	idx := NewIndex()

	// Test concurrent reads (AddMarket already has mutex protection)
	var wg sync.WaitGroup
	numGoroutines := 5
	numOperations := 100

	// First add some markets
	for i := 0; i < 10; i++ {
		market := types.Market{
			Exchange: "binance",
			Symbol:   "TEST" + string(rune(i+65)),
			Base:     "BASE" + string(rune(i+65)),
			Quote:    "USDT",
		}
		idx.AddMarket(market)
	}

	// Test concurrent reads of the index
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				// Test concurrent reads
				_ = len(idx.Markets)
				_ = len(idx.Triangles)
			}
		}()
	}

	wg.Wait()

	// Verify data integrity
	if len(idx.Markets) != 10 {
		t.Errorf("Expected 10 markets, got %d", len(idx.Markets))
	}
}

func TestIndexMarketIndexing(t *testing.T) {
	idx := NewIndex()

	market := types.Market{
		Exchange: "BINANCE",
		Symbol:   "btcusdt",
		Base:     "BTC",
		Quote:    "USDT",
	}

	idx.AddMarket(market)

	// Test case-insensitive lookup (but index stores uppercase)
	key1 := "BINANCE:BTCUSDT"

	if _, exists := idx.MarketIndexBySymbol[key1]; !exists {
		t.Errorf("Should find market with key %s", key1)
	}
}

func TestIndexFindNewTrianglesEdgeCases(t *testing.T) {
	idx := NewIndex()

	// Test with no existing markets
	market := types.Market{
		Exchange: "binance",
		Symbol:   "BTCUSDT",
		Base:     "BTC",
		Quote:    "USDT",
	}

	triangles := idx.findNewTriangles(market, 0)

	if len(triangles) != 0 {
		t.Errorf("Should find no triangles with single market, got %d", len(triangles))
	}

	// Add the market first
	idx.Markets = append(idx.Markets, market)
	idx.MarketIndexBySymbol["binance:btcusdt"] = 0
	idx.marketsByExchange["binance"] = make(map[string]int)
	idx.marketsByExchange["binance"]["btc/usdt"] = 0

	// Test with one existing market
	newMarket := types.Market{
		Exchange: "binance",
		Symbol:   "ETHUSDT",
		Base:     "ETH",
		Quote:    "USDT",
	}

	triangles2 := idx.findNewTriangles(newMarket, 1)

	if len(triangles2) != 0 {
		t.Errorf("Should find no triangles with two markets, got %d", len(triangles2))
	}
}

func TestIndexMultipleExchanges(t *testing.T) {
	idx := NewIndex()

	// Add markets from different exchanges
	markets := []types.Market{
		{Exchange: "binance", Symbol: "BTCUSDT", Base: "BTC", Quote: "USDT"},
		{Exchange: "coinbase", Symbol: "BTCUSDT", Base: "BTC", Quote: "USDT"},
		{Exchange: "binance", Symbol: "ETHUSDT", Base: "ETH", Quote: "USDT"},
		{Exchange: "coinbase", Symbol: "ETHUSDT", Base: "ETH", Quote: "USDT"},
	}

	for _, market := range markets {
		idx.AddMarket(market)
	}

	// Should have separate indexing for each exchange
	if len(idx.marketsByExchange) != 2 {
		t.Errorf("Expected 2 exchanges, got %d", len(idx.marketsByExchange))
	}

	if len(idx.marketsByExchange["binance"]) != 2 {
		t.Errorf("Expected 2 binance markets, got %d", len(idx.marketsByExchange["binance"]))
	}

	if len(idx.marketsByExchange["coinbase"]) != 2 {
		t.Errorf("Expected 2 coinbase markets, got %d", len(idx.marketsByExchange["coinbase"]))
	}
}

func TestIndexSnapshotConsistency(t *testing.T) {
	idx := NewIndex()

	// Add some markets and create triangles
	markets := []types.Market{
		{Exchange: "binance", Symbol: "BTCUSDT", Base: "BTC", Quote: "USDT"},
		{Exchange: "binance", Symbol: "ETHUSDT", Base: "ETH", Quote: "USDT"},
		{Exchange: "binance", Symbol: "ETHBTC", Base: "ETH", Quote: "BTC"},
	}

	for _, market := range markets {
		idx.AddMarket(market)
	}

	// Verify data consistency
	for i, market := range idx.Markets {
		if market.Symbol == "" {
			t.Errorf("Market %d has empty symbol", i)
		}

		// Verify reverse indexing (index uses uppercase)
		key := strings.ToUpper(market.Exchange) + ":" + strings.ToUpper(market.Symbol)
		if marketID, exists := idx.MarketIndexBySymbol[key]; !exists {
			t.Errorf("Market %s not found in index", key)
		} else if marketID != i {
			t.Errorf("Market %s has wrong ID: expected %d, got %d", key, i, marketID)
		}
	}

	// Verify triangle consistency
	for triangleID, triangle := range idx.Triangles {
		for _, marketID := range triangle.MarketIds {
			if marketID < 0 || marketID >= len(idx.Markets) {
				t.Errorf("Triangle %d has invalid market ID %d", triangleID, marketID)
			}
		}

		// Verify reverse indexing
		for _, marketID := range triangle.MarketIds {
			found := false
			for _, triID := range idx.TrianglesByMarket[marketID] {
				if triID == triangleID {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Triangle %d not found in reverse index for market %d", triangleID, marketID)
			}
		}
	}
}

// Benchmark tests
func BenchmarkIndexAddMarket(b *testing.B) {
	idx := NewIndex()

	market := types.Market{
		Exchange: "binance",
		Symbol:   "BTCUSDT",
		Base:     "BTC",
		Quote:    "USDT",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		idx.AddMarket(market)
	}
}

func BenchmarkIndexFindNewTriangles(b *testing.B) {
	idx := NewIndex()

	// Set up some existing markets
	for i := 0; i < 10; i++ {
		market := types.Market{
			Exchange: "binance",
			Symbol:   "TEST" + string(rune(i+65)) + "USDT",
			Base:     "TEST" + string(rune(i+65)),
			Quote:    "USDT",
		}
		idx.Markets = append(idx.Markets, market)
		idx.MarketIndexBySymbol[market.Exchange+":"+market.Symbol] = i
		if idx.marketsByExchange[market.Exchange] == nil {
			idx.marketsByExchange[market.Exchange] = make(map[string]int)
		}
		idx.marketsByExchange[market.Exchange][market.Base+"/"+market.Quote] = i
	}

	newMarket := types.Market{
		Exchange: "binance",
		Symbol:   "TESTKBTC",
		Base:     "TESTK",
		Quote:    "BTC",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		idx.findNewTriangles(newMarket, len(idx.Markets))
	}
}
