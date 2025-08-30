package bookstore

import (
	"sort"
	"sync"
	"testing"

	"github.com/armagg/circular-arbitrage-finder/pkg/types"
)

func TestNewTopOfBookStore(t *testing.T) {
	store := NewTopOfBookStore()

	if store == nil {
		t.Error("NewTopOfBookStore should return a non-nil store")
	}

	if store.data == nil {
		t.Error("Store data map should be initialized")
	}

	if len(store.data) != 0 {
		t.Errorf("New store should have empty data map, got %d items", len(store.data))
	}
}

func TestTopOfBookStoreSet(t *testing.T) {
	store := NewTopOfBookStore()

	tob := types.TopOfBook{
		BidPx: 49990.0,
		BidSz: 2.1,
		AskPx: 50010.0,
		AskSz: 1.8,
		Seq:   12345,
		TsNs:  1640995200000000000,
	}

	store.Set("BTCUSDT", tob)

	// Verify the data was stored
	retrieved, exists := store.Get("BTCUSDT")
	if !exists {
		t.Error("Expected data to exist after Set")
	}

	if retrieved != tob {
		t.Error("Retrieved data should match stored data")
	}
}

func TestTopOfBookStoreGet(t *testing.T) {
	store := NewTopOfBookStore()

	// Test getting non-existent data
	_, exists := store.Get("NONEXISTENT")
	if exists {
		t.Error("Get should return false for non-existent data")
	}

	// Test getting existing data
	tob := types.TopOfBook{BidPx: 50000.0, AskPx: 50005.0}
	store.Set("BTCUSDT", tob)

	retrieved, exists := store.Get("BTCUSDT")
	if !exists {
		t.Error("Get should return true for existing data")
	}

	if retrieved != tob {
		t.Error("Retrieved data should match stored data")
	}
}

func TestTopOfBookStoreSnapshot(t *testing.T) {
	store := NewTopOfBookStore()

	// Add some test data
	data := map[string]types.TopOfBook{
		"BTCUSDT": {BidPx: 50000.0, AskPx: 50005.0},
		"ETHUSDT": {BidPx: 3000.0, AskPx: 3005.0},
	}

	for symbol, tob := range data {
		store.Set(symbol, tob)
	}

	snapshot := store.Snapshot()

	if len(snapshot) != len(data) {
		t.Errorf("Snapshot should have %d items, got %d", len(data), len(snapshot))
	}

	for symbol, expected := range data {
		actual, exists := snapshot[symbol]
		if !exists {
			t.Errorf("Snapshot should contain symbol %s", symbol)
		}
		if actual != expected {
			t.Errorf("Snapshot data for %s should match original", symbol)
		}
	}
}

func TestTopOfBookStoreConcurrency(t *testing.T) {
	store := NewTopOfBookStore()

	// Test concurrent access
	var wg sync.WaitGroup
	numGoroutines := 10
	numOperations := 100

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				symbol := "BTCUSDT"
				tob := types.TopOfBook{
					BidPx: float64(id*1000 + j),
					AskPx: float64(id*1000 + j + 5),
				}
				store.Set(symbol, tob)

				_, _ = store.Get(symbol)
			}
		}(i)
	}

	wg.Wait()

	// Verify final state
	_, exists := store.Get("BTCUSDT")
	if !exists {
		t.Error("Data should exist after concurrent operations")
	}
}

func TestNewOrderBookStore(t *testing.T) {
	store := NewOrderBookStore()

	if store == nil {
		t.Error("NewOrderBookStore should return a non-nil store")
	}

	if store.data == nil {
		t.Error("Store data map should be initialized")
	}

	if len(store.data) != 0 {
		t.Errorf("New store should have empty data map, got %d items", len(store.data))
	}
}

func TestOrderBookStoreUpsert(t *testing.T) {
	store := NewOrderBookStore()

	bids := []types.Level{
		{Price: 49980.0, Qty: 1.5},
		{Price: 49990.0, Qty: 2.1},
		{Price: 49970.0, Qty: 0.8},
	}

	asks := []types.Level{
		{Price: 50030.0, Qty: 1.2},
		{Price: 50020.0, Qty: 1.8},
		{Price: 50010.0, Qty: 2.5},
	}

	store.Upsert("BTCUSDT", bids, asks, 12345, 1640995200000000000, 0)

	retrieved, exists := store.Get("BTCUSDT")
	if !exists {
		t.Error("Order book should exist after upsert")
	}

	// Check that bids are sorted descending
	for i := 1; i < len(retrieved.Bids); i++ {
		if retrieved.Bids[i].Price > retrieved.Bids[i-1].Price {
			t.Error("Bids should be sorted descending by price")
		}
	}

	// Check that asks are sorted ascending
	for i := 1; i < len(retrieved.Asks); i++ {
		if retrieved.Asks[i].Price < retrieved.Asks[i-1].Price {
			t.Error("Asks should be sorted ascending by price")
		}
	}

	if retrieved.Seq != 12345 {
		t.Errorf("Expected seq 12345, got %d", retrieved.Seq)
	}

	if retrieved.TsNs != 1640995200000000000 {
		t.Errorf("Expected timestamp 1640995200000000000, got %d", retrieved.TsNs)
	}
}

