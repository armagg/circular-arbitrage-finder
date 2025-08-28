package config

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/armagg/circular-arbitrage-finder/pkg/types"
	"gopkg.in/yaml.v2"
)

type Config struct {
	QuoteAssets []string  `yaml:"quote_assets"`
	Fees        Fees      `yaml:"fees"`
	Strategy    Strategy  `yaml:"strategy"`
	Log         LogConfig `yaml:"log"`
}

type Fees struct {
	Default   FeeConfig            `yaml:"default"`
	Exchanges map[string]FeeQuotes `yaml:"exchanges"`
}

type FeeQuotes map[string]FeeConfig

type FeeConfig struct {
	Taker float64 `yaml:"taker"`
	Maker float64 `yaml:"maker"`
}

type Strategy struct {
	MinProfitEdge float64            `yaml:"min_profit_edge"`
	SlippageBp    float64            `yaml:"slippage_bp"`
	TradeAmount   float64            `yaml:"trade_amount"`
	OrderbookDepth int               `yaml:"orderbook_depth"`
	TradeAmounts  map[string]float64 `yaml:"trade_amounts"`
}

type LogConfig struct {
	Level string `yaml:"level"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config yaml: %w", err)
	}
	sort.Slice(cfg.QuoteAssets, func(i, j int) bool { return len(cfg.QuoteAssets[i]) > len(cfg.QuoteAssets[j]) })
	return &cfg, nil
}

func (c *Config) ParseMarket(exchange, symbol string) (types.Market, error) {
	base, quote, err := parseSymbol(symbol, c.QuoteAssets)
	if err != nil {
		return types.Market{}, fmt.Errorf("failed to parse symbol %s: %w", symbol, err)
	}
	return types.Market{Exchange: exchange, Symbol: symbol, Base: base, Quote: quote}, nil
}

func parseSymbol(symbol string, quoteAssets []string) (string, string, error) {
	for _, q := range quoteAssets {
		if strings.HasSuffix(symbol, q) {
			return strings.TrimSuffix(symbol, q), q, nil
		}
	}
	return "", "", fmt.Errorf("could not determine base/quote for symbol %q", symbol)
}

func (c *Config) GetFee(exchange, quoteAsset string) types.Fee {
	ex := strings.ToUpper(exchange)
	qt := strings.ToUpper(quoteAsset)
	if exFees, ok := c.Fees.Exchanges[ex]; ok {
		if fee, ok := exFees[qt]; ok {
			return types.Fee{TakerBp: fee.Taker, MakerBp: fee.Maker}
		}
	}
	return types.Fee{TakerBp: c.Fees.Default.Taker, MakerBp: c.Fees.Default.Maker}
}