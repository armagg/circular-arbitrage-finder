#!/usr/bin/env python3
import os
import sys
import shutil
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


def ensure_python_stubs():
    try:
        import importlib

        importlib.import_module("pygen.exec.executor_pb2")
        importlib.import_module("pygen.exec.executor_pb2_grpc")
        return
    except Exception:
        pass
    os.makedirs(GEN_DIR, exist_ok=True)
    os.makedirs(os.path.join(GEN_DIR, "exec"), exist_ok=True)
    from grpc_tools import protoc

    args = [
        "protoc",
        f"-I{PROTO_DIR}",
        f"--python_out={GEN_DIR}",
        f"--grpc_python_out={GEN_DIR}",
        os.path.join(PROTO_DIR, "executor.proto"),
    ]
    if protoc.main(args) != 0:
        print("protoc python generation failed", file=sys.stderr)
        sys.exit(1)
    src_pb = os.path.join(GEN_DIR, "executor_pb2.py")
    src_grpc = os.path.join(GEN_DIR, "executor_pb2_grpc.py")
    dst_dir = os.path.join(GEN_DIR, "exec")
    if os.path.exists(src_pb):
        shutil.move(src_pb, os.path.join(dst_dir, "executor_pb2.py"))
    if os.path.exists(src_grpc):
        shutil.move(src_grpc, os.path.join(dst_dir, "executor_pb2_grpc.py"))
    for p in [GEN_DIR, os.path.join(GEN_DIR, "exec")]:
        initp = os.path.join(p, "__init__.py")
        if not os.path.exists(initp):
            with open(initp, "w") as f:
                f.write("")


def main():
    load_dotenv(os.path.join(ROOT, ".env"))
    ensure_python_stubs()
    sys.path.insert(0, os.path.join(GEN_DIR, "exec"))
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
