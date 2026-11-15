"""Shared helper for generating and locating Python gRPC/protobuf stubs."""

from __future__ import annotations

import importlib
import logging
import os
import shutil
import sys
from collections.abc import Sequence

THIS_DIR = os.path.dirname(os.path.abspath(__file__))
ROOT = os.path.abspath(os.path.join(THIS_DIR, ".."))
PROTO_DIR = os.path.join(ROOT, "proto")
GEN_DIR = os.path.join(ROOT, "pygen")

if ROOT not in sys.path:
    sys.path.insert(0, ROOT)


def ensure_sys_path(path: str) -> None:
    """Insert ``path`` at the front of ``sys.path`` if not already present."""
    if path not in sys.path:
        sys.path.insert(0, path)


def _stub_name(proto_file: str) -> str:
    return os.path.basename(proto_file).removesuffix(".proto")


def _proto_filename(proto: str) -> str:
    return proto if proto.endswith(".proto") else f"{proto}.proto"


def _stubs_importable(
    targets: Sequence[tuple[str, str]],
    gen_dir: str,
) -> bool:
    abs_gen_dir = os.path.abspath(gen_dir)
    if abs_gen_dir == GEN_DIR:
        ensure_sys_path(ROOT)
        for _, subdir in targets:
            dst_dir = os.path.join(abs_gen_dir, subdir)
            if os.path.isdir(dst_dir):
                ensure_sys_path(dst_dir)
        try:
            for proto, subdir in targets:
                stem = _stub_name(proto)
                importlib.import_module(f"pygen.{subdir}.{stem}_pb2")
                importlib.import_module(f"pygen.{subdir}.{stem}_pb2_grpc")
        except Exception:  # noqa: BLE001 - trigger regeneration on any stub load/version error
            return False
        else:
            return True

    for proto, subdir in targets:
        stem = _stub_name(proto)
        dst_dir = os.path.join(abs_gen_dir, subdir)
        required = (
            os.path.join(abs_gen_dir, "__init__.py"),
            os.path.join(dst_dir, "__init__.py"),
            os.path.join(dst_dir, f"{stem}_pb2.py"),
            os.path.join(dst_dir, f"{stem}_pb2_grpc.py"),
        )
        if not all(os.path.isfile(p) for p in required):
            return False
    for _, subdir in targets:
        ensure_sys_path(os.path.join(abs_gen_dir, subdir))
    return True


def ensure_python_stubs(
    targets: Sequence[tuple[str, str]],
    *,
    logger: logging.Logger | None = None,
    proto_dir: str = PROTO_DIR,
    gen_dir: str = GEN_DIR,
) -> None:
    """Generate Python protobuf/gRPC stubs if they are not already importable."""
    if _stubs_importable(targets, gen_dir):
        return

    if logger is not None:
        logger.info("Generating Python protobuf stubs...")

    os.makedirs(gen_dir, exist_ok=True)
    for _, subdir in targets:
        os.makedirs(os.path.join(gen_dir, subdir), exist_ok=True)

    try:
        from grpc_tools import protoc
    except ImportError:
        if logger is not None:
            logger.error("grpc_tools not found. Install with: pip install grpcio-tools")
            sys.exit(1)
        raise

    args = [
        "protoc",
        f"-I{proto_dir}",
        f"--python_out={gen_dir}",
        f"--grpc_python_out={gen_dir}",
        *[os.path.join(proto_dir, _proto_filename(proto)) for proto, _ in targets],
    ]

    if protoc.main(args) != 0:
        if logger is not None:
            logger.error("protoc python generation failed")
        else:
            print("protoc python generation failed", file=sys.stderr)
        sys.exit(1)

    for proto, subdir in targets:
        stem = _stub_name(proto)
        dst_dir = os.path.join(gen_dir, subdir)
        for suffix in ("_pb2.py", "_pb2_grpc.py"):
            src = os.path.join(gen_dir, f"{stem}{suffix}")
            if os.path.exists(src):
                shutil.move(src, os.path.join(dst_dir, f"{stem}{suffix}"))

    pkg_dirs = [gen_dir, *(os.path.join(gen_dir, subdir) for _, subdir in targets)]
    for pkg_dir in dict.fromkeys(pkg_dirs):
        init_path = os.path.join(pkg_dir, "__init__.py")
        if not os.path.exists(init_path):
            with open(init_path, "w") as f:
                f.write("")

    for _, subdir in targets:
        ensure_sys_path(os.path.join(gen_dir, subdir))

    if logger is not None:
        logger.info("Protobuf stubs generated successfully")
