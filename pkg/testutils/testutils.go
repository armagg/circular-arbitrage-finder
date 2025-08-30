package testutils

import (
	"sync"

	"github.com/armagg/circular-arbitrage-finder/pkg/types"
)

// MockPublisher implements a thread-safe mock publisher for testing
type MockPublisher struct {
	mu      sync.RWMutex
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
	if m.publish != nil {
		return m.publish(plan)
	}
	return nil
}

func (m *MockPublisher) GetPublishedPlans() []types.Plan {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]types.Plan, len(m.plans))
	copy(result, m.plans)
	return result
}

func (m *MockPublisher) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.plans = m.plans[:0]
}

func (m *MockPublisher) SetPublishFunc(fn func(plan types.Plan) error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.publish = fn
}

// MockTOBProvider provides mock top-of-book data for testing
type MockTOBProvider struct {
	data map[string]types.TopOfBook
}

func NewMockTOBProvider() *MockTOBProvider {
	return &MockTOBProvider{
		data: make(map[string]types.TopOfBook),
	}
}

func (m *MockTOBProvider) Set(symbol string, tob types.TopOfBook) {
	m.data[symbol] = tob
}

func (m *MockTOBProvider) Get(symbol string) (types.TopOfBook, bool) {
	tob, exists := m.data[symbol]
	return tob, exists
}

func (m *MockTOBProvider) GetAll() map[string]types.TopOfBook {
	result := make(map[string]types.TopOfBook)
	for k, v := range m.data {
		result[k] = v
	}
	return result
}

func (m *MockTOBProvider) Clear() {
	m.data = make(map[string]types.TopOfBook)
}

// MockFeeProvider provides mock fee data for testing
type MockFeeProvider struct {
	data map[string]types.Fee
}

func NewMockFeeProvider() *MockFeeProvider {
	return &MockFeeProvider{
		data: make(map[string]types.Fee),
	}
}

func (m *MockFeeProvider) Set(symbol string, fee types.Fee) {
	m.data[symbol] = fee
}

func (m *MockFeeProvider) Get(symbol string) (types.Fee, bool) {
	fee, exists := m.data[symbol]
	return fee, exists
}

func (m *MockFeeProvider) Clear() {
	m.data = make(map[string]types.Fee)
}

// CreateTestMarkets creates a set of test markets for common testing scenarios
func CreateTestMarkets() []types.Market {
	return []types.Market{
		{
			Exchange:    "binance",
			Symbol:      "BTCUSDT",
			Base:        "BTC",
			Quote:       "USDT",
			Multiplier:  100000000,
			MinQty:      0.0001,
			StepSize:    0.0001,
			MinNotional: 10.0,
			PriceTick:   0.01,
		},
		{
			Exchange:    "binance",
			Symbol:      "ETHUSDT",
			Base:        "ETH",
			Quote:       "USDT",
			Multiplier:  100000000,
			MinQty:      0.001,
			StepSize:    0.001,
			MinNotional: 10.0,
			PriceTick:   0.01,
		},
		{
			Exchange:    "binance",
			Symbol:      "ETHBTC",
			Base:        "ETH",
			Quote:       "BTC",
			Multiplier:  100000000,
			MinQty:      0.001,
			StepSize:    0.001,
			MinNotional: 0.0001,
			PriceTick:   0.000001,
		},
		{
			Exchange:    "binance",
			Symbol:      "ADAUSDT",
			Base:        "ADA",
			Quote:       "USDT",
			Multiplier:  100000000,
			MinQty:      1.0,
			StepSize:    1.0,
			MinNotional: 10.0,
			PriceTick:   0.0001,
		},
	}
}

// CreateTestTriangle creates a test triangle from the given markets
func CreateTestTriangle(markets []types.Market, marketIds [3]int) types.Triangle {
	if len(markets) < 3 || len(marketIds) != 3 {
		panic("Need at least 3 markets and 3 market IDs")
	}

	quoteCcy := markets[marketIds[0]].Quote

	return types.Triangle{
		MarketIds: marketIds,
		Dirs:      [3]int8{1, -1, -1},
		QuoteCcy:  quoteCcy,
	}
}