func TestOrderBookStoreUpsertWithDepth(t *testing.T) {
	store := NewOrderBookStore()

	// Create more levels than depth limit
	bids := []types.Level{
		{Price: 49990.0, Qty: 2.1},
		{Price: 49980.0, Qty: 1.5},
		{Price: 49970.0, Qty: 0.8},
		{Price: 49960.0, Qty: 1.2},
		{Price: 49950.0, Qty: 0.5},
	}

	asks := []types.Level{
		{Price: 50010.0, Qty: 2.5},
		{Price: 50020.0, Qty: 1.8},
		{Price: 50030.0, Qty: 1.2},
		{Price: 50040.0, Qty: 0.8},
		{Price: 50050.0, Qty: 0.3},
	}

	depth := 3
	store.Upsert("BTCUSDT", bids, asks, 12345, 1640995200000000000, depth)

	retrieved, exists := store.Get("BTCUSDT")
	if !exists {
		t.Error("Order book should exist after upsert")
	}

	if len(retrieved.Bids) != depth {
		t.Errorf("Expected %d bids after depth limit, got %d", depth, len(retrieved.Bids))
	}

	if len(retrieved.Asks) != depth {
		t.Errorf("Expected %d asks after depth limit, got %d", depth, len(retrieved.Asks))
	}
}

func TestOrderBookStoreGet(t *testing.T) {
	store := NewOrderBookStore()

	// Test getting non-existent data
	_, exists := store.Get("NONEXISTENT")
	if exists {
		t.Error("Get should return false for non-existent data")
	}

	// Test getting existing data
	orderBook := types.OrderBook{
		Bids: []types.Level{{Price: 50000.0, Qty: 1.0}},
		Asks: []types.Level{{Price: 50005.0, Qty: 1.0}},
		Seq:  12345,
		TsNs: 1640995200000000000,
	}

	bids := []types.Level{{Price: 50000.0, Qty: 1.0}}
	asks := []types.Level{{Price: 50005.0, Qty: 1.0}}
	store.Upsert("BTCUSDT", bids, asks, 12345, 1640995200000000000, 0)

	retrieved, exists := store.Get("BTCUSDT")
	if !exists {
		t.Error("Get should return true for existing data")
	}

	if len(retrieved.Bids) != len(orderBook.Bids) || len(retrieved.Asks) != len(orderBook.Asks) {
		t.Error("Retrieved order book should match stored data")
	}
}

