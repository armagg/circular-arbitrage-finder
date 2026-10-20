#!/usr/bin/env python3
"""
Market Data Feeder - Python Integration Test Client

This script feeds realistic market data to the Go arbitrage finder service
using the same protobuf definitions. It simulates real market conditions
with varying prices, volumes, and update frequencies.
"""

import os
import sys
import time
import random
import argparse
import logging
from typing import List, Dict, Iterator
import shutil

import grpc
from dotenv import load_dotenv

THIS_DIR = os.path.dirname(os.path.abspath(__file__))
ROOT = os.path.abspath(os.path.join(THIS_DIR, ".."))
PROTO_DIR = os.path.join(ROOT, "proto")
GEN_DIR = os.path.join(ROOT, "pygen")

if ROOT not in sys.path:
    sys.path.insert(0, ROOT)

logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)


def ensure_python_stubs():
    """Generate Python protobuf stubs if they don't exist."""
    try:
        import importlib
        importlib.import_module("pygen.md.marketdata_pb2")
        importlib.import_module("pygen.md.marketdata_pb2_grpc")
        return
    except Exception:
        pass

    logger.info("Generating Python protobuf stubs...")

    os.makedirs(GEN_DIR, exist_ok=True)
    os.makedirs(os.path.join(GEN_DIR, "md"), exist_ok=True)

    try:
        from grpc_tools import protoc

        args = [
            "protoc",
            f"-I{PROTO_DIR}",
            f"--python_out={GEN_DIR}",
            f"--grpc_python_out={GEN_DIR}",
            os.path.join(PROTO_DIR, "marketdata.proto"),
        ]

        if protoc.main(args) != 0:
            logger.error("protoc python generation failed")
            sys.exit(1)

        src_pb = os.path.join(GEN_DIR, "marketdata_pb2.py")
        src_grpc = os.path.join(GEN_DIR, "marketdata_pb2_grpc.py")
        dst_dir = os.path.join(GEN_DIR, "md")

        if os.path.exists(src_pb):
            shutil.move(src_pb, os.path.join(dst_dir, "marketdata_pb2.py"))
        if os.path.exists(src_grpc):
            shutil.move(src_grpc, os.path.join(dst_dir, "marketdata_pb2_grpc.py"))

        for p in [GEN_DIR, dst_dir]:
            initp = os.path.join(p, "__init__.py")
            if not os.path.exists(initp):
                with open(initp, "w") as f:
                    f.write("")

        logger.info("Protobuf stubs generated successfully")

    except ImportError:
        logger.error("grpc_tools not found. Install with: pip install grpcio-tools")
        sys.exit(1)


class MarketDataGenerator:
    """Generates realistic market data for testing."""

    def __init__(self, base_prices: Dict[str, float] = None):
        self.sequence = 1
        self.base_prices = base_prices or {
            "BTCUSDT": 50000.0,
            "ETHUSDT": 3000.0,
            "ETHBTC": 0.06,
            "ADAUSDT": 1.5,
            "ADABTC": 0.000030,
            "ADAETH": 0.0005,
        }
        self.volatility = {
            "BTCUSDT": 0.002,
            "ETHUSDT": 0.003,
            "ETHBTC": 0.005,
            "ADAUSDT": 0.008,
            "ADABTC": 0.010,
            "ADAETH": 0.012,
        }
        self.current_prices = self.base_prices.copy()
        self.last_update = time.time()

    def generate_realistic_levels(self, symbol: str, side: str, num_levels: int = 5) -> List[Dict]:
        """Generate realistic bid/ask levels with decreasing volume."""
        levels = []
        base_price = self.current_prices[symbol]

        price_change = random.gauss(0, self.volatility[symbol])
        base_price *= (1 + price_change)
        self.current_prices[symbol] = base_price

        spread = self.volatility[symbol] * base_price * 0.001

        for i in range(num_levels):
            if side == "bid":
                price = base_price - (spread * (i + 1))
                qty = random.uniform(0.1, 5.0) * (1 / (i + 1))
            else:
                price = base_price + (spread * (i + 1))
                qty = random.uniform(0.1, 5.0) * (1 / (i + 1))

            levels.append({"price": price, "qty": qty})

        return levels

    def create_orderbook_delta(self, symbol: str, exchange: str = "BINANCE", is_snapshot: bool = False):
        """Create an OrderBookDelta message."""
        sys.path.insert(0, os.path.join(GEN_DIR, "md"))
        import marketdata_pb2

        bids = self.generate_realistic_levels(symbol, "bid")
        asks = self.generate_realistic_levels(symbol, "ask")

        pb_bids = [marketdata_pb2.Level(price=b["price"], qty=b["qty"]) for b in bids]
        pb_asks = [marketdata_pb2.Level(price=a["price"], qty=a["qty"]) for a in asks]

        delta = marketdata_pb2.OrderBookDelta(
            market=marketdata_pb2.MarketId(exchange=exchange, symbol=symbol),
            sequence=self.sequence,
            ts_ns=int(time.time_ns()),
            bids=pb_bids,
            asks=pb_asks,
            is_snapshot=is_snapshot,
        )

        self.sequence += 1
        return delta

    def get_arbitrage_opportunity(self) -> Dict[str, Dict]:
        """Generate a set of prices that create arbitrage opportunities."""
        btc_usdt_price = 50000.0
        eth_btc_price = 0.06
        eth_usdt_price = 2980.0

        return {
            "BTCUSDT": {"bid": btc_usdt_price, "ask": btc_usdt_price + 10},
            "ETHBTC": {"bid": eth_btc_price, "ask": eth_btc_price + 0.0001},
            "ETHUSDT": {"bid": eth_usdt_price, "ask": eth_usdt_price + 5},
        }


