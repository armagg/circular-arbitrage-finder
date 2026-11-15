#!/usr/bin/env python3
import os
import sys
import time
from concurrent import futures

import grpc

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
        [("executor.proto", "exec"), ("marketdata.proto", "md")],
        proto_dir=PROTO_DIR,
        gen_dir=GEN_DIR,
    )


def run_executor_server(listen_addr):
    _proto_stubs.ensure_sys_path(os.path.join(GEN_DIR, "exec"))
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
    _proto_stubs.ensure_sys_path(os.path.join(GEN_DIR, "md"))
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
