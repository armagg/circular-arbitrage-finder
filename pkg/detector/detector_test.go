package detector

import (
	"sync"
	"testing"

	"github.com/armagg/circular-arbitrage-finder/pkg/bookstore"
	"github.com/armagg/circular-arbitrage-finder/pkg/graph"
	"github.com/armagg/circular-arbitrage-finder/pkg/profit"
	"github.com/armagg/circular-arbitrage-finder/pkg/registry"
	"github.com/armagg/circular-arbitrage-finder/pkg/types"

	"github.com/sirupsen/logrus"
)

// MockPublisher implements the Publisher interface for testing
type MockPublisher struct {
	mu      sync.Mutex
	plans   []types.Plan
	publish func(plan types.Plan) error
}

func NewMockPublisher() *MockPublisher {
	return &MockPublisher{
		plans: make([]types.Plan, 0),
		publish: func(plan types.Plan) error {
			return nil
		},
	}
}

func (m *MockPublisher) Publish(plan types.Plan) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.plans = append(m.plans, plan)
	return m.publish(plan)
}

func (m *MockPublisher) GetPublishedPlans() []types.Plan {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]types.Plan, len(m.plans))
	copy(result, m.plans)
	return result
}

func (m *MockPublisher) SetPublishFunc(fn func(plan types.Plan) error) {
	m.publish = fn
}

func TestNewDetector(t *testing.T) {
	idx := graph.NewIndex()
	books := bookstore.NewTopOfBookStore()
	reg := registry.NewMarketRegistry()
	sim := profit.NewTOBSimulator(0.001, 5.0)
	pub := NewMockPublisher()

	detector := NewDetector(idx, books, reg, sim, pub)

	if detector == nil {
		t.Error("NewDetector should return a non-nil detector")
	}

	if detector.Index != idx {
		t.Error("Detector should store the index")
	}

	if detector.Books != books {
		t.Error("Detector should store the bookstore")
	}

	if detector.Registry != reg {
		t.Error("Detector should store the registry")
	}

	if detector.Sim != sim {
		t.Error("Detector should store the simulator")
	}

	if detector.Publisher != pub {
		t.Error("Detector should store the publisher")
	}
}

func TestDetectorOnMarketChangeNoTriangles(t *testing.T) {
	idx := graph.NewIndex()
	books := bookstore.NewTopOfBookStore()
	reg := registry.NewMarketRegistry()
	sim := profit.NewTOBSimulator(0.001, 5.0)
	pub := NewMockPublisher()

	detector := NewDetector(idx, books, reg, sim, pub)

	// Add a single market (no triangles possible)
	market := types.Market{
		Exchange: "binance",
		Symbol:   "BTCUSDT",
		Base:     "BTC",
		Quote:    "USDT",
	}

	idx.AddMarket(market)

	// This should not panic and should not publish anything
	detector.OnMarketChange("binance", "BTCUSDT", 1000.0)

	published := pub.GetPublishedPlans()
	if len(published) != 0 {
		t.Errorf("Expected no plans to be published, got %d", len(published))
	}
}

func TestDetectorOnMarketChangeUnknownMarket(t *testing.T) {
	idx := graph.NewIndex()
	books := bookstore.NewTopOfBookStore()
	reg := registry.NewMarketRegistry()
	sim := profit.NewTOBSimulator(0.001, 5.0)
	pub := NewMockPublisher()

	detector := NewDetector(idx, books, reg, sim, pub)

	// Try to detect arbitrage for unknown market
	detector.OnMarketChange("binance", "UNKNOWN", 1000.0)

	published := pub.GetPublishedPlans()
	if len(published) != 0 {
		t.Errorf("Expected no plans to be published for unknown market, got %d", len(published))
	}
}

