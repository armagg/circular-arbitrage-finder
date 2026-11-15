#!/usr/bin/env python3
"""Mock executor server: receives strategies from the Go finder and logs them."""

import os
import sys
import time
import json
import argparse
import logging
from datetime import datetime
from typing import Dict
from concurrent import futures

import grpc
from dotenv import load_dotenv

THIS_DIR = os.path.dirname(os.path.abspath(__file__))
ROOT = os.path.abspath(os.path.join(THIS_DIR, ".."))
PROTO_DIR = os.path.join(ROOT, "proto")
GEN_DIR = os.path.join(ROOT, "pygen")

if ROOT not in sys.path:
    sys.path.insert(0, ROOT)

if __package__:
    from . import _proto_stubs
else:
    try:
        import _proto_stubs
    except ImportError:
        from scripts import _proto_stubs

logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)


def ensure_python_stubs():
    """Generate Python protobuf stubs if they don't exist."""
    _proto_stubs.ensure_python_stubs(
        [("executor.proto", "exec")],
        proto_dir=PROTO_DIR,
        gen_dir=GEN_DIR,
        logger=logger,
    )


class StrategyLogger:
    """Logs and analyzes arbitrage strategies."""

    def __init__(self, log_file: str = None, json_output: bool = False):
        self.log_file = log_file
        self.json_output = json_output
        self.strategies_received = []
        self.logger = logger

        if self.log_file:
            with open(self.log_file, 'w') as f:
                f.write(f"# Arbitrage Strategy Log - Started at {datetime.now().isoformat()}\n")
                f.write("# Format: timestamp, plan_id, exchange, profit, legs...\n\n")

    def log_strategy(self, strategy_data: Dict):
        """Log a strategy in the specified format."""
        timestamp = datetime.now().isoformat()
        plan_id = strategy_data.get('plan_id', 'unknown')
        exchange = strategy_data.get('exchange', 'unknown')
        profit = strategy_data.get('expected_profit_quote', 0.0)
        quote_currency = strategy_data.get('quote_ccy', 'unknown')
        legs = strategy_data.get('legs', [])

        if self.json_output:
            log_entry = {
                'timestamp': timestamp,
                'plan_id': plan_id,
                'exchange': exchange,
                'expected_profit_quote': profit,
                'quote_currency': quote_currency,
                'legs': legs,
                'max_slippage_bp': strategy_data.get('max_slippage_bp', 0),
                'valid_ms': strategy_data.get('valid_ms', 0),
            }
            print(json.dumps(log_entry, indent=2))
        else:
            legs_str = ", ".join([
                f"{leg['market']} {leg['side']} {leg['qty']:.8f}@{leg['limit_price']:.8f}"
                for leg in legs
            ])

            log_msg = (
                f"STRATEGY | {timestamp} | {plan_id} | {exchange} | "
                f"Profit: {profit:.8f} {quote_currency} | "
                f"Legs: [{legs_str}] | "
                f"Slippage: {strategy_data.get('max_slippage_bp', 0):.2f}bp | "
                f"Valid: {strategy_data.get('valid_ms', 0)}ms"
            )

            print(log_msg)
            self.logger.info(log_msg)

        if self.log_file:
            with open(self.log_file, 'a') as f:
                if self.json_output:
                    json.dump({
                        'timestamp': timestamp,
                        'strategy': strategy_data
                    }, f, indent=2)
                    f.write('\n')
                else:
                    f.write(f"{timestamp},{plan_id},{exchange},{profit:.8f},{quote_currency}")
                    for leg in legs:
                        f.write(f",{leg['market']},{leg['side']},{leg['qty']:.8f},{leg['limit_price']:.8f}")
                    f.write('\n')

        strategy_data['timestamp'] = timestamp
        self.strategies_received.append(strategy_data)

    def get_statistics(self) -> Dict:
        """Get statistics about received strategies."""
        if not self.strategies_received:
            return {'total_strategies': 0}

        profits = [s.get('expected_profit_quote', 0) for s in self.strategies_received]
        exchanges = [s.get('exchange', 'unknown') for s in self.strategies_received]

        return {
            'total_strategies': len(self.strategies_received),
            'average_profit': sum(profits) / len(profits),
            'max_profit': max(profits),
            'min_profit': min(profits),
            'exchanges': list(set(exchanges)),
            'time_span': {
                'first': self.strategies_received[0]['timestamp'],
                'last': self.strategies_received[-1]['timestamp']
            }
        }

    def print_summary(self):
        """Print a summary of all received strategies."""
        stats = self.get_statistics()

        print("\n" + "="*80)
        print("ARBITRAGE STRATEGY SUMMARY")
        print("="*80)
        print(f"Total strategies received: {stats['total_strategies']}")

        if stats['total_strategies'] > 0:
            print(f"Average profit: {stats['average_profit']:.8f}")
            print(f"Max profit: {stats['max_profit']:.8f}")
            print(f"Min profit: {stats['min_profit']:.8f}")
            print(f"Exchanges: {', '.join(stats['exchanges'])}")
            print(f"Time span: {stats['time_span']['first']} to {stats['time_span']['last']}")

            exchange_stats = {}
            for strategy in self.strategies_received:
                exchange = strategy.get('exchange', 'unknown')
                if exchange not in exchange_stats:
                    exchange_stats[exchange] = []
                exchange_stats[exchange].append(strategy.get('expected_profit_quote', 0))

            print("\nPer-exchange statistics:")
            for exchange, profits in exchange_stats.items():
                avg_profit = sum(profits) / len(profits)
                print(f"  {exchange}: {avg_profit:.8f} average profit")
        print("="*80)


