# Running circular-arbitrage-finder

Detects circular arbitrage opportunities across configured exchanges/quote
assets. The detection logic is solid; **the executor is still a logging
stub** — it finds opportunities but never places real trades (HANDOFF §5.7).
Safe to run.

## Fastest way: the monorepo infra stack

```bash
cd "../infra"
./scripts/local.sh up --all   # profile "apps-trading"
```
gRPC ingress on `:50052`.

## Standalone (host-run)

```bash
go build -o arb-finder ./cmd/arb-finder
INGRESS_ADDR=:50052 EXECUTOR_ADDR=127.0.0.1:60051 ./arb-finder
```
Strategy/fee parameters live in `config.yaml` (quote assets, per-exchange
taker/maker fees, `min_profit_edge`, `slippage_bp`, `trade_amount`,
`orderbook_depth`) — edit that, not env vars, for strategy tuning. `.env`
only carries a couple of fee overrides and the two addresses above.

## Tests

```bash
go build ./... && go vet ./... && go test ./...
```
