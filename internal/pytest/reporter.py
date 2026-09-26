import json
import os

_PATH = os.environ["PBT_NODE_RESULTS"]


def _write(report):
    with open(_PATH, "a", encoding="utf-8") as output:
        output.write(json.dumps({"nodeid": report.nodeid, "outcome": report.outcome}) + "\n")


def pytest_runtest_logreport(report):
    if report.when == "setup" and report.skipped:
        _write(report)
    elif report.when == "call":
        _write(report)
