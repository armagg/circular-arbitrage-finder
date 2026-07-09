# Python Integration Test Scripts

This directory contains Python scripts for testing the arbitrage finder integration with external systems. These scripts use the same protobuf definitions as the Go service to provide realistic end-to-end testing.

## Overview

The integration test suite consists of three main components:

1. **Market Data Feeder** (`market_data_feeder.py`) - Sends realistic market data to the Go service
2. **Strategy Logger** (`strategy_logger.py`) - Receives and logs arbitrage strategies
3. **Integration Test** (`integration_test.py`) - Orchestrates the complete test pipeline

## Prerequisites

### 1. Build the Go Service

```bash
cd /path/to/circular-arbitrage-finder
go build -o bin/arb-finder ./cmd/arb-finder
```

### 2. Install Python Dependencies

```bash
cd scripts
pip install grpcio grpcio-tools python-dotenv
```

### 3. Environment Setup

Create a `.env` file in the project root:

```bash
# Go service configuration
INGRESS_ADDR=127.0.0.1:50051
EXECUTOR_ADDR=127.0.0.1:60051

# Python service configuration
MARKET_DATA_ADDR=127.0.0.1:50051
STRATEGY_LOGGER_ADDR=127.0.0.1:60051
```

## Scripts Usage

### 1. Market Data Feeder

Sends realistic market data to the Go arbitrage finder service.

```bash
# Basic usage - continuous market data
python market_data_feeder.py

# Send arbitrage opportunity data
python market_data_feeder.py --mode arbitrage

# Custom server and symbols
python market_data_feeder.py --server 192.168.1.100:50051 --symbols BTCUSDT ETHUSDT

# Options
--server ADDRESS      Server address (default: 127.0.0.1:50051)
--symbols SYMBOLS     Trading symbols (default: BTCUSDT ETHUSDT ETHBTC)
--mode MODE          Feed mode: initial, continuous, arbitrage (default: continuous)
--interval SECONDS   Update interval (default: 1.0)
--verbose            Enable verbose logging
```

### 2. Strategy Logger

Acts as an executor server to receive and log arbitrage strategies.

```bash
# Basic usage
python strategy_logger.py

# Log to file with JSON output
python strategy_logger.py --log-file strategies.log --json-output

# Custom address
python strategy_logger.py --address 0.0.0.0:60051

# Options
--address ADDRESS       Server address (default: 127.0.0.1:60051)
--log-file FILE        Save strategies to file
--json-output          Output in JSON format
--summary-interval SEC Print summary every N seconds (default: 30)
--verbose              Enable verbose logging
```

### 3. Integration Test

Runs the complete end-to-end test pipeline.

```bash
# Basic integration test
python integration_test.py

# Custom configuration
python integration_test.py --go-binary /path/to/arb-finder --config /path/to/config.yaml

# Quick test
python integration_test.py --mode arbitrage --duration 5

# Options
--go-binary PATH      Path to Go binary
--config PATH         Path to config file
--mode MODE          Test mode: initial, continuous, arbitrage
--duration SECONDS   Test duration
--verbose            Enable verbose logging
```

## Test Scenarios

### 1. Basic Functionality Test

```bash
# Terminal 1: Start Go service
./bin/arb-finder

# Terminal 2: Start strategy logger
python strategy_logger.py

# Terminal 3: Send market data
python market_data_feeder.py --mode arbitrage
```

### 2. Continuous Market Data Test

```bash
# Terminal 1: Start services
python integration_test.py --mode continuous --duration 60
```

### 3. Load Testing

```bash
# Send high-frequency market data
python market_data_feeder.py --mode continuous --interval 0.1 --symbols BTCUSDT ETHUSDT ETHBTC ADABTC
```

### 4. Arbitrage Detection Test

```bash
# Create arbitrage opportunities
python market_data_feeder.py --mode arbitrage

# Monitor strategies
python strategy_logger.py --log-file arbitrage_test.log
```

