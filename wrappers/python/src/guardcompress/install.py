"""Installer: download core binary + ffmpeg static dari GitHub Releases
ke ~/.cache/guardcompress, verifikasi SHA256 via CHECKSUMS.txt.
Usage: python -m guardcompress.install [version]
Idempotent: file yang hash-nya sudah cocok dilewati.
"""
import hashlib
import os
import platform
import sys
import urllib.request
from pathlib import Path


def _plat_arch():
    plat = {"Windows": "windows", "Linux": "linux", "Darwin": "darwin"}.get(platform.system(), "linux")
    arch = "arm64" if platform.machine().lower() in ("arm64", "aarch64") else "amd64"
    return plat, arch


def _fetch(url: str, timeout=60) -> bytes | None:
    try:
        with urllib.request.urlopen(url, timeout=timeout) as r:
            return r.read()
    except Exception as e:
        print(f"fetch gagal {url}: {e}", file=sys.stderr)
        return None


def main(version: str | None = None) -> int:
    version = version or os.environ.get("GUARDCOMPRESS_VERSION", "v0.1.3")
    base = os.environ.get(
        "GUARDCOMPRESS_RELEASE_BASE",
        "https://github.com/NTI-Nova-Teknologi-Inovatif/guardcompress/releases/download",
    )
    plat, arch = _plat_arch()
    ext = ".exe" if plat == "windows" else ""
    files = [f"guardcompress-{plat}-{arch}{ext}", f"ffmpeg-{plat}-{arch}{ext}"]
    dest_dir = Path.home() / ".cache" / "guardcompress"
    dest_dir.mkdir(parents=True, exist_ok=True)

    raw = _fetch(f"{base}/{version}/CHECKSUMS.txt")
    if raw is None:
        print("CHECKSUMS tak bisa diunduh. Set GUARDCOMPRESS_BIN manual.", file=sys.stderr)
        return 0  # jangan gagalkan pip install
    want: dict[str, str] = {}
    for line in raw.decode().splitlines():
        parts = line.strip().split()
        if len(parts) == 2 and len(parts[0]) == 64:
            want[parts[1]] = parts[0]

    fail = 0
    for f in files:
        is_ffmpeg = f.startswith("ffmpeg-")
        dest = dest_dir / f
        if f not in want:
            print(f"{f} belum dirilis, lewati (mode guard-only)." if is_ffmpeg
                  else f"checksum {f} tidak ada di CHECKSUMS.txt")
            if not is_ffmpeg:
                fail = 1
            continue
        if dest.is_file() and hashlib.sha256(dest.read_bytes()).hexdigest() == want[f]:
            print(f"{f} sudah ada & cocok, lewati.")
            continue
        print(f"Downloading {f} ...")
        buf = _fetch(f"{base}/{version}/{f}")
        if buf is None or hashlib.sha256(buf).hexdigest() != want[f]:
            print(f"{f} gagal (download/checksum tidak cocok), dibuang.", file=sys.stderr)
            if not is_ffmpeg:
                fail = 1
            continue
        dest.write_bytes(buf)
        if plat != "windows":
            dest.chmod(0o755)
        print(f"Installed: {dest}")
    return fail


if __name__ == "__main__":
    sys.exit(main(sys.argv[1] if len(sys.argv) > 1 else None))