// Test the sorting optimization functions
func TestIsSortedDescending(t *testing.T) {
	tests := []struct {
		name     string
		levels   []types.Level
		expected bool
	}{
		{
			name: "Already sorted descending",
			levels: []types.Level{
				{Price: 50000.0, Qty: 1.0},
				{Price: 49990.0, Qty: 1.5},
				{Price: 49980.0, Qty: 2.0},
			},
			expected: true,
		},
		{
			name: "Not sorted descending",
			levels: []types.Level{
				{Price: 49980.0, Qty: 2.0},
				{Price: 50000.0, Qty: 1.0},
				{Price: 49990.0, Qty: 1.5},
			},
			expected: false,
		},
		{
			name:     "Empty slice",
			levels:   []types.Level{},
			expected: true,
		},
		{
			name: "Single element",
			levels: []types.Level{
				{Price: 50000.0, Qty: 1.0},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isSortedDescending(tt.levels)
			if result != tt.expected {
				t.Errorf("isSortedDescending() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestIsSortedAscending(t *testing.T) {
	tests := []struct {
		name     string
		levels   []types.Level
		expected bool
	}{
		{
			name: "Already sorted ascending",
			levels: []types.Level{
				{Price: 49980.0, Qty: 2.0},
				{Price: 49990.0, Qty: 1.5},
				{Price: 50000.0, Qty: 1.0},
			},
			expected: true,
		},
		{
			name: "Not sorted ascending",
			levels: []types.Level{
				{Price: 50000.0, Qty: 1.0},
				{Price: 49980.0, Qty: 2.0},
				{Price: 49990.0, Qty: 1.5},
			},
			expected: false,
		},
		{
			name:     "Empty slice",
			levels:   []types.Level{},
			expected: true,
		},
		{
			name: "Single element",
			levels: []types.Level{
				{Price: 50000.0, Qty: 1.0},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isSortedAscending(tt.levels)
			if result != tt.expected {
				t.Errorf("isSortedAscending() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestInsertionSortDescending(t *testing.T) {
	levels := []types.Level{
		{Price: 49980.0, Qty: 2.0},
		{Price: 50000.0, Qty: 1.0},
		{Price: 49990.0, Qty: 1.5},
	}

	insertionSortDescending(levels)

	// Verify sorted descending
	for i := 1; i < len(levels); i++ {
		if levels[i].Price > levels[i-1].Price {
			t.Error("Levels should be sorted descending by price")
		}
	}

	// Check specific order
	expected := []float64{50000.0, 49990.0, 49980.0}
	for i, level := range levels {
		if level.Price != expected[i] {
			t.Errorf("Expected price %f at index %d, got %f", expected[i], i, level.Price)
		}
	}
}

func TestInsertionSortAscending(t *testing.T) {
	levels := []types.Level{
		{Price: 50000.0, Qty: 1.0},
		{Price: 49980.0, Qty: 2.0},
		{Price: 49990.0, Qty: 1.5},
	}

	insertionSortAscending(levels)

	// Verify sorted ascending
	for i := 1; i < len(levels); i++ {
		if levels[i].Price < levels[i-1].Price {
			t.Error("Levels should be sorted ascending by price")
		}
	}

	// Check specific order
	expected := []float64{49980.0, 49990.0, 50000.0}
	for i, level := range levels {
		if level.Price != expected[i] {
			t.Errorf("Expected price %f at index %d, got %f", expected[i], i, level.Price)
		}
	}
}

// Test sorting optimization behavior
func TestSortingOptimization(t *testing.T) {
	store := NewOrderBookStore()

	// Test with already sorted data (should skip sorting)
	sortedBids := []types.Level{
		{Price: 50000.0, Qty: 1.0},
		{Price: 49990.0, Qty: 1.5},
		{Price: 49980.0, Qty: 2.0},
	}
	sortedAsks := []types.Level{
		{Price: 50010.0, Qty: 2.0},
		{Price: 50020.0, Qty: 1.5},
		{Price: 50030.0, Qty: 1.0},
	}

	store.Upsert("BTCUSDT", sortedBids, sortedAsks, 12345, 1640995200000000000, 0)

	retrieved, _ := store.Get("BTCUSDT")

	// Verify bids are still sorted descending
	for i := 1; i < len(retrieved.Bids); i++ {
		if retrieved.Bids[i].Price > retrieved.Bids[i-1].Price {
			t.Error("Bids should remain sorted descending")
		}
	}

	// Verify asks are still sorted ascending
	for i := 1; i < len(retrieved.Asks); i++ {
		if retrieved.Asks[i].Price < retrieved.Asks[i-1].Price {
			t.Error("Asks should remain sorted ascending")
		}
	}
}

func TestOrderBookStoreConcurrency(t *testing.T) {
	store := NewOrderBookStore()

	// Test concurrent access
	var wg sync.WaitGroup
	numGoroutines := 10
	numOperations := 50

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				symbol := "BTCUSDT"
				bids := []types.Level{
					{Price: float64(50000 + id*10 + j), Qty: 1.0},
				}
				asks := []types.Level{
					{Price: float64(50010 + id*10 + j), Qty: 1.0},
				}
				store.Upsert(symbol, bids, asks, uint64(id*numOperations+j), 1640995200000000000, 0)

				_, _ = store.Get(symbol)
			}
		}(i)
	}

	wg.Wait()

	// Verify final state
	_, exists := store.Get("BTCUSDT")
	if !exists {
		t.Error("Data should exist after concurrent operations")
	}
}

// Benchmark tests for sorting performance
func BenchmarkInsertionSortDescending(b *testing.B) {
	levels := make([]types.Level, 32)
	for i := range levels {
		levels[i] = types.Level{Price: float64(32 - i), Qty: 1.0}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		insertionSortDescending(levels)
	}
}

func BenchmarkGoSortDescending(b *testing.B) {
	levels := make([]types.Level, 32)
	for i := range levels {
		levels[i] = types.Level{Price: float64(32 - i), Qty: 1.0}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sort.Slice(levels, func(j, k int) bool { return levels[j].Price > levels[k].Price })
	}
}

func BenchmarkInsertionSortAscending(b *testing.B) {
	levels := make([]types.Level, 32)
	for i := range levels {
		levels[i] = types.Level{Price: float64(i), Qty: 1.0}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		insertionSortAscending(levels)
	}
}

func BenchmarkGoSortAscending(b *testing.B) {
	levels := make([]types.Level, 32)
	for i := range levels {
		levels[i] = types.Level{Price: float64(i), Qty: 1.0}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sort.Slice(levels, func(j, k int) bool { return levels[j].Price < levels[k].Price })
	}
}
