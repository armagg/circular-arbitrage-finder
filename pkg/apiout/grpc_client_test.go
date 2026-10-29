package apiout

import (
	"context"
	"net"
	"sync/atomic"
	"testing"

	"github.com/armagg/circular-arbitrage-finder/pkg/types"
	exppb "github.com/armagg/circular-arbitrage-finder/proto/exec"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

type recordingExecutor struct {
	exppb.UnimplementedExecutorServer
	calls atomic.Int32
	seen  chan *exppb.Plan
	err   error
}

func (s *recordingExecutor) ProposePlan(_ context.Context, plan *exppb.Plan) (*exppb.ProposeReply, error) {
	s.calls.Add(1)
	s.seen <- plan
	if s.err != nil {
		return nil, s.err
	}
	return &exppb.ProposeReply{Accepted: true}, nil
}

func TestGRPCPublisherTransport(t *testing.T) {
	for _, code := range []codes.Code{codes.OK, codes.Unavailable} {
		t.Run(code.String(), func(t *testing.T) {
			listener := bufconn.Listen(1 << 20)
			server := grpc.NewServer()
			executor := &recordingExecutor{
				seen: make(chan *exppb.Plan, 8),
				err:  status.Error(code, "executor result"),
			}
			exppb.RegisterExecutorServer(server, executor)
			go server.Serve(listener)
			t.Cleanup(server.Stop)
			t.Cleanup(func() { listener.Close() })
			conn, err := grpc.NewClient("passthrough:///executor",
				grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
					return listener.Dial()
				}),
				grpc.WithTransportCredentials(insecure.NewCredentials()),
			)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { conn.Close() })
			plan := types.Plan{
				Exchange: "bitpin", PlanID: "transport-regression",
				QuoteCurrency: "USDT", ExpectedProfitQuote: 1.25,
				ValidMs: 500, MaxSlippageBp: 2,
				Legs: [3]types.TriangleLeg{
					{Market: "BTCUSDT", Side: types.SideBuy, Qty: 0.001, LimitPrice: 50000},
					{Market: "ETHBTC", Side: types.SideBuy, Qty: 0.01, LimitPrice: 0.05},
					{Market: "ETHUSDT", Side: types.SideSell, Qty: 0.01, LimitPrice: 2501},
				},
			}
			err = NewGRPCPublisher(conn).Publish(plan)
			if status.Code(err) != code {
				t.Fatalf("status = %v, want %v", status.Code(err), code)
			}
			if executor.calls.Load() != 1 {
				t.Fatalf("executor calls = %d, want 1", executor.calls.Load())
			}
			got := <-executor.seen
			if got.PlanId != plan.PlanID || got.Exchange != plan.Exchange ||
				got.QuoteCcy != plan.QuoteCurrency || got.ValidMs != plan.ValidMs ||
				got.MaxSlippageBp != plan.MaxSlippageBp || got.ExpectedProfitQuote != plan.ExpectedProfitQuote {
				t.Fatalf("plan metadata changed across gRPC: %v", got)
			}
			if len(got.Legs) != len(plan.Legs) {
				t.Fatalf("legs = %d, want %d", len(got.Legs), len(plan.Legs))
			}
			for i, leg := range plan.Legs {
				if got.Legs[i].Market != leg.Market || got.Legs[i].Side != string(leg.Side) ||
					got.Legs[i].Qty != leg.Qty || got.Legs[i].LimitPrice != leg.LimitPrice {
					t.Fatalf("leg %d changed across gRPC: %v", i, got.Legs[i])
				}
			}
		})
	}
}
