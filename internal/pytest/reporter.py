import html
import json
import os

_NODE_RESULTS = os.environ["PBT_NODE_RESULTS"]
_JUNIT_PATH = os.environ["PBT_JUNIT_PATH"]
_LIMIT = int(os.environ["PBT_RUN_ARTIFACT_LIMIT"])
_LIMIT_MARKER = os.environ["PBT_ARTIFACT_LIMIT_MARKER"]
_IS_XDIST_WORKER = "PYTEST_XDIST_WORKER" in os.environ
_JUNIT_CLOSING_TAGS = "</testsuite></testsuites>"
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
    outcome = {
        "passed": "",
        "skipped": "<skipped/>",
        "failed": "<failure/>",
    }.get(report.outcome, "<error/>")
    _write(
        _JUNIT_PATH,
        f'<testcase name="{html.escape(report.nodeid, quote=True)}">{outcome}</testcase>',
        len(_JUNIT_CLOSING_TAGS),
    )


def pytest_sessionstart(session):
    if _IS_XDIST_WORKER:
        return
    _create(_NODE_RESULTS)
    _create(_JUNIT_PATH)
    _write(_JUNIT_PATH, '<?xml version="1.0" encoding="UTF-8"?><testsuites><testsuite name="pytest">')


def pytest_runtest_logreport(report):
    if _IS_XDIST_WORKER:
        return
    if report.when == "setup" and report.skipped:
        _write_report(report)
    elif report.when == "call":
        _write_report(report)


def pytest_sessionfinish(session, exitstatus):
    if _IS_XDIST_WORKER:
        return
    _write(_JUNIT_PATH, _JUNIT_CLOSING_TAGS)
