package pytest

import _ "embed"

// Reporter is the bundled pytest node-result plugin.
//
//go:embed reporter.py
var Reporter string
