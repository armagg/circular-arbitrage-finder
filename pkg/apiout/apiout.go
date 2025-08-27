package apiout

import (
	"github.com/armagg/circular-arbitrage-finder/pkg/logger"
	"github.com/armagg/circular-arbitrage-finder/pkg/types"

	"github.com/sirupsen/logrus"
)

// Publisher emits executable plans to an external executor service.
type Publisher interface {
	Publish(p types.Plan) error
}

// LogPublisher prints the plan; acts as a placeholder before real gRPC implementation.
type LogPublisher struct{}

func (p LogPublisher) Publish(plan types.Plan) error {
	logger.Log.WithFields(logrus.Fields{
		"exchange":       plan.Exchange,
		"profit_quote":   plan.ExpectedProfitQuote,
		"quote_currency": plan.QuoteCurrency,
		"leg1":           formatLeg(plan.Legs[0]),
		"leg2":           formatLeg(plan.Legs[1]),
		"leg3":           formatLeg(plan.Legs[2]),
	}).Info("publishing plan")
	return nil
}

func formatLeg(leg types.TriangleLeg) string {
	return string(leg.Side) + " " + leg.Market
}