class MarketDataFeeder:
    """Feeds market data to the Go service."""

    def __init__(self, server_address: str, symbols: List[str] = None):
        self.server_address = server_address
        self.symbols = symbols or ["BTCUSDT", "ETHUSDT", "ETHBTC"]
        self.generator = MarketDataGenerator()
        self.logger = logger

    def connect(self):
        """Establish gRPC connection."""
        try:
            channel = grpc.insecure_channel(self.server_address)
            sys.path.insert(0, os.path.join(GEN_DIR, "md"))
            import marketdata_pb2_grpc
            stub = marketdata_pb2_grpc.OrderBookIngressStub(channel)
            return stub
        except Exception as e:
            self.logger.error(f"Failed to connect to {self.server_address}: {e}")
            raise

    def send_initial_snapshots(self, stub):
        """Send initial order book snapshots."""
        self.logger.info("Sending initial order book snapshots...")

        for symbol in self.symbols:
            delta = self.generator.create_orderbook_delta(symbol, is_snapshot=True)
            yield delta
            self.logger.info(f"Sent snapshot for {symbol}")
            time.sleep(0.1)

    def send_continuous_updates(self, stub, interval: float = 1.0):
        """Send continuous market data updates."""
        self.logger.info(f"Sending continuous updates every {interval} seconds...")

        while True:
            for symbol in self.symbols:
                delta = self.generator.create_orderbook_delta(symbol, is_snapshot=False)
                yield delta
                self.logger.debug(f"Sent update for {symbol}")
                time.sleep(interval / len(self.symbols))

    def send_arbitrage_opportunity(self, stub):
        """Send a specific set of prices that create arbitrage opportunities."""
        self.logger.info("Sending arbitrage opportunity prices...")

        arb_prices = self.generator.get_arbitrage_opportunity()

        for symbol, prices in arb_prices.items():
            sys.path.insert(0, os.path.join(GEN_DIR, "md"))
            import marketdata_pb2

            bids = [marketdata_pb2.Level(price=prices["bid"] - i, qty=10.0 / (i + 1)) for i in range(3)]
            asks = [marketdata_pb2.Level(price=prices["ask"] + i, qty=10.0 / (i + 1)) for i in range(3)]

            delta = marketdata_pb2.OrderBookDelta(
                market=marketdata_pb2.MarketId(exchange="BINANCE", symbol=symbol),
                sequence=self.generator.sequence,
                ts_ns=int(time.time_ns()),
                bids=bids,
                asks=asks,
                is_snapshot=False,
            )

            yield delta
            self.generator.sequence += 1
            self.logger.info(f"Sent arbitrage prices for {symbol}: bid={prices['bid']}, ask={prices['ask']}")
            time.sleep(0.1)

    def run(self, mode: str = "continuous", interval: float = 1.0):
        """Run the data feeder."""
        try:
            stub = self.connect()
            self.logger.info(f"Connected to {self.server_address}")

            def data_generator():
                if mode == "initial":
                    yield from self.send_initial_snapshots(stub)
                elif mode == "arbitrage":
                    yield from self.send_arbitrage_opportunity(stub)
                    yield from self.send_continuous_updates(stub, interval)
                else:
                    yield from self.send_initial_snapshots(stub)
                    yield from self.send_continuous_updates(stub, interval)

            response = stub.PushDeltas(data_generator())
            self.logger.info(f"Data feed completed. Server response: {response}")

        except grpc.RpcError as e:
            self.logger.error(f"gRPC error: {e}")
        except KeyboardInterrupt:
            self.logger.info("Interrupted by user")
        except Exception as e:
            self.logger.error(f"Unexpected error: {e}")
            raise


def main():
    parser = argparse.ArgumentParser(description="Market Data Feeder for Arbitrage Finder")
    parser.add_argument("--server", default="127.0.0.1:50051",
                       help="Server address (default: 127.0.0.1:50051)")
    parser.add_argument("--symbols", nargs="+",
                       default=["BTCUSDT", "ETHUSDT", "ETHBTC"],
                       help="Symbols to feed (default: BTCUSDT ETHUSDT ETHBTC)")
    parser.add_argument("--mode", choices=["initial", "continuous", "arbitrage"],
                       default="continuous",
                       help="Feed mode: initial (one snapshot), continuous (ongoing), arbitrage (create arbitrage opportunity)")
    parser.add_argument("--interval", type=float, default=1.0,
                       help="Update interval in seconds (default: 1.0)")
    parser.add_argument("--verbose", "-v", action="store_true",
                       help="Enable verbose logging")

    args = parser.parse_args()

    if args.verbose:
        logging.getLogger().setLevel(logging.DEBUG)

    load_dotenv(os.path.join(ROOT, ".env"))

    if not args.server.startswith("127.0.0.1") and not args.server.startswith("localhost"):
        os.environ["INGRESS_ADDR"] = args.server

    ensure_python_stubs()

    feeder = MarketDataFeeder(args.server, args.symbols)
    feeder.run(args.mode, args.interval)


if __name__ == "__main__":
    main()
