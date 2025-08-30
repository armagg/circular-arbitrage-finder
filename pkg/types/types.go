package types

type Side string


const (
	SideBuy  Side = "BUY"
	SideSell Side = "SELL"
)

type Market struct {
	Exchange string
	Symbol   string
	Base     string
	Quote    string
	Multiplier int64

	MinQty      float64
	StepSize    float64
	MinNotional float64 
	PriceTick   float64
}

type Fee struct {
	TakerBp float64
	MakerBp float64
}

type Level struct {
	Price float64
	Qty   float64
}

type TopOfBook struct {
	BidPx float64 
	BidSz float64
	AskPx float64
	AskSz float64
	Seq   uint64
	TsNs  int64 //
}

type OrderBook struct {
	Bids []Level
	Asks []Level
	Seq  uint64
	TsNs int64
}

type Triangle struct {
	MarketIds [3]int
	Dirs      [3]int8
	QuoteCcy  string
}

type TriangleLeg struct {
	Market     string
	Side       Side
	Qty        float64
	LimitPrice float64
}

type Plan struct {
	Exchange            string
	Legs                [3]TriangleLeg
	ExpectedProfitQuote float64
	QuoteCurrency       string
	ValidMs             uint64
	MaxSlippageBp       float64
	PlanID              string
}
