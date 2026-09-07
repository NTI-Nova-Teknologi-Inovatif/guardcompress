"""Contoh Celery task (Python): GuardCompress di background.
ANTI-DOWN: jalankan worker dengan concurrency terbatas, misal:
  celery -A examples.celery-task worker --concurrency=2
(lihat `doctor` -> recommended_jobs; ffmpeg_threads default 2/job)
"""
from celery import Celery
from guardcompress import process, BlockedError
import os

app = Celery("media", broker="redis://localhost:6379/0")

@app.task(bind=True, max_retries=3)
def compress_upload(self, tmp_path: str, user_id: int):
    try:
        r = process(tmp_path, {"max_mb": 500, "video_crf": 28, "timeoutSec": 600})
        # TODO: upload r["path"] ke S3, simpan r["report"] ke DB
        return {"stored": r["path"]}
    except BlockedError as e:
        return {"blocked": str(e)}  # jangan retry untuk file kotor
    except Exception as e:
        raise self.retry(exc=e, countdown=30)
    finally:
        try:
            os.remove(tmp_path)
        except OSError:
            pass
