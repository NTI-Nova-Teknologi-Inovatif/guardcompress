class BlockedError(Exception):
    def __init__(self, msg, report=None):
        super().__init__(msg)
        self.report = report or {}


class BusyError(Exception):
    def __init__(self, msg, report=None):
        super().__init__(msg)
        self.report = report or {}