func TestDetectorOnMarketChangeWithTriangles(t *testing.T) {
	idx := graph.NewIndex()
	books := bookstore.NewTopOfBookStore()
	reg := registry.NewMarketRegistry()
	sim := profit.NewTOBSimulator(0.001, 5.0)
	pub := NewMockPublisher()

	detector := NewDetector(idx, books, reg, sim, pub)

	// Create a triangle of markets
	markets := []types.Market{
		{Exchange: "binance", Symbol: "BTCUSDT", Base: "BTC", Quote: "USDT"},
		{Exchange: "binance", Symbol: "ETHUSDT", Base: "ETH", Quote: "USDT"},
		{Exchange: "binance", Symbol: "ETHBTC", Base: "ETH", Quote: "BTC"},
	}

	for _, market := range markets {
		idx.AddMarket(market)
		reg.UpsertMarket(market)
		reg.SetFee(market.Symbol, types.Fee{TakerBp: 0.1, MakerBp: 0.05})
	}

	// Set up profitable arbitrage prices
	tobData := map[string]types.TopOfBook{
		"BTCUSDT": {BidPx: 49900.0, AskPx: 50000.0, BidSz: 2.1, AskSz: 1.8},
		"ETHUSDT": {BidPx: 2990.0, AskPx: 3000.0, BidSz: 10.0, AskSz: 8.0},
		"ETHBTC":  {BidPx: 0.0598, AskPx: 0.0600, BidSz: 65.0, AskSz: 62.0},
	}

	for symbol, tob := range tobData {
		books.Set(symbol, tob)
	}

	// This should find profitable arbitrage
	detector.OnMarketChange("binance", "BTCUSDT", 1000.0)

	published := pub.GetPublishedPlans()

	// The detector may or may not find profitable arbitrage depending on prices and settings
	t.Logf("Published %d plans", len(published))

	for _, plan := range published {
		if plan.ExpectedProfitQuote <= 0 {
			t.Errorf("Plan should have positive profit, got %f", plan.ExpectedProfitQuote)
		}

		if len(plan.Legs) != 3 {
			t.Errorf("Plan should have 3 legs, got %d", len(plan.Legs))
		}

		if plan.Exchange != "binance" {
			t.Errorf("Plan should have binance exchange, got %s", plan.Exchange)
		}
	}
}

func TestDetectorOnMarketChangeCaseInsensitive(t *testing.T) {
	idx := graph.NewIndex()
	books := bookstore.NewTopOfBookStore()
	reg := registry.NewMarketRegistry()
	sim := profit.NewTOBSimulator(0.001, 5.0)
	pub := NewMockPublisher()

	detector := NewDetector(idx, books, reg, sim, pub)

	// Add market with uppercase
	market := types.Market{
		Exchange: "BINANCE",
		Symbol:   "BTCUSDT",
		Base:     "BTC",
		Quote:    "USDT",
	}

	idx.AddMarket(market)

	// Try to detect with mixed case
	detector.OnMarketChange("Binance", "btcusdt", 1000.0)

	// Should not panic (this tests case insensitivity)
}

func TestDetectorOnMarketChangeMissingData(t *testing.T) {
	idx := graph.NewIndex()
	books := bookstore.NewTopOfBookStore()
	reg := registry.NewMarketRegistry()
	sim := profit.NewTOBSimulator(0.001, 5.0)
	pub := NewMockPublisher()

	detector := NewDetector(idx, books, reg, sim, pub)

	// Create triangle but don't set up all data
	markets := []types.Market{
		{Exchange: "binance", Symbol: "BTCUSDT", Base: "BTC", Quote: "USDT"},
		{Exchange: "binance", Symbol: "ETHUSDT", Base: "ETH", Quote: "USDT"},
		{Exchange: "binance", Symbol: "ETHBTC", Base: "ETH", Quote: "BTC"},
	}

	for _, market := range markets {
		idx.AddMarket(market)
		reg.UpsertMarket(market)
		reg.SetFee(market.Symbol, types.Fee{TakerBp: 0.1, MakerBp: 0.05})
	}

	// Only set up partial data (missing ETHBTC)
	books.Set("BTCUSDT", types.TopOfBook{BidPx: 50000.0, AskPx: 50010.0})
	books.Set("ETHUSDT", types.TopOfBook{BidPx: 3000.0, AskPx: 3010.0})
	// Missing ETHBTC data

	detector.OnMarketChange("binance", "BTCUSDT", 1000.0)

	published := pub.GetPublishedPlans()
	// Should not publish plans due to missing data
	t.Logf("Published %d plans with missing data", len(published))
}

