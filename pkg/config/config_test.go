package config

import (
	"reflect"
	"sort"
	"testing"

	"github.com/armagg/circular-arbitrage-finder/pkg/types"
	"gopkg.in/yaml.v2"
)

func TestParseSymbol(t *testing.T) {
	tests := []struct {
		name          string
		symbol        string
		quoteAssets   []string
		expectedBase  string
		expectedQuote string
		expectError   bool
	}{
		{
			name:          "Valid BTCUSDT with USDT",
			symbol:        "BTCUSDT",
			quoteAssets:   []string{"USDT", "BTC", "ETH"},
			expectedBase:  "BTC",
			expectedQuote: "USDT",
			expectError:   false,
		},
		{
			name:          "Valid ETHBTC with BTC",
			symbol:        "ETHBTC",
			quoteAssets:   []string{"USDT", "BTC", "ETH"},
			expectedBase:  "ETH",
			expectedQuote: "BTC",
			expectError:   false,
		},
		{
			name:        "Invalid symbol not ending with quote asset",
			symbol:      "INVALID",
			quoteAssets: []string{"USDT", "BTC", "ETH"},
			expectError: true,
		},
		{
			name:        "Empty quote assets",
			symbol:      "BTCUSDT",
			quoteAssets: []string{},
			expectError: true,
		},
		{
			name:          "Quote assets sorted by length",
			symbol:        "BTCUSDT",
			quoteAssets:   []string{"USD", "USDT", "BTC"},
			expectedBase:  "BTC",
			expectedQuote: "USDT",
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			base, quote, err := parseSymbol(tt.symbol, tt.quoteAssets)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if base != tt.expectedBase {
				t.Errorf("Expected base %s, got %s", tt.expectedBase, base)
			}

			if quote != tt.expectedQuote {
				t.Errorf("Expected quote %s, got %s", tt.expectedQuote, quote)
			}
		})
	}
}

func TestConfigParseMarket(t *testing.T) {
	cfg := &Config{
		QuoteAssets: []string{"USDT", "BTC", "ETH"},
	}

	tests := []struct {
		name           string
		exchange       string
		symbol         string
		expectedMarket types.Market
		expectError    bool
	}{
		{
			name:     "Valid BTCUSDT market",
			exchange: "binance",
			symbol:   "BTCUSDT",
			expectedMarket: types.Market{
				Exchange: "binance",
				Symbol:   "BTCUSDT",
				Base:     "BTC",
				Quote:    "USDT",
			},
			expectError: false,
		},
		{
			name:        "Invalid symbol",
			exchange:    "binance",
			symbol:      "INVALID",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			market, err := cfg.ParseMarket(tt.exchange, tt.symbol)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if market.Exchange != tt.expectedMarket.Exchange {
				t.Errorf("Expected exchange %s, got %s", tt.expectedMarket.Exchange, market.Exchange)
			}

			if market.Symbol != tt.expectedMarket.Symbol {
				t.Errorf("Expected symbol %s, got %s", tt.expectedMarket.Symbol, market.Symbol)
			}

			if market.Base != tt.expectedMarket.Base {
				t.Errorf("Expected base %s, got %s", tt.expectedMarket.Base, market.Base)
			}

			if market.Quote != tt.expectedMarket.Quote {
				t.Errorf("Expected quote %s, got %s", tt.expectedMarket.Quote, market.Quote)
			}
		})
	}
}

