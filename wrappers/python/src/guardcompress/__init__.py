"""Thin wrapper: subprocess -> guardcompress binary -> report.json"""
import json, os, platform, shutil, subprocess, tempfile
from pathlib import Path

class BlockedError(Exception):
    def __init__(self, msg, report=None):
        super().__init__(msg)
        self.report = report or {}


class BusyError(Exception):
    """Server penuh (backpressure) -> balas HTTP 429 + retry, bukan 422."""

    def __init__(self, msg, report=None):
        super().__init__(msg)
        self.report = report or {}

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
    """Return {'path': out_path, 'report': {...}}. Raise BlockedError jika exit 2."""
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
        shutil.rmtree(out, ignore_errors=True)  # file kotor: buang output
        raise BlockedError("blocked: " + str(report.get("reason")), report)
    if r.returncode != 0:
        shutil.rmtree(out, ignore_errors=True)
        if isinstance(report.get("details"), dict) and report["details"].get("busy"):
            raise BusyError(str(report.get("reason") or "server busy"), report)
        raise RuntimeError("guardcompress failed: " + str(report.get("reason", r.stderr)))
    # Gagal cepat di batas: jangan kembalikan path None yang meledak belakangan.
    if not report.get("out_path"):
        shutil.rmtree(out, ignore_errors=True)
        raise RuntimeError("guardcompress: out_path hilang dari report")
    return {"path": report.get("out_path"), "report": report}


# Preset per jenis (1 sistem di belakangnya, opts user menang bila menimpa).
def image(in_path: str, opts: dict | None = None) -> dict:
    return process(in_path, {"allow_ext": ["jpg", "jpeg", "png", "webp", "gif"], **(opts or {})})


def video(in_path: str, opts: dict | None = None) -> dict:
    return process(in_path, {"allow_ext": ["mp4", "mov", "webm", "mkv", "avi"], **(opts or {})})


def audio(in_path: str, opts: dict | None = None) -> dict:
    return process(in_path, {"allow_ext": ["mp3", "wav", "ogg", "oga", "m4a", "flac"], **(opts or {})})


def batch(items, opts: dict | None = None) -> dict | list:
    """Batch multi-input beda jenis sekaligus.
    items: {"avatar": path, "video": path} atau [{"path":..., "opts":...}].
    File ditolak terkumpul (ok False); error teknis tetap raise.
    Paralel bila opts["jobs"] > 1 (ThreadPool, default 1 = sekuensial).
    Urutan hasil selalu sama dengan urutan input.
    """
    import concurrent.futures

    opts = opts or {}
    jobs = opts.get("jobs", 1)
    try:
        jobs = int(jobs)
    except (TypeError, ValueError):
        jobs = 1
    if isinstance(items, dict):
        entries = [(k, ({"path": v} if isinstance(v, str) else v)) for k, v in items.items()]
        as_dict = True
    else:
        entries = [(i, ({"path": v} if isinstance(v, str) else v)) for i, v in enumerate(items)]
        as_dict = False

    def _one(it):
        merged = {k: v for k, v in opts.items() if k != "jobs"}
        merged.update(it.get("opts") or {})
        try:
            r = process(it["path"], merged)
            return {"ok": True, **r}
        except BlockedError as e:
            return {"ok": False, "blocked": True, "reason": str(e), "report": e.report}

    if jobs > 1:
        with concurrent.futures.ThreadPoolExecutor(max_workers=jobs) as ex:
            vals = list(ex.map(lambda kv: _one(kv[1]), entries))
    else:
        # Sekuensial: error teknis raise langsung (fail-fast).
        vals = [_one(it) for _, it in entries]
    if as_dict:
        return {k: v for (k, _), v in zip(entries, vals)}
    return vals
