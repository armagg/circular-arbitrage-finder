#!/usr/bin/env python3
import os
import sys
import time
import shutil
from concurrent import futures

import grpc

THIS_DIR = os.path.dirname(os.path.abspath(__file__))
ROOT = os.path.abspath(os.path.join(THIS_DIR, ".."))
PROTO_DIR = os.path.join(ROOT, "proto")
GEN_DIR = os.path.join(ROOT, "pygen")
if ROOT not in sys.path:
    sys.path.insert(0, ROOT)


def ensure_python_stubs():
    try:
        import importlib

        importlib.import_module("pygen.exec.executor_pb2")
        importlib.import_module("pygen.exec.executor_pb2_grpc")
        importlib.import_module("pygen.md.marketdata_pb2")
        importlib.import_module("pygen.md.marketdata_pb2_grpc")
        return
    except Exception:
        pass
    os.makedirs(GEN_DIR, exist_ok=True)
    os.makedirs(os.path.join(GEN_DIR, "exec"), exist_ok=True)
    os.makedirs(os.path.join(GEN_DIR, "md"), exist_ok=True)
    from grpc_tools import protoc

    for p in [GEN_DIR, os.path.join(GEN_DIR, "exec"), os.path.join(GEN_DIR, "md")]:
        initp = os.path.join(p, "__init__.py")
        if not os.path.exists(initp):
            with open(initp, "w") as f:
                f.write("")

    args = [
        "protoc",
        f"-I{PROTO_DIR}",
        f"--python_out={GEN_DIR}",
        f"--grpc_python_out={GEN_DIR}",
        os.path.join(PROTO_DIR, "executor.proto"),
        os.path.join(PROTO_DIR, "marketdata.proto"),
    ]
    if protoc.main(args) != 0:
        print("protoc python generation failed", file=sys.stderr)
        sys.exit(1)
    for name, sub in [("executor", "exec"), ("marketdata", "md")]:
        src_pb = os.path.join(GEN_DIR, f"{name}_pb2.py")
        src_grpc = os.path.join(GEN_DIR, f"{name}_pb2_grpc.py")
        dst_dir = os.path.join(GEN_DIR, sub)
        if os.path.exists(src_pb):
            shutil.move(src_pb, os.path.join(dst_dir, f"{name}_pb2.py"))
        if os.path.exists(src_grpc):
            shutil.move(src_grpc, os.path.join(dst_dir, f"{name}_pb2_grpc.py"))


def run_executor_server(listen_addr):
    sys.path.insert(0, os.path.join(GEN_DIR, "exec"))
    import executor_pb2_grpc, executor_pb2

    class ExecutorServicer(executor_pb2_grpc.ExecutorServicer):
        def ProposePlan(self, request, context):
            print("Executor received plan:", request)
            return executor_pb2.ProposeReply(accepted=True, reason="ok")

    server = grpc.server(futures.ThreadPoolExecutor(max_workers=2))
    executor_pb2_grpc.add_ExecutorServicer_to_server(ExecutorServicer(), server)
    server.add_insecure_port(listen_addr)
    server.start()
    return server


def push_deltas(ingress_addr):
    sys.path.insert(0, os.path.join(GEN_DIR, "md"))
    import marketdata_pb2_grpc, marketdata_pb2

    if ingress_addr.startswith(":"):
        ingress_addr = "127.0.0.1" + ingress_addr
    channel = grpc.insecure_channel(ingress_addr)
    stub = marketdata_pb2_grpc.OrderBookIngressStub(channel)
    stream = stub.PushDeltas()

    def send(market, bid_px, bid_sz, ask_px, ask_sz):
        stream.send(
            marketdata_pb2.OrderBookDelta(
                market=marketdata_pb2.MarketId(exchange="BINANCE", symbol=market),
                sequence=1,
                ts_ns=int(time.time_ns()),
                bids=[marketdata_pb2.Level(price=bid_px, qty=bid_sz)],
                asks=[marketdata_pb2.Level(price=ask_px, qty=ask_sz)],
                is_snapshot=True,
            )
        )

    send("BTCUSDT", 60000.0, 1.0, 60010.0, 1.0)
    send("ETHBTC", 0.06, 10.0, 0.06001, 10.0)
    send("ETHUSDT", 3600.0, 10.0, 3605.0, 10.0)

    try:
        ack = stream.close_and_grpc_response()
        print("Ingress ack:", ack)
    except grpc.RpcError as e:
        print("PushDeltas error:", e)


if __name__ == "__main__":
    ensure_python_stubs()
    exec_addr = os.environ.get("EXECUTOR_ADDR", "127.0.0.1:60051")
    ingress_addr = os.environ.get("INGRESS_ADDR", "127.0.0.1:50051")

    server = run_executor_server(exec_addr)
    print("Executor server started on", exec_addr)

    time.sleep(1.0)

    push_deltas(ingress_addr)

    time.sleep(0.5)
    server.stop(0)