// CreateTestOrderBook creates a test order book with realistic data
func CreateTestOrderBook(symbol string, bidPrice, askPrice float64) types.OrderBook {
	bids := []types.Level{
		{Price: bidPrice, Qty: 10.0},
		{Price: bidPrice - 1.0, Qty: 15.0},
		{Price: bidPrice - 2.0, Qty: 20.0},
		{Price: bidPrice - 3.0, Qty: 25.0},
		{Price: bidPrice - 4.0, Qty: 30.0},
	}

	asks := []types.Level{
		{Price: askPrice, Qty: 10.0},
		{Price: askPrice + 1.0, Qty: 15.0},
		{Price: askPrice + 2.0, Qty: 20.0},
		{Price: askPrice + 3.0, Qty: 25.0},
		{Price: askPrice + 4.0, Qty: 30.0},
	}

	return types.OrderBook{
		Bids: bids,
		Asks: asks,
		Seq:  12345,
		TsNs: 1640995200000000000,
	}
}

// CreateProfitableArbitragePrices creates a set of prices that should result in profitable arbitrage
func CreateProfitableArbitragePrices() map[string]types.TopOfBook {
	return map[string]types.TopOfBook{
		"BTCUSDT": {BidPx: 49800.0, AskPx: 49900.0, BidSz: 2.1, AskSz: 1.8},
		"ETHUSDT": {BidPx: 2980.0, AskPx: 2990.0, BidSz: 10.0, AskSz: 8.0},
		"ETHBTC":  {BidPx: 0.0598, AskPx: 0.0600, BidSz: 65.0, AskSz: 62.0},
	}
}

// CreateNoArbitragePrices creates a set of prices that should NOT result in profitable arbitrage
func CreateNoArbitragePrices() map[string]types.TopOfBook {
	return map[string]types.TopOfBook{
		"BTCUSDT": {BidPx: 50000.0, AskPx: 50000.0, BidSz: 2.1, AskSz: 1.8},
		"ETHUSDT": {BidPx: 3000.0, AskPx: 3000.0, BidSz: 10.0, AskSz: 8.0},
		"ETHBTC":  {BidPx: 0.06, AskPx: 0.06, BidSz: 65.0, AskSz: 62.0},
	}
}

// SetupTestTriangle creates a complete test setup with triangle markets
func SetupTestTriangle() (markets []types.Market, triangle types.Triangle) {
	markets = CreateTestMarkets()

	// Create triangle from first 3 markets
	triangle = CreateTestTriangle(markets, [3]int{0, 1, 2})

	return markets, triangle
}

// AssertPlanValid checks if a plan is valid
func AssertPlanValid(plan types.Plan) bool {
	if plan.ExpectedProfitQuote <= 0 {
		return false
	}

	if len(plan.Legs) != 3 {
		return false
	}

	if plan.Exchange == "" {
		return false
	}

	if plan.QuoteCurrency == "" {
		return false
	}

	for _, leg := range plan.Legs {
		if leg.Qty <= 0 || leg.LimitPrice <= 0 {
			return false
		}
		if leg.Market == "" {
			return false
		}
		if leg.Side != types.SideBuy && leg.Side != types.SideSell {
			return false
		}
	}

	return true
}

// CreateTestConfig creates a test configuration
func CreateTestConfig() map[string]interface{} {
	return map[string]interface{}{
		"quote_assets": []string{"USDT", "BTC", "ETH"},
		"fees": map[string]interface{}{
			"default": map[string]interface{}{
				"taker": 0.1,
				"maker": 0.05,
			},
			"exchanges": map[string]interface{}{
				"binance": map[string]interface{}{
					"USDT": map[string]interface{}{
						"taker": 0.08,
						"maker": 0.04,
					},
				},
			},
		},
		"strategy": map[string]interface{}{
			"min_profit_edge": 0.001,
			"slippage_bp":     5.0,
			"trade_amount":    1000.0,
			"orderbook_depth": 10,
		},
		"log": map[string]interface{}{
			"level": "info",
		},
	}
}

// TestDataBuilder helps build test data in a fluent way
type TestDataBuilder struct {
	markets []types.Market
	tobData map[string]types.TopOfBook
	feeData map[string]types.Fee
}

func NewTestDataBuilder() *TestDataBuilder {
	return &TestDataBuilder{
		markets: make([]types.Market, 0),
		tobData: make(map[string]types.TopOfBook),
		feeData: make(map[string]types.Fee),
	}
}

func (b *TestDataBuilder) WithMarket(market types.Market) *TestDataBuilder {
	b.markets = append(b.markets, market)
	return b
}

func (b *TestDataBuilder) WithTOB(symbol string, tob types.TopOfBook) *TestDataBuilder {
	b.tobData[symbol] = tob
	return b
}

func (b *TestDataBuilder) WithFee(symbol string, fee types.Fee) *TestDataBuilder {
	b.feeData[symbol] = fee
	return b
}

func (b *TestDataBuilder) Build() ([]types.Market, map[string]types.TopOfBook, map[string]types.Fee) {
	return b.markets, b.tobData, b.feeData
}
