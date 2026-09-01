#!/usr/bin/env bash
# Agent Team 流水线验收脚本
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
FAIL=0

check() {
  if eval "$2" >/dev/null 2>&1; then
    echo "  ✓ $1"
  else
    echo "  ✗ $1"
    FAIL=1
  fi
}

echo "==> Agent Team Pipeline Verify"
echo ""
echo "[目录结构]"
check "apps/hub" "test -d apps/hub"
check "apps/web" "test -d apps/web"
check "services/" "test -d services/agent-bridge"
check "brain/wiki" "test -d brain/wiki"
check "team/" "test -f team/charter.md"

echo ""
echo "[环境]"
check "AGENTS.md" "test -f AGENTS.md"
check "README.md" "test -f README.md"
check "团队 Skills" "test -f .cursor/skills/agent-team-orchestration/SKILL.md"
check "Superpowers" "test -d .cursor/skills/superpowers/using-superpowers"
check "mock-hub binary" "test -f services/mock-hub/mock-hub.exe || test -f services/mock-hub/mock-hub"

echo ""
echo "[测试]"
(cd services/mock-hub && go test ./...) && echo "  ✓ mock-hub tests" || { echo "  ✗ mock-hub tests"; FAIL=1; }
(cd services/agent-bridge && go test ./...) && echo "  ✓ agent-bridge tests" || { echo "  ✗ agent-bridge tests"; FAIL=1; }

echo ""
if [ "$FAIL" -eq 0 ]; then
  echo "✅ Pipeline verify passed"
  exit 0
else
  echo "❌ Pipeline verify failed — run: bash scripts/setup-dev.sh"
  exit 1
fi