class ExecutorServicer:
    """gRPC service implementation for receiving strategies."""

    def __init__(self, strategy_logger: StrategyLogger):
        self.strategy_logger = strategy_logger
        self.logger = logger

    def ProposePlan(self, request, context):
        """Handle incoming strategy proposals."""
        try:
            strategy_data = {
                'plan_id': request.plan_id,
                'exchange': request.exchange,
                'expected_profit_quote': request.expected_profit_quote,
                'quote_ccy': request.quote_ccy,
                'max_slippage_bp': request.max_slippage_bp,
                'valid_ms': request.valid_ms,
                'legs': []
            }

            for leg in request.legs:
                strategy_data['legs'].append({
                    'market': leg.market,
                    'side': leg.side,
                    'qty': leg.qty,
                    'limit_price': leg.limit_price,
                })

            self.strategy_logger.log_strategy(strategy_data)

            self.logger.debug(f"Accepted strategy {request.plan_id}")

            _proto_stubs.ensure_sys_path(os.path.join(GEN_DIR, "exec"))
            import executor_pb2
            return executor_pb2.ProposeReply(accepted=True, reason="Strategy logged successfully")

        except Exception as e:
            self.logger.error(f"Error processing strategy {request.plan_id}: {e}")
            _proto_stubs.ensure_sys_path(os.path.join(GEN_DIR, "exec"))
            import executor_pb2
            return executor_pb2.ProposeReply(accepted=False, reason=f"Error: {str(e)}")


def main():
    parser = argparse.ArgumentParser(description="Strategy Logger for Arbitrage Finder")
    parser.add_argument("--address", default="127.0.0.1:60051",
                       help="Server address to listen on (default: 127.0.0.1:60051)")
    parser.add_argument("--log-file", help="File to save strategy logs")
    parser.add_argument("--json-output", action="store_true",
                       help="Output strategies in JSON format")
    parser.add_argument("--summary-interval", type=int, default=30,
                       help="Print summary every N seconds (0 to disable)")
    parser.add_argument("--verbose", "-v", action="store_true",
                       help="Enable verbose logging")

    args = parser.parse_args()

    if args.verbose:
        logging.getLogger().setLevel(logging.DEBUG)

    load_dotenv(os.path.join(ROOT, ".env"))

    ensure_python_stubs()

    logger_instance = StrategyLogger(args.log_file, args.json_output)

    server = grpc.server(futures.ThreadPoolExecutor(max_workers=4))

    _proto_stubs.ensure_sys_path(os.path.join(GEN_DIR, "exec"))
    import executor_pb2_grpc

    servicer = ExecutorServicer(logger_instance)
    executor_pb2_grpc.add_ExecutorServicer_to_server(servicer, server)

    server.add_insecure_port(args.address)
    server.start()

    logger.info(f"Strategy Logger listening on {args.address}")
    logger.info("Ready to receive arbitrage strategies from Go service")
    logger.info("Press Ctrl+C to stop and show summary")

    if args.log_file:
        logger.info(f"Strategies will be logged to: {args.log_file}")

    try:
        start_time = time.time()
        while True:
            time.sleep(1)

            if args.summary_interval > 0:
                elapsed = time.time() - start_time
                if elapsed >= args.summary_interval:
                    logger_instance.print_summary()
                    start_time = time.time()

    except KeyboardInterrupt:
        logger.info("Stopping strategy logger...")
        server.stop(0)

        logger_instance.print_summary()

        if args.log_file:
            logger.info(f"Strategies logged to: {args.log_file}")


if __name__ == "__main__":
    main()
