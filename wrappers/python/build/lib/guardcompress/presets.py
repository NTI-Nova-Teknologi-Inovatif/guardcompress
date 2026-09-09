from .client import process


def image(in_path: str, opts: dict | None = None) -> dict:
    return process(in_path, {"allow_ext": ["jpg", "jpeg", "png", "webp", "gif"], **(opts or {})})


def video(in_path: str, opts: dict | None = None) -> dict:
    return process(in_path, {"allow_ext": ["mp4", "mov", "webm", "mkv", "avi"], **(opts or {})})


def audio(in_path: str, opts: dict | None = None) -> dict:
    return process(in_path, {"allow_ext": ["mp3", "wav", "ogg", "oga", "m4a", "flac"], **(opts or {})})
