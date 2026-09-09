import concurrent.futures

from .client import process
from .errors import BlockedError


def batch(items, opts: dict | None = None) -> dict | list:
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
        vals = [_one(it) for _, it in entries]
    if as_dict:
        return {k: v for (k, _), v in zip(entries, vals)}
    return vals