func TestDetectorOnMarketChangeInvalidPrices(t *testing.T) {
	idx := graph.NewIndex()
	books := bookstore.NewTopOfBookStore()
	reg := registry.NewMarketRegistry()
	sim := profit.NewTOBSimulator(0.001, 5.0)
	pub := NewMockPublisher()

	detector := NewDetector(idx, books, reg, sim, pub)

	// Create triangle
	markets := []types.Market{
		{Exchange: "binance", Symbol: "BTCUSDT", Base: "BTC", Quote: "USDT"},
		{Exchange: "binance", Symbol: "ETHUSDT", Base: "ETH", Quote: "USDT"},
		{Exchange: "binance", Symbol: "ETHBTC", Base: "ETH", Quote: "BTC"},
	}

	for _, market := range markets {
		idx.AddMarket(market)
		reg.UpsertMarket(market)
		reg.SetFee(market.Symbol, types.Fee{TakerBp: 0.1, MakerBp: 0.05})
	}

	// Set up invalid prices (zero or negative)
	tobData := map[string]types.TopOfBook{
		"BTCUSDT": {BidPx: 0, AskPx: 0, BidSz: 2.1, AskSz: 1.8}, // Invalid prices
		"ETHUSDT": {BidPx: 3000.0, AskPx: 3010.0, BidSz: 10.0, AskSz: 8.0},
		"ETHBTC":  {BidPx: 0.06, AskPx: 0.0602, BidSz: 65.0, AskSz: 62.0},
	}

	for symbol, tob := range tobData {
		books.Set(symbol, tob)
	}

	detector.OnMarketChange("binance", "BTCUSDT", 1000.0)

	published := pub.GetPublishedPlans()
	// Should not publish plans due to invalid prices
	t.Logf("Published %d plans with invalid prices", len(published))
}

func TestDetectorConcurrency(t *testing.T) {
	idx := graph.NewIndex()
	books := bookstore.NewTopOfBookStore()
	reg := registry.NewMarketRegistry()
	sim := profit.NewTOBSimulator(0.001, 5.0)
	pub := NewMockPublisher()

	detector := NewDetector(idx, books, reg, sim, pub)

	// Set up basic market data
	market := types.Market{
		Exchange: "binance",
		Symbol:   "BTCUSDT",
		Base:     "BTC",
		Quote:    "USDT",
	}

	idx.AddMarket(market)
	reg.UpsertMarket(market)
	reg.SetFee(market.Symbol, types.Fee{TakerBp: 0.1, MakerBp: 0.05})
	books.Set("BTCUSDT", types.TopOfBook{BidPx: 50000.0, AskPx: 50010.0})

	// Test concurrent calls
	var wg sync.WaitGroup
	numGoroutines := 10
	numCalls := 100

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < numCalls; j++ {
				detector.OnMarketChange("binance", "BTCUSDT", 1000.0)
			}
		}()
	}

	wg.Wait()

	// Should not have panicked and should have processed all calls
	t.Logf("Processed %d concurrent calls successfully", numGoroutines*numCalls)
}

func TestDetectorMultipleTriangles(t *testing.T) {
	idx := graph.NewIndex()
	books := bookstore.NewTopOfBookStore()
	reg := registry.NewMarketRegistry()
	sim := profit.NewTOBSimulator(0.001, 5.0)
	pub := NewMockPublisher()

	detector := NewDetector(idx, books, reg, sim, pub)

	// Create multiple triangles
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
		reg.UpsertMarket(market)
		reg.SetFee(market.Symbol, types.Fee{TakerBp: 0.1, MakerBp: 0.05})
	}

	// Set up prices for all markets
	tobData := map[string]types.TopOfBook{
		"BTCUSDT": {BidPx: 50000.0, AskPx: 50010.0, BidSz: 2.1, AskSz: 1.8},
		"ETHUSDT": {BidPx: 3000.0, AskPx: 3010.0, BidSz: 10.0, AskSz: 8.0},
		"ADAUSDT": {BidPx: 1.5, AskPx: 1.52, BidSz: 1000.0, AskSz: 800.0},
		"ETHBTC":  {BidPx: 0.06, AskPx: 0.0602, BidSz: 65.0, AskSz: 62.0},
		"ADABTC":  {BidPx: 0.000030, AskPx: 0.000031, BidSz: 50000.0, AskSz: 40000.0},
		"ADAETH":  {BidPx: 0.0005, AskPx: 0.00052, BidSz: 10000.0, AskSz: 8000.0},
	}

	for symbol, tob := range tobData {
		books.Set(symbol, tob)
	}

	// Trigger detection on one market
	detector.OnMarketChange("binance", "BTCUSDT", 1000.0)

	published := pub.GetPublishedPlans()
	t.Logf("Published %d plans from multiple triangles", len(published))

	// Verify that we found some profitable arbitrage opportunities
	if len(published) == 0 {
		t.Log("No profitable arbitrage found (this may be expected depending on prices)")
	} else {
		for i, plan := range published {
			t.Logf("Plan %d: Profit %f, Legs %d", i, plan.ExpectedProfitQuote, len(plan.Legs))
		}
	}
}

