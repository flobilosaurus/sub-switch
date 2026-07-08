#!/usr/bin/env bash
set -euo pipefail

tag="${GITHUB_REF_NAME:-$(git describe --tags --abbrev=0 2>/dev/null || git rev-parse --short HEAD)}"
previous_tag="$(git describe --tags --abbrev=0 "${tag}^" 2>/dev/null || true)"

if [[ -n "${previous_tag}" ]]; then
  range="${previous_tag}..${tag}"
else
  range="${tag}"
fi

printf '# %s\n\n' "${tag}"

if [[ -n "${previous_tag}" ]]; then
  printf 'Changes since `%s`.\n\n' "${previous_tag}"
else
  printf 'Initial release.\n\n'
fi

commits="$(git log --no-merges --format='%s%x1f%b%x1e' "${range}")"

print_section() {
  local title="$1"
  local regex="$2"
  local include_breaking_body="${3:-false}"
  local output=""

  output="$({
    printf '%s' "${commits}" | awk -v RS='\036' -v FS='\037' -v regex="${regex}" -v include_breaking_body="${include_breaking_body}" '
      function trim(s) {
        sub(/^[[:space:]]+/, "", s)
        sub(/[[:space:]]+$/, "", s)
        return s
      }
      {
        subject = trim($1)
        body = $2
        if (subject == "") next
        if (subject ~ regex || (include_breaking_body == "true" && body ~ /(^|\n)BREAKING[ -]CHANGE:/)) {
          description = subject
          if (match(subject, /^[a-zA-Z]+(\([^)]+\))?!?:[[:space:]]*/)) {
            description = substr(subject, RLENGTH + 1)
          }
          print "- " description
        }
      }
    '
  } | sort -u)"

  if [[ -n "${output}" ]]; then
    printf '## %s\n\n%s\n\n' "${title}" "${output}"
  fi
}

print_section 'Breaking Changes' '^[a-zA-Z]+(\([^)]+\))?!:' true
print_section 'Features' '^feat(\([^)]+\))?:'
print_section 'Fixes' '^fix(\([^)]+\))?:'
print_section 'Documentation' '^docs(\([^)]+\))?:'
print_section 'Maintenance' '^(build|ci|chore|perf|refactor|revert|style|test)(\([^)]+\))?:'

uncategorized="$({
  printf '%s' "${commits}" | awk -v RS='\036' -v FS='\037' '
    function trim(s) {
      sub(/^[[:space:]]+/, "", s)
      sub(/[[:space:]]+$/, "", s)
      return s
    }
    {
      subject = trim($1)
      if (subject == "") next
      if (subject !~ /^[a-zA-Z]+(\([^)]+\))?!?:/) print "- " subject
    }
  '
} | sort -u)"

if [[ -n "${uncategorized}" ]]; then
  printf '## Other Changes\n\n%s\n\n' "${uncategorized}"
fi

if [[ -n "${previous_tag}" ]]; then
  printf '**Full Changelog**: https://github.com/%s/compare/%s...%s\n' "${GITHUB_REPOSITORY:-florian-balling/sub-switch}" "${previous_tag}" "${tag}"
fi
