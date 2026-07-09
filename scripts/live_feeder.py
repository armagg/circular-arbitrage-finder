#!/usr/bin/env python3
"""LIVE market-data feeder: real venue orderbooks -> arb-finder ingress.

This is the production counterpart of market_data_feeder.py (which
SIMULATES data for tests). It polls real books through
iranian_exchanges_sdk -- therefore **Toman-normalized**, the same units
the executor (market_maker/arb_executor.py) re-checks plans against. Feed
the finder anything else and every plan dies at the executor's freshness
gate, by design.

ONE EXCHANGE PER FINDER INSTANCE: the finder's book stores are keyed by
symbol only, so USDTIRT from two venues would overwrite each other. Run
one finder + one feeder per exchange (triangles are intra-exchange
anyway). This script enforces a single --exchange.

Env/flags:
  FEED_EXCHANGE / --exchange   e.g. nobitex   (SDK credential envs apply)
  FEED_MARKETS  / --markets    csv, e.g. USDTIRT,BTCIRT,BTCUSDT
                               (must close triangles for the finder to
                               find anything)
  FEED_INGRESS  / --ingress    finder address, default localhost:50052
  FEED_INTERVAL_SEC            poll interval, default 2.0
  FEED_DEPTH                   levels per side to push, default 5
"""
import argparse
import logging
import os
import sys
import time

import grpc
from decouple import config as env

THIS_DIR = os.path.dirname(os.path.abspath(__file__))
ROOT = os.path.abspath(os.path.join(THIS_DIR, ".."))
MD_DIR = os.path.join(ROOT, "pygen", "md")
for p in (ROOT, MD_DIR):
    if p not in sys.path:
        sys.path.insert(0, p)

import marketdata_pb2  # noqa: E402
import marketdata_pb2_grpc  # noqa: E402
from iranian_exchanges_sdk import get_exchanges, FiatUnit  # noqa: E402

logging.basicConfig(level=logging.INFO,
                    format="%(asctime)s %(levelname)s %(message)s")
logger = logging.getLogger("live_feeder")


def delta_stream(exchange, handles, interval_sec, depth, seq_start=1):
    """Yield one OrderBookDelta per (market, tick), forever."""
    seq = seq_start
    while True:
        started = time.time()
        for symbol, handle in handles.items():
            try:
                bids = handle.bids()[:depth]
                asks = handle.asks()[:depth]
            except Exception:
                logger.exception("book fetch failed for %s", symbol)
                continue
            if not bids or not asks:
                logger.warning("empty book for %s; skipping tick", symbol)
                continue
            yield marketdata_pb2.OrderBookDelta(
                market=marketdata_pb2.MarketId(exchange=exchange,
                                               symbol=symbol),
                sequence=seq,
                ts_ns=int(time.time() * 1e9),
                bids=[marketdata_pb2.Level(price=o.price, qty=o.amount)
                      for o in bids],
                asks=[marketdata_pb2.Level(price=o.price, qty=o.amount)
                      for o in asks],
                is_snapshot=True,
            )
            seq += 1
        time.sleep(max(0.0, interval_sec - (time.time() - started)))


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--exchange",
                        default=env("FEED_EXCHANGE", default=""))
    parser.add_argument("--markets",
                        default=env("FEED_MARKETS", default=""))
    parser.add_argument("--ingress",
                        default=env("FEED_INGRESS",
                                    default="localhost:50052"))
    parser.add_argument("--interval", type=float,
                        default=env("FEED_INTERVAL_SEC", default=2.0,
                                    cast=float))
    parser.add_argument("--depth", type=int,
                        default=env("FEED_DEPTH", default=5, cast=int))
    args = parser.parse_args()

    if not args.exchange or not args.markets:
        parser.error("--exchange and --markets are required "
                     "(or FEED_EXCHANGE / FEED_MARKETS)")
    markets = [m.strip().upper() for m in args.markets.split(",")
               if m.strip()]

    client = get_exchanges(args.exchange, fiat_unit=FiatUnit.TOMAN)
    handles = {m: client.market(m) for m in markets}
    logger.info("feeding %s %s -> %s every %.1fs (depth %d, Toman units)",
                args.exchange, markets, args.ingress, args.interval,
                args.depth)

    # PushDeltas is client-streaming: keep one stream open; on any error
    # reconnect with fresh sequence numbers (finder treats each delta as
    # a snapshot, so a reset is harmless).
    seq = 1
    while True:
        try:
            channel = grpc.insecure_channel(args.ingress)
            stub = marketdata_pb2_grpc.OrderBookIngressStub(channel)
            stub.PushDeltas(delta_stream(args.exchange, handles,
                                         args.interval, args.depth,
                                         seq_start=seq))
        except KeyboardInterrupt:
            return
        except Exception:
            seq += 1_000_000  # keep sequences monotonic across resets
            logger.exception("stream to %s broke; reconnecting in 5s",
                             args.ingress)
            time.sleep(5)


if __name__ == "__main__":
    main()