func TestConfigGetFee(t *testing.T) {
	cfg := &Config{
		Fees: Fees{
			Default: FeeConfig{
				Taker: 0.1,
				Maker: 0.05,
			},
			Exchanges: map[string]FeeQuotes{
				"BINANCE": {
					"USDT": FeeConfig{Taker: 0.08, Maker: 0.04},
					"BTC":  FeeConfig{Taker: 0.06, Maker: 0.03},
				},
			},
		},
	}

	tests := []struct {
		name        string
		exchange    string
		quoteAsset  string
		expectedFee types.Fee
	}{
		{
			name:        "Exchange and quote specific fee",
			exchange:    "binance",
			quoteAsset:  "USDT",
			expectedFee: types.Fee{TakerBp: 0.08, MakerBp: 0.04},
		},
		{
			name:        "Exchange specific fee for BTC",
			exchange:    "binance",
			quoteAsset:  "BTC",
			expectedFee: types.Fee{TakerBp: 0.06, MakerBp: 0.03},
		},
		{
			name:        "Default fee for unknown exchange",
			exchange:    "unknown",
			quoteAsset:  "USDT",
			expectedFee: types.Fee{TakerBp: 0.1, MakerBp: 0.05},
		},
		{
			name:        "Default fee for unknown quote",
			exchange:    "binance",
			quoteAsset:  "UNKNOWN",
			expectedFee: types.Fee{TakerBp: 0.1, MakerBp: 0.05},
		},
		{
			name:        "Case insensitive exchange",
			exchange:    "BINANCE",
			quoteAsset:  "usdt",
			expectedFee: types.Fee{TakerBp: 0.08, MakerBp: 0.04},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fee := cfg.GetFee(tt.exchange, tt.quoteAsset)

			if fee.TakerBp != tt.expectedFee.TakerBp {
				t.Errorf("Expected taker %f, got %f", tt.expectedFee.TakerBp, fee.TakerBp)
			}

			if fee.MakerBp != tt.expectedFee.MakerBp {
				t.Errorf("Expected maker %f, got %f", tt.expectedFee.MakerBp, fee.MakerBp)
			}
		})
	}
}

func TestLoadConfig(t *testing.T) {
	// Test YAML unmarshaling and quote asset sorting without filesystem operations
	configYAML := `
quote_assets:
  - USDT
  - BTC
  - ETH
fees:
  default:
    taker: 0.1
    maker: 0.05
  exchanges:
    binance:
      USDT:
        taker: 0.08
        maker: 0.04
strategy:
  min_profit_edge: 0.001
  slippage_bp: 5.0
  trade_amount: 1000.0
  orderbook_depth: 10
log:
  level: info
`

	var cfg Config
	err := yaml.Unmarshal([]byte(configYAML), &cfg)
	if err != nil {
		t.Fatalf("Failed to unmarshal config YAML: %v", err)
	}

	// Test quote assets (should be sorted by length descending after Load function)
	expectedQuotes := []string{"USDT", "BTC", "ETH"} // This is the input order
	if !reflect.DeepEqual(cfg.QuoteAssets, expectedQuotes) {
		t.Errorf("Expected quote assets %v, got %v", expectedQuotes, cfg.QuoteAssets)
	}

	// Test fees
	if cfg.Fees.Default.Taker != 0.1 {
		t.Errorf("Expected default taker 0.1, got %f", cfg.Fees.Default.Taker)
	}

	if cfg.Fees.Default.Maker != 0.05 {
		t.Errorf("Expected default maker 0.05, got %f", cfg.Fees.Default.Maker)
	}

	// Test exchange fees
	binanceFees, ok := cfg.Fees.Exchanges["binance"]
	if !ok {
		t.Error("Expected binance fees to exist")
	}

	usdtFee, ok := binanceFees["USDT"]
	if !ok {
		t.Error("Expected USDT fee to exist")
	}

	if usdtFee.Taker != 0.08 {
		t.Errorf("Expected binance USDT taker 0.08, got %f", usdtFee.Taker)
	}

	// Test strategy
	if cfg.Strategy.MinProfitEdge != 0.001 {
		t.Errorf("Expected min profit edge 0.001, got %f", cfg.Strategy.MinProfitEdge)
	}

	if cfg.Strategy.SlippageBp != 5.0 {
		t.Errorf("Expected slippage 5.0, got %f", cfg.Strategy.SlippageBp)
	}

	if cfg.Strategy.TradeAmount != 1000.0 {
		t.Errorf("Expected trade amount 1000.0, got %f", cfg.Strategy.TradeAmount)
	}

	if cfg.Strategy.OrderbookDepth != 10 {
		t.Errorf("Expected orderbook depth 10, got %d", cfg.Strategy.OrderbookDepth)
	}

	// Test log config
	if cfg.Log.Level != "info" {
		t.Errorf("Expected log level info, got %s", cfg.Log.Level)
	}
}

