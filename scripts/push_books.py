#!/usr/bin/env python3
import os
import sys
import time

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


def ensure_python_stubs():
    _proto_stubs.ensure_python_stubs(
        [("marketdata.proto", "md")],
        proto_dir=PROTO_DIR,
        gen_dir=GEN_DIR,
    )


def main():
    load_dotenv(os.path.join(ROOT, ".env"))
    ensure_python_stubs()
    _proto_stubs.ensure_sys_path(os.path.join(GEN_DIR, "md"))
    import marketdata_pb2_grpc, marketdata_pb2

    ingress = os.environ.get("INGRESS_ADDR", "127.0.0.1:50051")
    if ingress.startswith(":"):
        ingress = "127.0.0.1" + ingress
    channel = grpc.insecure_channel(ingress)
    stub = marketdata_pb2_grpc.OrderBookIngressStub(channel)

    def gen():
        seq = 1
        yield marketdata_pb2.OrderBookDelta(
            market=marketdata_pb2.MarketId(exchange="BINANCE", symbol="BTCUSDT"),
            sequence=seq,
            ts_ns=int(time.time_ns()),
            bids=[marketdata_pb2.Level(price=60000.0, qty=1.0)],
            asks=[marketdata_pb2.Level(price=60010.0, qty=1.0)],
            is_snapshot=True,
        )
        seq += 1
        yield marketdata_pb2.OrderBookDelta(
            market=marketdata_pb2.MarketId(exchange="BINANCE", symbol="ETHBTC"),
            sequence=seq,
            ts_ns=int(time.time_ns()),
            bids=[marketdata_pb2.Level(price=0.06, qty=10.0)],
            asks=[marketdata_pb2.Level(price=0.06001, qty=10.0)],
            is_snapshot=False,
        )
        seq += 1
        yield marketdata_pb2.OrderBookDelta(
            market=marketdata_pb2.MarketId(exchange="BINANCE", symbol="ETHUSDT"),
            sequence=seq,
            ts_ns=int(time.time_ns()),
            bids=[marketdata_pb2.Level(price=3580.0, qty=10.0)],
            asks=[marketdata_pb2.Level(price=3585.0, qty=10.0)],
            is_snapshot=False,
        )

    try:
        ack = stub.PushDeltas(gen())
        print("Ingress ack:", ack)
    except grpc.RpcError as e:
        print("PushDeltas error:", e)


if __name__ == "__main__":
    main()
