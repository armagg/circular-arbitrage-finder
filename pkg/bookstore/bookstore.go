package bookstore

import (
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