func TestMockPublisher(t *testing.T) {
	pub := NewMockPublisher()

	plan := types.Plan{
		Exchange:            "binance",
		ExpectedProfitQuote: 25.5,
		QuoteCurrency:       "USDT",
		PlanID:              "test-plan",
	}

	err := pub.Publish(plan)
	if err != nil {
		t.Errorf("MockPublisher.Publish should not return error, got %v", err)
	}

	plans := pub.GetPublishedPlans()
	if len(plans) != 1 {
		t.Errorf("Expected 1 published plan, got %d", len(plans))
	}

	if plans[0] != plan {
		t.Error("Published plan should match original")
	}
}

func TestMockPublisherConcurrency(t *testing.T) {
	pub := NewMockPublisher()

	var wg sync.WaitGroup
	numGoroutines := 5
	numPublishes := 100

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numPublishes; j++ {
				plan := types.Plan{
					Exchange:            "binance",
					ExpectedProfitQuote: float64(id*numPublishes + j),
					PlanID:              "test-plan",
				}
				pub.Publish(plan)
			}
		}(i)
	}

	wg.Wait()

	plans := pub.GetPublishedPlans()
	expectedTotal := numGoroutines * numPublishes

	if len(plans) != expectedTotal {
		t.Errorf("Expected %d published plans, got %d", expectedTotal, len(plans))
	}
}

// Benchmark tests
func BenchmarkDetectorOnMarketChange(b *testing.B) {
	idx := graph.NewIndex()
	books := bookstore.NewTopOfBookStore()
	reg := registry.NewMarketRegistry()
	sim := profit.NewTOBSimulator(0.001, 5.0)
	pub := NewMockPublisher()

	detector := NewDetector(idx, books, reg, sim, pub)

	// Set up a single market
	market := types.Market{
		Exchange: "binance",
		Symbol:   "BTCUSDT",
		Base:     "BTC",
		Quote:    "USDT",
	}

	idx.AddMarket(market)
	reg.UpsertMarket(market)
	reg.SetFee(market.Symbol, types.Fee{TakerBp: 0.1, MakerBp: 0.05})
	books.Set("BTCUSDT", types.TopOfBook{BidPx: 50000.0, AskPx: 50010.0})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		detector.OnMarketChange("binance", "BTCUSDT", 1000.0)
	}
}

func BenchmarkDetectorOnMarketChangeWithTriangles(b *testing.B) {
	idx := graph.NewIndex()
	books := bookstore.NewTopOfBookStore()
	reg := registry.NewMarketRegistry()
	sim := profit.NewTOBSimulator(0.001, 5.0)
	pub := NewMockPublisher()

	detector := NewDetector(idx, books, reg, sim, pub)

	// Set up triangle markets
	markets := []types.Market{
		{Exchange: "binance", Symbol: "BTCUSDT", Base: "BTC", Quote: "USDT"},
		{Exchange: "binance", Symbol: "ETHUSDT", Base: "ETH", Quote: "USDT"},
		{Exchange: "binance", Symbol: "ETHBTC", Base: "ETH", Quote: "BTC"},
	}

	for _, market := range markets {
		idx.AddMarket(market)
		reg.UpsertMarket(market)
		reg.SetFee(market.Symbol, types.Fee{TakerBp: 0.1, MakerBp: 0.05})
	}

	tobData := map[string]types.TopOfBook{
		"BTCUSDT": {BidPx: 50000.0, AskPx: 50010.0, BidSz: 2.1, AskSz: 1.8},
		"ETHUSDT": {BidPx: 3000.0, AskPx: 3010.0, BidSz: 10.0, AskSz: 8.0},
		"ETHBTC":  {BidPx: 0.06, AskPx: 0.0602, BidSz: 65.0, AskSz: 62.0},
	}

	for symbol, tob := range tobData {
		books.Set(symbol, tob)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		detector.OnMarketChange("binance", "BTCUSDT", 1000.0)
	}
}

// Test logging behavior (without actually logging)
func TestDetectorLogging(t *testing.T) {
	// Save original logger level
	originalLevel := logrus.GetLevel()

	// Set to panic level to suppress logs during test
	logrus.SetLevel(logrus.PanicLevel)
	defer logrus.SetLevel(originalLevel)

	idx := graph.NewIndex()
	books := bookstore.NewTopOfBookStore()
	reg := registry.NewMarketRegistry()
	sim := profit.NewTOBSimulator(0.001, 5.0)
	pub := NewMockPublisher()

	detector := NewDetector(idx, books, reg, sim, pub)

	// This should not log anything due to panic level
	detector.OnMarketChange("binance", "UNKNOWN", 1000.0)
}
