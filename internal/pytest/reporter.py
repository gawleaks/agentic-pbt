import json
import os

_NODE_RESULTS = os.environ["PBT_NODE_RESULTS"]
_LIMIT = int(os.environ["PBT_RUN_ARTIFACT_LIMIT"])
_LIMIT_MARKER = os.environ["PBT_ARTIFACT_LIMIT_MARKER"]
_IS_XDIST_WORKER = "PYTEST_XDIST_WORKER" in os.environ
_SIZES = {}


def _mark_limit_exceeded():
    with open(_LIMIT_MARKER, "w", encoding="utf-8"):
        pass


def _write(path, content, reserve=0):
    data = content.encode("utf-8")
    size = _SIZES.get(path, 0)
    if size + len(data) + reserve > _LIMIT:
        _mark_limit_exceeded()
        return
    with open(path, "ab") as output:
        output.write(data)
    _SIZES[path] = size + len(data)


def _create(path):
    with open(path, "ab"):
        pass
    _SIZES[path] = 0


def _write_report(report):
    _write(
        _NODE_RESULTS,
        json.dumps({"nodeid": report.nodeid, "outcome": report.outcome}) + "\n",
    )


def pytest_sessionstart(session):
    if _IS_XDIST_WORKER:
        return
    _create(_NODE_RESULTS)


def pytest_runtest_logreport(report):
    if _IS_XDIST_WORKER:
        return
    if report.when == "setup" and report.skipped:
        _write_report(report)
    elif report.when == "call":
        _write_report(report)