## Output Examples

### Strategy Logger Output

```
2024-01-15 10:30:15,678 - INFO - Strategy Logger listening on 127.0.0.1:60051
2024-01-15 10:30:20,123 - INFO - STRATEGY | 2024-01-15T10:30:20.123456 | plan-123 | BINANCE | Profit: 25.50 USDT | Legs: [BTCUSDT BUY 1.50000000@50000.00000000, ETHBTC SELL 1.50000000@0.06000000, ETHUSDT SELL 50.00000000@1500.00000000] | Slippage: 5.00bp | Valid: 250ms
```

### JSON Output

```json
{
  "timestamp": "2024-01-15T10:30:20.123456",
  "plan_id": "plan-123",
  "exchange": "BINANCE",
  "expected_profit_quote": 25.5,
  "quote_currency": "USDT",
  "legs": [
    {
      "market": "BTCUSDT",
      "side": "BUY",
      "qty": 1.5,
      "limit_price": 50000.0
    }
  ],
  "max_slippage_bp": 5.0,
  "valid_ms": 250
}
```

## Troubleshooting

### 1. Connection Issues

```bash
# Check if ports are available
netstat -tlnp | grep 50051
netstat -tlnp | grep 60051

# Kill processes using ports
lsof -ti:50051 | xargs kill -9
lsof -ti:60051 | xargs kill -9
```

### 2. Protobuf Generation Issues

```bash
# Regenerate Python stubs
cd scripts
python -c "from grpc_tools import protoc; protoc.main(['protoc', '-I../proto', '--python_out=../pygen', '--grpc_python_out=../pygen', '../proto/marketdata.proto', '../proto/executor.proto'])"
```

### 3. Import Errors

```bash
# Ensure Python path is correct
export PYTHONPATH="/path/to/project:$PYTHONPATH"
```

### 4. Go Service Issues

```bash
# Check Go service logs
RUST_LOG=debug ./bin/arb-finder

# Verify config file
cat config.yaml
```

## Architecture

```
┌─────────────────┐    gRPC    ┌─────────────────┐    gRPC    ┌─────────────────┐
│ Market Data     │ ────────── │   Go Service    │ ────────── │ Strategy Logger │
│ Feeder (Python) │            │ (Arbitrage      │            │    (Python)     │
│                 │            │   Detector)     │            │                 │
└─────────────────┘            └─────────────────┘            └─────────────────┘
         │                             │                             │
         └─────────────────────────────┼─────────────────────────────┘
                                       │
                                       ▼
                              ┌─────────────────┐
                              │   Log Files     │
                              │   JSON/CSV      │
                              └─────────────────┘
```

## Performance Metrics

The integration tests measure:

- **Latency**: Time from market data input to strategy output
- **Throughput**: Number of strategies processed per second
- **Accuracy**: Correctness of arbitrage detection
- **Reliability**: Error handling and recovery

## Extending the Tests

### Adding New Market Symbols

```python
# In market_data_feeder.py
symbols = ["BTCUSDT", "ETHUSDT", "ETHBTC", "ADAUSDT", "ADABTC"]
feeder = MarketDataFeeder("127.0.0.1:50051", symbols)
```

### Custom Strategy Analysis

```python
# In strategy_logger.py
def analyze_strategy(strategy):
    # Add custom analysis logic
    profit_margin = strategy['expected_profit_quote']
    if profit_margin > 100:
        logger.warning(f"High profit strategy detected: {profit_margin}")
```

### Integration with Other Systems

```python
# Example: Send strategies to external trading system
def execute_strategy(strategy):
    # Integration with trading API
    for leg in strategy['legs']:
        # Execute each leg
        execute_trade(leg['market'], leg['side'], leg['qty'], leg['limit_price'])
```

This test suite provides a solid foundation for validating the arbitrage detection system and can be extended to integrate with real trading systems and market data feeds.
