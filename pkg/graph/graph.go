package graph

import (
	"fmt"
	"strings"
	"sync"

	"github.com/armagg/circular-arbitrage-finder/pkg/logger"
	"github.com/armagg/circular-arbitrage-finder/pkg/types"

	"github.com/sirupsen/logrus"
)


type Index struct {
	Markets             []types.Market
	MarketIndexBySymbol map[string]int
	marketsByExchange   map[string]map[string]int
	Triangles           []types.Triangle
	TrianglesByMarket   map[int][]int
	mu                  sync.Mutex
}

func NewIndex() *Index {
	return &Index{
		Markets:             make([]types.Market, 0),
		MarketIndexBySymbol: make(map[string]int),
		marketsByExchange:   make(map[string]map[string]int),
		Triangles:           make([]types.Triangle, 0),
		TrianglesByMarket:   make(map[int][]int),
	}
}



func (idx *Index) AddMarket(m types.Market) (newTriangles []types.Triangle, isNew bool) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	key := fmt.Sprintf("%s:%s", strings.ToUpper(m.Exchange), strings.ToUpper(m.Symbol))
	if _, ok := idx.MarketIndexBySymbol[key]; ok {
		return nil, false
	}

	marketID := len(idx.Markets)
	idx.Markets = append(idx.Markets, m)
	idx.MarketIndexBySymbol[key] = marketID

	if _, ok := idx.marketsByExchange[m.Exchange]; !ok {
		idx.marketsByExchange[m.Exchange] = make(map[string]int)
	}
	pairKey := m.Base + "/" + m.Quote
	idx.marketsByExchange[m.Exchange][pairKey] = marketID

	newTriangles = idx.findNewTriangles(m, marketID)
	if len(newTriangles) > 0 {
		for _, t := range newTriangles {
			logger.Log.WithFields(logrus.Fields{
				"market_ids": t.MarketIds,
				"markets": []string{
					idx.Markets[t.MarketIds[0]].Symbol,
					idx.Markets[t.MarketIds[1]].Symbol,
					idx.Markets[t.MarketIds[2]].Symbol,
				},
			}).Info("graph: found triangle")
			idx.Triangles = append(idx.Triangles, t)
			ti := len(idx.Triangles) - 1
			for _, mid := range t.MarketIds {
				idx.TrianglesByMarket[mid] = append(idx.TrianglesByMarket[mid], ti)
			}
		}
	}
	return newTriangles, true
}



func (idx *Index) findNewTriangles(m types.Market, mID int) []types.Triangle {
	var triangles []types.Triangle

	byPair := idx.marketsByExchange[m.Exchange]
	if byPair == nil {
		return nil
	}
	marketsOnExchange := make([]types.Market, 0)
	for _, id := range byPair {
		marketsOnExchange = append(marketsOnExchange, idx.Markets[id])
	}


	a1, c1 := m.Base, m.Quote
	for _, m2 := range marketsOnExchange {
		if m2.Base == a1 && m2.Quote != c1 {
			b2 := m2.Quote
			if m3ID, ok := byPair[b2+"/"+c1]; ok {
				m2ID := byPair[m2.Base+"/"+m2.Quote]
				triangles = append(triangles, makeTriangle(mID, m2ID, m3ID, c1))
			}
		}
	}


	a2, b2 := m.Base, m.Quote
	for _, m1 := range marketsOnExchange {
		if m1.Base == a2 && m1.Quote != b2 {
			c1 := m1.Quote
			if m3ID, ok := byPair[b2+"/"+c1]; ok {
				m1ID := byPair[m1.Base+"/"+m1.Quote]
				triangles = append(triangles, makeTriangle(m1ID, mID, m3ID, c1))
			}
		}
	}


	b3, c3 := m.Base, m.Quote
	for _, m1 := range marketsOnExchange {
		if m1.Quote == c3 && m1.Base != b3 {
			a1 := m1.Base
			if m2ID, ok := byPair[a1+"/"+b3]; ok {
				m1ID := byPair[m1.Base+"/"+m1.Quote]
				triangles = append(triangles, makeTriangle(m1ID, m2ID, mID, c3))
			}
		}
	}
	return triangles
}

func makeTriangle(id1, id2, id3 int, quoteCcy string) types.Triangle {
	return types.Triangle{
		MarketIds: [3]int{id1, id2, id3},
		Dirs:      [3]int8{+1, -1, -1},
		QuoteCcy:  quoteCcy,
	}
}
