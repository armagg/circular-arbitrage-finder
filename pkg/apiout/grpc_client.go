package apiout

import (
	"context"
	"time"

	"github.com/armagg/circular-arbitrage-finder/pkg/types"
	exppb "github.com/armagg/circular-arbitrage-finder/proto/exec"

	"google.golang.org/grpc"
)


type GRPCPublisher struct {
	client exppb.ExecutorClient
}

func NewGRPCPublisher(conn *grpc.ClientConn) *GRPCPublisher {
	return &GRPCPublisher{client: exppb.NewExecutorClient(conn)}
}

func (p *GRPCPublisher) Publish(plan types.Plan) error {
	legs := make([]*exppb.TriangleLeg, 0, 3)
	for _, l := range plan.Legs {
		legs = append(legs, &exppb.TriangleLeg{
			Market:     l.Market,
			Side:       string(l.Side),
			Qty:        l.Qty,
			LimitPrice: l.LimitPrice,
		})
	}
	req := &exppb.Plan{
		Exchange:            plan.Exchange,
		Legs:                legs,
		ExpectedProfitQuote: plan.ExpectedProfitQuote,
		QuoteCcy:            plan.QuoteCurrency,
		ValidMs:             plan.ValidMs,
		MaxSlippageBp:       plan.MaxSlippageBp,
		PlanId:              plan.PlanID,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	_, err := p.client.ProposePlan(ctx, req)
	return err
}