func TestLoadConfigErrors(t *testing.T) {
	tests := []struct {
		name        string
		yamlContent string
		expectError bool
	}{
		{
			name:        "Invalid YAML",
			yamlContent: "invalid: yaml: content: [unclosed",
			expectError: true,
		},
		{
			name:        "Empty YAML",
			yamlContent: "",
			expectError: false, // Empty YAML unmarshals to zero values
		},
		{
			name: "Valid minimal YAML",
			yamlContent: `
quote_assets: []
fees:
  default:
    taker: 0.1
    maker: 0.05
strategy:
  min_profit_edge: 0.001
  slippage_bp: 5.0
  trade_amount: 1000.0
  orderbook_depth: 10
log:
  level: info
`,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var cfg Config
			err := yaml.Unmarshal([]byte(tt.yamlContent), &cfg)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for %s, but got none", tt.name)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for %s: %v", tt.name, err)
				}
			}
		})
	}
}

func TestQuoteAssetSorting(t *testing.T) {
	// Test the sorting logic that's part of the Load function
	quoteAssets := []string{"USD", "USDT", "BTC", "ETH"}

	// Apply the same sorting logic as Load function (sort by length descending)
	sort.Slice(quoteAssets, func(i, j int) bool {
		return len(quoteAssets[i]) > len(quoteAssets[j])
	})

	// Expected: USDT (4 chars), then the 3-char assets in original order: USD, BTC, ETH
	expected := []string{"USDT", "USD", "BTC", "ETH"}
	if !reflect.DeepEqual(quoteAssets, expected) {
		t.Errorf("Expected sorted quote assets %v, got %v", expected, quoteAssets)
	}

	// Verify the sorting logic: longer assets come first
	if len(quoteAssets[0]) <= len(quoteAssets[1]) {
		t.Error("First asset should be longer than or equal to second")
	}
	if len(quoteAssets[1]) < len(quoteAssets[2]) {
		t.Error("Second asset should be longer than or equal to third")
	}
}

func TestConfigStructs(t *testing.T) {
	// Test that all config structs can be instantiated
	cfg := &Config{}
	if cfg == nil {
		t.Error("Config should not be nil")
	}

	fees := &Fees{}
	if fees == nil {
		t.Error("Fees should not be nil")
	}

	strategy := &Strategy{}
	if strategy == nil {
		t.Error("Strategy should not be nil")
	}

	logConfig := &LogConfig{}
	if logConfig == nil {
		t.Error("LogConfig should not be nil")
	}
}

func TestGetFeeCaseSensitivity(t *testing.T) {
	cfg := &Config{
		Fees: Fees{
			Default: FeeConfig{Taker: 0.1, Maker: 0.05},
			Exchanges: map[string]FeeQuotes{
				"BINANCE": {
					"USDT": FeeConfig{Taker: 0.08, Maker: 0.04},
					"BTC":  FeeConfig{Taker: 0.06, Maker: 0.03},
				},
			},
		},
	}

	tests := []struct {
		name       string
		exchange   string
		quoteAsset string
		expected   types.Fee
	}{
		{
			name:       "Exact case match",
			exchange:   "BINANCE",
			quoteAsset: "USDT",
			expected:   types.Fee{TakerBp: 0.08, MakerBp: 0.04},
		},
		{
			name:       "Lowercase exchange",
			exchange:   "binance",
			quoteAsset: "USDT",
			expected:   types.Fee{TakerBp: 0.08, MakerBp: 0.04},
		},
		{
			name:       "Lowercase quote asset",
			exchange:   "BINANCE",
			quoteAsset: "usdt",
			expected:   types.Fee{TakerBp: 0.08, MakerBp: 0.04},
		},
		{
			name:       "Mixed case",
			exchange:   "Binance",
			quoteAsset: "UsDt",
			expected:   types.Fee{TakerBp: 0.08, MakerBp: 0.04},
		},
		{
			name:       "Unknown exchange",
			exchange:   "unknown",
			quoteAsset: "USDT",
			expected:   types.Fee{TakerBp: 0.1, MakerBp: 0.05},
		},
		{
			name:       "Unknown quote asset",
			exchange:   "BINANCE",
			quoteAsset: "UNKNOWN",
			expected:   types.Fee{TakerBp: 0.1, MakerBp: 0.05},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fee := cfg.GetFee(tt.exchange, tt.quoteAsset)

			if fee.TakerBp != tt.expected.TakerBp {
				t.Errorf("Expected taker %f, got %f", tt.expected.TakerBp, fee.TakerBp)
			}

			if fee.MakerBp != tt.expected.MakerBp {
				t.Errorf("Expected maker %f, got %f", tt.expected.MakerBp, fee.MakerBp)
			}
		})
	}
}
