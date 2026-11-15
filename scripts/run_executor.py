#!/usr/bin/env python3
import os
import sys
import time
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


def ensure_python_stubs():
    _proto_stubs.ensure_python_stubs(
        [("executor.proto", "exec")],
        proto_dir=PROTO_DIR,
        gen_dir=GEN_DIR,
    )


def main():
    load_dotenv(os.path.join(ROOT, ".env"))
    ensure_python_stubs()
    _proto_stubs.ensure_sys_path(os.path.join(GEN_DIR, "exec"))
    import executor_pb2_grpc, executor_pb2

    class ExecutorServicer(executor_pb2_grpc.ExecutorServicer):
        def ProposePlan(self, request, context):
            ts = time.strftime("%Y-%m-%d %H:%M:%S")
            legs = []
            for l in request.legs:
                legs.append(f"{l.market} {l.side} {l.qty:.8f}@{l.limit_price:.8f}")
            legs_str = ", ".join(legs)
            print(
                f"{ts} PLAN exchange={request.exchange} quote={request.quote_ccy} profit={request.expected_profit_quote:.8f} slippage_bp={request.max_slippage_bp:.2f} valid_ms={request.valid_ms} legs=[{legs_str}] id={request.plan_id}"
            )
            return executor_pb2.ProposeReply(accepted=True, reason="ok")

    addr = os.environ.get("EXECUTOR_ADDR", "127.0.0.1:60051")
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=4))
    executor_pb2_grpc.add_ExecutorServicer_to_server(ExecutorServicer(), server)
    server.add_insecure_port(addr)
    server.start()
    print("Executor listening on", addr)
    try:
        server.wait_for_termination()
    except KeyboardInterrupt:
        server.stop(0)


if __name__ == "__main__":
    main()
