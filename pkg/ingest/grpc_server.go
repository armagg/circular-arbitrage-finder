package ingest

import (
	"context"
	"fmt"
	"net"
	"strings"

	"github.com/armagg/circular-arbitrage-finder/pkg/bookstore"
	"github.com/armagg/circular-arbitrage-finder/pkg/config"
	"github.com/armagg/circular-arbitrage-finder/pkg/detector"
	"github.com/armagg/circular-arbitrage-finder/pkg/logger"
	"github.com/armagg/circular-arbitrage-finder/pkg/types"
	mdpb "github.com/armagg/circular-arbitrage-finder/proto/md"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

type GRPCServer struct {
	mdpb.UnimplementedOrderBookIngressServer
	TOBStore   *bookstore.TopOfBookStore
	OBStore    *bookstore.OrderBookStore
	Detector   *detector.Detector
	Config     *config.Config
}

func NewGRPCServer(tobs *bookstore.TopOfBookStore, det *detector.Detector, cfg *config.Config, obs *bookstore.OrderBookStore) *GRPCServer {
	return &GRPCServer{TOBStore: tobs, OBStore: obs, Detector: det, Config: cfg}
}

func (s *GRPCServer) PushDeltas(stream mdpb.OrderBookIngress_PushDeltasServer) error {
	for {
		d, err := stream.Recv()
		if err != nil {
			return stream.SendAndClose(&mdpb.Ack{Ok: false})
		}
		exchange := strings.ToUpper(d.GetMarket().GetExchange())
		symbol := strings.ToUpper(d.GetMarket().GetSymbol())

		key := fmt.Sprintf("%s:%s", exchange, symbol)
		if _, ok := s.Detector.Index.MarketIndexBySymbol[key]; !ok {
			market, err := s.Config.ParseMarket(exchange, symbol)
			if err != nil {
				logger.Log.WithFields(logrus.Fields{"exchange": exchange, "symbol": symbol, "error": err}).Warn("ingest: failed to parse new market")
				continue
			}
			if _, isNew := s.Detector.Index.AddMarket(market); isNew {
				s.Detector.Registry.UpsertMarket(market)
				s.Detector.Registry.SetFee(symbol, s.Config.GetFee(exchange, market.Quote))
				logger.Log.WithFields(logrus.Fields{"exchange": exchange, "symbol": symbol}).Info("ingest: discovered and added new market")
			}
		}

		var bids []types.Level
		var asks []types.Level
		if len(d.Bids) > 0 {
			b := d.Bids[0]
			bids = append(bids, types.Level{Price: b.Price, Qty: b.Qty})
		}
		if len(d.Asks) > 0 {
			a := d.Asks[0]
			asks = append(asks, types.Level{Price: a.Price, Qty: a.Qty})
		}
		// Depth-aware store
		s.OBStore.Upsert(symbol, toLevels(d.Bids), toLevels(d.Asks), d.Sequence, int64(d.TsNs), s.Config.Strategy.OrderbookDepth)
		// Maintain legacy TOB for detector/simulator compatibility
		if len(bids) > 0 && len(asks) > 0 {
			s.TOBStore.Set(symbol, types.TopOfBook{BidPx: bids[0].Price, BidSz: bids[0].Qty, AskPx: asks[0].Price, AskSz: asks[0].Qty, Seq: d.Sequence, TsNs: int64(d.TsNs)})
			if s.Detector != nil {
				s.Detector.OnMarketChange(exchange, symbol, s.pickTradeAmount(symbol))
			}
		}
	}
}

func toLevels(src []*mdpb.Level) []types.Level {
	res := make([]types.Level, 0, len(src))
	for _, l := range src {
		res = append(res, types.Level{Price: l.Price, Qty: l.Qty})
	}
	return res
}

func (s *GRPCServer) pickTradeAmount(symbol string) float64 {
	for q, amt := range s.Config.Strategy.TradeAmounts {
		if strings.HasSuffix(symbol, q) { return amt }
	}
	return s.Config.Strategy.TradeAmount
}

func Serve(ctx context.Context, listenAddr string, srv *GRPCServer) error {
	lis, err := net.Listen("tcp", listenAddr)
	if err != nil { return err }
	grpcServer := grpc.NewServer()
	mdpb.RegisterOrderBookIngressServer(grpcServer, srv)
	go func() { <-ctx.Done(); grpcServer.GracefulStop() }()
	logger.Log.Infof("ingress gRPC listening on %s", listenAddr)
	return grpcServer.Serve(lis)
}
