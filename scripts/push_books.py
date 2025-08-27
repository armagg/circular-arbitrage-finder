#!/usr/bin/env python3
import os
import sys
import time
import shutil

import grpc
from dotenv import load_dotenv

THIS_DIR = os.path.dirname(os.path.abspath(__file__))
ROOT = os.path.abspath(os.path.join(THIS_DIR, ".."))
PROTO_DIR = os.path.join(ROOT, "proto")
GEN_DIR = os.path.join(ROOT, "pygen")
if ROOT not in sys.path:
    sys.path.insert(0, ROOT)


def ensure_python_stubs():
    try:
        import importlib

        importlib.import_module("pygen.md.marketdata_pb2")
        importlib.import_module("pygen.md.marketdata_pb2_grpc")
        return
    except Exception:
        pass
    os.makedirs(GEN_DIR, exist_ok=True)
    os.makedirs(os.path.join(GEN_DIR, "md"), exist_ok=True)
    from grpc_tools import protoc

    args = [
        "protoc",
        f"-I{PROTO_DIR}",
        f"--python_out={GEN_DIR}",
        f"--grpc_python_out={GEN_DIR}",
        os.path.join(PROTO_DIR, "marketdata.proto"),
    ]
    if protoc.main(args) != 0:
        print("protoc python generation failed", file=sys.stderr)
        sys.exit(1)
    src_pb = os.path.join(GEN_DIR, "marketdata_pb2.py")
    src_grpc = os.path.join(GEN_DIR, "marketdata_pb2_grpc.py")
    dst_dir = os.path.join(GEN_DIR, "md")
    if os.path.exists(src_pb):
        shutil.move(src_pb, os.path.join(dst_dir, "marketdata_pb2.py"))
    if os.path.exists(src_grpc):
        shutil.move(src_grpc, os.path.join(dst_dir, "marketdata_pb2_grpc.py"))
    for p in [GEN_DIR, os.path.join(GEN_DIR, "md")]:
        initp = os.path.join(p, "__init__.py")
        if not os.path.exists(initp):
            with open(initp, "w") as f:
                f.write("")


def main():
    load_dotenv(os.path.join(ROOT, ".env"))
    ensure_python_stubs()
    sys.path.insert(0, os.path.join(GEN_DIR, "md"))
    import marketdata_pb2_grpc, marketdata_pb2

    ingress = os.environ.get("INGRESS_ADDR", "127.0.0.1:50051")
    if ingress.startswith(":"):
        ingress = "127.0.0.1" + ingress
    channel = grpc.insecure_channel(ingress)
    stub = marketdata_pb2_grpc.OrderBookIngressStub(channel)

    def gen():
        seq = 1
        # Original non-profitable prices:
        # BTCUSDT: 60020/60030, ETHBTC: 0.06010/0.06012, ETHUSDT: 3610/3612
        # Profitable cycle: USDT -> ETH -> BTC -> USDT
        # (ETHBTC_bid * BTCUSDT_bid) / ETHUSDT_ask > 1 + fees
        # We need ETHUSDT ask to be lower.
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
