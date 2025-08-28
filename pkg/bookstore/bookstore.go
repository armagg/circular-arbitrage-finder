package bookstore

import (
	"sort"
	"sync"

	"github.com/armagg/circular-arbitrage-finder/pkg/types"
)

type TopOfBookStore struct {
	mu   sync.RWMutex
	data map[string]types.TopOfBook
}

func NewTopOfBookStore() *TopOfBookStore {
	return &TopOfBookStore{data: make(map[string]types.TopOfBook)}
}

func (s *TopOfBookStore) Set(symbol string, tob types.TopOfBook) {
	s.mu.Lock()
	s.data[symbol] = tob
	s.mu.Unlock()
}

func (s *TopOfBookStore) Get(symbol string) (types.TopOfBook, bool) {
	s.mu.RLock()
	v, ok := s.data[symbol]
	s.mu.RUnlock()
	return v, ok
}

func (s *TopOfBookStore) Snapshot() map[string]types.TopOfBook {
	s.mu.RLock()
	defer s.mu.RUnlock()
	cp := make(map[string]types.TopOfBook, len(s.data))
	for k, v := range s.data {
		cp[k] = v
	}
	return cp
}

type OrderBookStore struct {
	mu   sync.RWMutex
	data map[string]types.OrderBook
}

func NewOrderBookStore() *OrderBookStore {
	return &OrderBookStore{data: make(map[string]types.OrderBook)}
}

func (s *OrderBookStore) Upsert(symbol string, bids []types.Level, asks []types.Level, seq uint64, ts int64, depth int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sort.Slice(bids, func(i, j int) bool { return bids[i].Price > bids[j].Price })
	sort.Slice(asks, func(i, j int) bool { return asks[i].Price < asks[j].Price })
	if depth > 0 {
		if len(bids) > depth { bids = bids[:depth] }
		if len(asks) > depth { asks = asks[:depth] }
	}
	s.data[symbol] = types.OrderBook{Bids: bids, Asks: asks, Seq: seq, TsNs: ts}
}

func (s *OrderBookStore) Get(symbol string) (types.OrderBook, bool) {
	s.mu.RLock()
	v, ok := s.data[symbol]
	s.mu.RUnlock()
	return v, ok
}
