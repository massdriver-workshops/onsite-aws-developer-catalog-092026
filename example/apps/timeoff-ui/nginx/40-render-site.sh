#!/bin/sh
# Copies the static site under BASE_PATH so nginx can serve it from a plain root,
# and renders the runtime config the browser reads.
set -eu

base="$(printf '%s' "${BASE_PATH:-}" | sed 's#^/*##; s#/*$##')"
site="/tmp/www"
[ -n "$base" ] && site="/tmp/www/$base"

mkdir -p "$site"
cp -R /app/. "$site/"

prefix=""
[ -n "$base" ] && prefix="/$base"

cat > "$site/config.json" <<JSON
{
  "namespace": "${NAMESPACE:-local}",
  "basePath": "${prefix}",
  "timeoffApi": "${TIMEOFF_API_URL:-${prefix}/api/timeoff}",
  "payrollApi": "${PAYROLL_API_URL:-${prefix}/api/payroll}",
  "uiVersion": "${UI_VERSION:-dev}"
}
JSON
echo "timeoff-ui serving ${prefix:-/} -> timeoff ${TIMEOFF_API_URL:-${prefix}/api/timeoff}, payroll ${PAYROLL_API_URL:-${prefix}/api/payroll}"
