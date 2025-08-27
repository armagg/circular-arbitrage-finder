package registry

import (
	"sync"

	"github.com/armagg/circular-arbitrage-finder/pkg/types"
)

type MarketRegistry struct {
	mu      sync.RWMutex
	markets map[string]types.Market
	fees    map[string]types.Fee
}

func NewMarketRegistry() *MarketRegistry {
	return &MarketRegistry{
		markets: make(map[string]types.Market),
		fees:    make(map[string]types.Fee),
	}
}

func (r *MarketRegistry) UpsertMarket(m types.Market) {
	r.mu.Lock()
	r.markets[m.Symbol] = m
	r.mu.Unlock()
}

func (r *MarketRegistry) GetMarket(symbol string) (types.Market, bool) {
	r.mu.RLock()
	m, ok := r.markets[symbol]
	r.mu.RUnlock()
	return m, ok
}

func (r *MarketRegistry) SetFee(symbol string, f types.Fee) {
	r.mu.Lock()
	r.fees[symbol] = f
	r.mu.Unlock()
}

func (r *MarketRegistry) GetFee(symbol string) (types.Fee, bool) {
	r.mu.RLock()
	f, ok := r.fees[symbol]
	r.mu.RUnlock()
	return f, ok
}


func (r *MarketRegistry) Snapshot() (map[string]types.Market, map[string]types.Fee) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	m := make(map[string]types.Market, len(r.markets))
	for k, v := range r.markets {
		m[k] = v
	}
	f := make(map[string]types.Fee, len(r.fees))
	for k, v := range r.fees {
		f[k] = v
	}
	return m, f
}
