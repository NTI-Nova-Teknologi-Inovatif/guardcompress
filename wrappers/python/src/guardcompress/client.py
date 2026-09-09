import json
import os
import platform
import shutil
import subprocess
import tempfile
from pathlib import Path

from .errors import BlockedError, BusyError


def cleanup(dir_path: str) -> None:
    shutil.rmtree(dir_path, ignore_errors=True)


def _resolve_binary() -> str:
    if os.environ.get("GUARDCOMPRESS_BIN"):
        return os.environ["GUARDCOMPRESS_BIN"]
    plat = {"Windows": "windows", "Linux": "linux", "Darwin": "darwin"}.get(platform.system(), "linux")
    arch = "arm64" if platform.machine().lower() in ("arm64", "aarch64") else "amd64"
    ext = ".exe" if plat == "windows" else ""
    name = f"guardcompress-{plat}-{arch}{ext}"
    cands = [
        Path.home() / ".cache" / "guardcompress" / name,
        Path(__file__).resolve().parents[3] / "core" / "bin" / name,
    ]
    for p in cands:
        if p.is_file():
            return str(p)
    raise FileNotFoundError(f"binary not found ({name}). Set GUARDCOMPRESS_BIN.")


def process(in_path: str, opts: dict | None = None) -> dict:
    b = _resolve_binary()
    out = tempfile.mkdtemp(prefix="gc-")
    cfg = json.dumps(opts or {})
    try:
        r = subprocess.run([b, "check", "--in", in_path, "--out-dir", out,
                            "--config", cfg, "--json"],
                           capture_output=True, text=True, timeout=(opts or {}).get("timeoutSec", 120))
    except Exception:
        shutil.rmtree(out, ignore_errors=True)
        raise
    line = (r.stdout or "").strip().splitlines()
    try:
        report = json.loads(line[-1]) if line else {"reason": r.stderr}
    except json.JSONDecodeError:
        report = {"reason": (r.stdout or "") + (r.stderr or "")}
    if r.returncode == 2:
        shutil.rmtree(out, ignore_errors=True)
        raise BlockedError("blocked: " + str(report.get("reason")), report)
    if r.returncode != 0:
        shutil.rmtree(out, ignore_errors=True)
        if isinstance(report.get("details"), dict) and report["details"].get("busy"):
            raise BusyError(str(report.get("reason") or "server busy"), report)
        raise RuntimeError("guardcompress failed: " + str(report.get("reason", r.stderr)))
    if not report.get("out_path"):
        shutil.rmtree(out, ignore_errors=True)
        raise RuntimeError("guardcompress: out_path hilang dari report")
    return {"path": report.get("out_path"), "report": report}
