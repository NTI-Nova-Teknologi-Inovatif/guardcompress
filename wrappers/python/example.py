"""Contoh Django/Flask: guardcompress.process(upload_path)"""
from guardcompress import process, BlockedError

def handle_upload(tmp_path):
    try:
        r = process(tmp_path, {"max_mb": 500, "video_crf": 28})
        print("bersih:", r["path"], r["report"].get("new_bytes"))
        # TODO: upload r["path"] ke S3, hapus tmp
        return r
    except BlockedError as e:
        print("diblokir:", e, e.report)
        raise
