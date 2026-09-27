#!/bin/sh
set -eu

skill=skills/property-based-testing/SKILL.md
reference=skills/property-based-testing/references/python-hypothesis.md

require() {
  file=$1
  text=$2
  grep -F -- "$text" "$file" >/dev/null || {
    printf 'Missing %s in %s\n' "$text" "$file" >&2
    exit 1
  }
}

require "$skill" 'pbt begin --task <id>'
require "$skill" 'pbt inspect --task <id> --json'
require "$skill" 'pbt verify --task <id> --json'
require "$skill" 'pbt gate --task <id> --json'
require "$skill" 'Do not use not_applicable for a behavioral change.'
require "$reference" 'https://hypothesis.readthedocs.io/en/latest/'
require "$reference" '@given'

printf 'PASS\n'
