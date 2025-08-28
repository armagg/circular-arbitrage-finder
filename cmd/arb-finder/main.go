package main

import (
	"context"
	"net"
	"os"

	"github.com/armagg/circular-arbitrage-finder/pkg/apiout"
	"github.com/armagg/circular-arbitrage-finder/pkg/bookstore"
	"github.com/armagg/circular-arbitrage-finder/pkg/config"
	"github.com/armagg/circular-arbitrage-finder/pkg/detector"
	"github.com/armagg/circular-arbitrage-finder/pkg/graph"
	"github.com/armagg/circular-arbitrage-finder/pkg/ingest"
	"github.com/armagg/circular-arbitrage-finder/pkg/logger"
	"github.com/armagg/circular-arbitrage-finder/pkg/profit"
	"github.com/armagg/circular-arbitrage-finder/pkg/registry"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	cfg, err := config.Load("config.yaml")
	if err != nil { logger.Log.Fatalf("failed to load config: %v", err) }
	if err := logger.Init(cfg.Log.Level); err != nil { logger.Log.Fatalf("failed to initialize logger: %v", err) }

	reg := registry.NewMarketRegistry()
	idx := graph.NewIndex()
	tob := bookstore.NewTopOfBookStore()
	obs := bookstore.NewOrderBookStore()
	sim := profit.NewTOBSimulator(cfg.Strategy.MinProfitEdge, cfg.Strategy.SlippageBp)
	var publisher apiout.Publisher = apiout.LogPublisher{}
	if addr := os.Getenv("EXECUTOR_ADDR"); addr != "" {
		if _, _, err := net.SplitHostPort(addr); err != nil { logger.Log.Fatalf("invalid EXECUTOR_ADDR: %v", err) }
		conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil { logger.Log.Fatalf("failed to dial executor: %v", err) }
		defer conn.Close()
		publisher = apiout.NewGRPCPublisher(conn)
	}
	det := detector.NewDetector(idx, tob, reg, sim, publisher)
	listenAddr := os.Getenv("INGRESS_ADDR"); if listenAddr == "" { listenAddr = ":50051" }
	ctx, cancel := context.WithCancel(context.Background()); defer cancel()
	srv := ingest.NewGRPCServer(tob, det, cfg, obs)
	go func() { if err := ingest.Serve(ctx, listenAddr, srv); err != nil { logger.Log.Fatalf("ingress server error: %v", err) } }()
	logger.Log.Infof("arb-finder listening on %s", listenAddr)
	select {}
}
