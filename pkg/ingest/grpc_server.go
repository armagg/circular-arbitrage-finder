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
	Books    *bookstore.TopOfBookStore
	Detector *detector.Detector
	Config   *config.Config
}

func NewGRPCServer(books *bookstore.TopOfBookStore, det *detector.Detector, cfg *config.Config) *GRPCServer {
	return &GRPCServer{Books: books, Detector: det, Config: cfg}
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
				logger.Log.WithFields(logrus.Fields{
					"exchange": exchange,
					"symbol":   symbol,
					"error":    err,
				}).Warn("ingest: failed to parse new market")
				continue
			}
			if _, isNew := s.Detector.Index.AddMarket(market); isNew {
				s.Detector.Registry.UpsertMarket(market)
				s.Detector.Registry.SetFee(symbol, s.Config.GetFee(exchange, market.Quote))
				logger.Log.WithFields(logrus.Fields{
					"exchange": exchange,
					"symbol":   symbol,
				}).Info("ingest: discovered and added new market")
			}
		}

		var bidPx, bidSz, askPx, askSz float64
		if len(d.Bids) > 0 {
			bidPx = d.Bids[0].Price
			bidSz = d.Bids[0].Qty
		}
		if len(d.Asks) > 0 {
			askPx = d.Asks[0].Price
			askSz = d.Asks[0].Qty
		}
		if bidPx > 0 && askPx > 0 {
			s.Books.Set(symbol, types.TopOfBook{BidPx: bidPx, BidSz: bidSz, AskPx: askPx, AskSz: askSz, Seq: d.Sequence, TsNs: int64(d.TsNs)})
			if s.Detector != nil {
				s.Detector.OnMarketChange(exchange, symbol, s.Config.Strategy.TradeAmount)
			}
		}
	}
}


func Serve(ctx context.Context, listenAddr string, srv *GRPCServer) error {
	lis, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return err
	}
	grpcServer := grpc.NewServer()
	mdpb.RegisterOrderBookIngressServer(grpcServer, srv)
	go func() {
		<-ctx.Done()
		grpcServer.GracefulStop()
	}()
	logger.Log.Infof("ingress gRPC listening on %s", listenAddr)
	return grpcServer.Serve(lis)
}
