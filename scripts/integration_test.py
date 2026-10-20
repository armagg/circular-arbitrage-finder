#!/usr/bin/env python3
"""
Integration Test - Complete Arbitrage Detection Pipeline

This script demonstrates the complete integration between:
1. Python Market Data Feeder -> Go Arbitrage Finder -> Python Strategy Logger

It tests the real functionality by feeding market data and logging detected strategies.
"""

import os
import sys
import time
import subprocess
import signal
import argparse
import logging
from pathlib import Path

THIS_DIR = os.path.dirname(os.path.abspath(__file__))
ROOT = os.path.abspath(os.path.join(THIS_DIR, ".."))

if ROOT not in sys.path:
    sys.path.insert(0, ROOT)

logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)


class IntegrationTest:
    """Manages the complete integration test pipeline."""

    def __init__(self, go_binary_path: str = None, config_path: str = None):
        self.go_binary_path = go_binary_path or os.path.join(ROOT, "bin", "arb-finder")
        self.config_path = config_path or os.path.join(ROOT, "config.yaml")
        self.processes = []
        self.logger = logger

    def check_dependencies(self):
        """Check that all required components are available."""
        if not os.path.exists(self.go_binary_path):
            self.logger.error(f"Go binary not found at: {self.go_binary_path}")
            self.logger.info("Please build the Go binary first:")
            self.logger.info("  cd /path/to/project && go build -o bin/arb-finder ./cmd/arb-finder")
            return False

        if not os.path.exists(self.config_path):
            self.logger.error(f"Config file not found at: {self.config_path}")
            return False

        feeder_script = os.path.join(THIS_DIR, "market_data_feeder.py")
        logger_script = os.path.join(THIS_DIR, "strategy_logger.py")

        if not os.path.exists(feeder_script):
            self.logger.error(f"Market data feeder script not found: {feeder_script}")
            return False

        if not os.path.exists(logger_script):
            self.logger.error(f"Strategy logger script not found: {logger_script}")
            return False

        return True

    def start_go_service(self):
        """Start the Go arbitrage finder service."""
        self.logger.info("Starting Go arbitrage finder service...")

        env = os.environ.copy()
        env["INGRESS_ADDR"] = "127.0.0.1:50051"
        env["EXECUTOR_ADDR"] = "127.0.0.1:60051"

        try:
            process = subprocess.Popen(
                [self.go_binary_path],
                cwd=ROOT,
                env=env,
                stdout=subprocess.PIPE,
                stderr=subprocess.PIPE,
                text=True
            )
            self.processes.append(("go-service", process))
            self.logger.info("Go service started successfully")
            return True
        except Exception as e:
            self.logger.error(f"Failed to start Go service: {e}")
            return False

    def start_strategy_logger(self):
        """Start the Python strategy logger."""
        self.logger.info("Starting Python strategy logger...")

        try:
            cmd = [sys.executable, os.path.join(THIS_DIR, "strategy_logger.py")]
            process = subprocess.Popen(
                cmd,
                cwd=THIS_DIR,
                stdout=subprocess.PIPE,
                stderr=subprocess.PIPE,
                text=True
            )
            self.processes.append(("strategy-logger", process))
            self.logger.info("Strategy logger started successfully")
            return True
        except Exception as e:
            self.logger.error(f"Failed to start strategy logger: {e}")
            return False

    def run_market_data_feeder(self, mode: str = "arbitrage", duration: int = 10):
        """Run the market data feeder."""
        self.logger.info(f"Running market data feeder in {mode} mode for {duration} seconds...")

        try:
            cmd = [
                sys.executable,
                os.path.join(THIS_DIR, "market_data_feeder.py"),
                "--mode", mode,
                "--interval", "0.5"
            ]

            process = subprocess.Popen(
                cmd,
                cwd=THIS_DIR,
                stdout=subprocess.PIPE,
                stderr=subprocess.PIPE,
                text=True
            )

            try:
                process.wait(timeout=duration)
                self.logger.info("Market data feeder completed successfully")
                return True
            except subprocess.TimeoutExpired:
                process.terminate()
                self.logger.info("Market data feeder timed out (expected)")
                return True

        except Exception as e:
            self.logger.error(f"Failed to run market data feeder: {e}")
            return False

    def monitor_processes(self, timeout: int = 30):
        """Monitor running processes and collect their output."""
        self.logger.info("Monitoring processes...")

        start_time = time.time()
        while time.time() - start_time < timeout:
            all_running = True
            for name, process in self.processes:
                if process.poll() is not None:
                    stdout, stderr = process.communicate()
                    if stdout:
                        self.logger.info(f"{name} stdout: {stdout.strip()}")
                    if stderr:
                        self.logger.error(f"{name} stderr: {stderr.strip()}")
                else:
                    all_running = False

            if all_running:
                time.sleep(1)
            else:
                break

    def cleanup(self):
        """Clean up all running processes."""
        self.logger.info("Cleaning up processes...")

        for name, process in self.processes:
            if process.poll() is None:
                self.logger.info(f"Terminating {name}...")
                try:
                    process.terminate()
                    process.wait(timeout=5)
                except subprocess.TimeoutExpired:
                    self.logger.warning(f"Force killing {name}...")
                    process.kill()
                    process.wait()

        self.processes.clear()

    def run_test(self, mode: str = "arbitrage", duration: int = 15):
        """Run the complete integration test."""
        self.logger.info("Starting integration test...")
        self.logger.info(f"Test mode: {mode}")
        self.logger.info(f"Test duration: {duration} seconds")

        try:
            if not self.check_dependencies():
                return False

            if not self.start_go_service():
                return False

            time.sleep(2)

            if not self.start_strategy_logger():
                return False

            time.sleep(1)

            if not self.run_market_data_feeder(mode, duration):
                return False

            self.monitor_processes(duration + 5)

            self.logger.info("Integration test completed successfully!")
            return True

        except Exception as e:
            self.logger.error(f"Integration test failed: {e}")
            return False

        finally:
            self.cleanup()


def main():
    parser = argparse.ArgumentParser(description="Integration Test for Arbitrage Detection Pipeline")
    parser.add_argument("--go-binary", help="Path to Go binary")
    parser.add_argument("--config", help="Path to config file")
    parser.add_argument("--mode", choices=["initial", "continuous", "arbitrage"],
                       default="arbitrage",
                       help="Test mode (default: arbitrage)")
    parser.add_argument("--duration", type=int, default=15,
                       help="Test duration in seconds (default: 15)")
    parser.add_argument("--verbose", "-v", action="store_true",
                       help="Enable verbose logging")

    args = parser.parse_args()

    if args.verbose:
        logging.getLogger().setLevel(logging.DEBUG)

    test = IntegrationTest(args.go_binary, args.config)

    success = test.run_test(args.mode, args.duration)

    if success:
        logger.info("✅ Integration test PASSED")
        sys.exit(0)
    else:
        logger.error("❌ Integration test FAILED")
        sys.exit(1)


if __name__ == "__main__":
    main()
