# Local fixture cases for the bash steps of c9-evidence.yml (Review 157
# N-157-2; review-resolution §208/§209). Not a runner execution.
#
# usage: bash bash-cases.sh <workdir> <label>...
# <workdir>/runs/<label>.step<N>.sh are the run blocks extracted by
# yaml-check.mjs (step 2: identity, step 3: LE-4, step 5: Collect). Each
# block runs as GitHub runs a bash step: bash --noprofile --norc -eo
# pipefail <file>. No real Go runs: "go" is a fixture script that prints a
# fixed text. Git runs with the user's and the system's config disabled.
#
# MSYS_NO_PATHCONV=1: this machine's MSYS2 runtime rewrites an argument
# such as HEAD:.github/workflows/c9-evidence.yml as a path list. The
# windows-latest runner's Git Bash passed that argument unchanged in phase
# 2 (docs/evidence/wp2.5-c9-phase2/ci-43906849891/job-windows.log, line
# 218), so the fixtures pass arguments unchanged too, and the harness
# gives git -C a Windows-form path.
set -u
export MSYS_NO_PATHCONV=1
W=$1
shift
REALGIT=$(command -v git)
g() { local d=$1; shift; "$REALGIT" -C "$(cygpath -m "$d")" "$@"; }
SYS="$(dirname "$REALGIT"):/usr/bin:/bin"
export GIT_CONFIG_GLOBAL=/dev/null GIT_CONFIG_NOSYSTEM=1
export GIT_AUTHOR_NAME=fixture GIT_AUTHOR_EMAIL=fixture@example.invalid
export GIT_COMMITTER_NAME=fixture GIT_COMMITTER_EMAIL=fixture@example.invalid
export GIT_AUTHOR_DATE="2026-11-28T00:00:00Z" GIT_COMMITTER_DATE="2026-11-28T00:00:00Z"

# The fixture "go": no Go toolchain is involved.
FAKEGO=$W/fakego
mkdir -p "$FAKEGO"
cat > "$FAKEGO/go" <<'EOS'
#!/bin/bash
case "$1" in
  version) echo "go version go1.27.3 fixture/fake" ;;
  mod) echo "fixture: go $*"; if [ "${FAKE_GO_DIRTY:-}" = 1 ]; then echo x > dirtied-by-download; fi ;;
  test) echo "fixture: go $*" ;;
  *) echo "fixture go: unexpected $*" >&2; exit 2 ;;
esac
EOS
chmod +x "$FAKEGO/go"

# A git wrapper that fails or answers empty for one chosen read; every
# other call goes to the real git.
FAKEGIT=$W/fakegit
mkdir -p "$FAKEGIT"
cat > "$FAKEGIT/git" <<EOS
#!/bin/bash
if [ -n "\${FAKE_GIT_FAIL:-}" ] && [ "\$*" = "\$FAKE_GIT_FAIL" ]; then echo "fixture git: failing \$*" >&2; exit 128; fi
if [ -n "\${FAKE_GIT_EMPTY:-}" ] && [ "\$*" = "\$FAKE_GIT_EMPTY" ]; then exit 0; fi
exec "$REALGIT" "\$@"
EOS
chmod +x "$FAKEGIT/git"

# fixture <dir> <layout>: a git repository with one commit. layout "full"
# has go.mod, go.sum, prototype/c9evidence/ and the workflow file;
# "no-subtree" lacks prototype/c9evidence/; "no-gomod" lacks go.mod.
fixture() {
  local d=$1 layout=$2
  rm -rf "$d"
  mkdir -p "$d/.github/workflows" "$d/prototype/c9evidence"
  echo "module example.invalid/fixture" > "$d/go.mod"
  echo "" > "$d/go.sum"
  echo "module example.invalid/fixture/c9evidence" > "$d/prototype/c9evidence/go.mod"
  echo "name: fixture" > "$d/.github/workflows/c9-evidence.yml"
  case "$layout" in
    no-subtree) rm -rf "$d/prototype" ;;
    no-gomod) rm -f "$d/go.mod" ;;
  esac
  g "$d" -c init.defaultBranch=main init -q
  g "$d" add -A
  g "$d" commit -q -m fixture
}

# run_case <label> <step> <name> <layout> <dir-suffix> <env...>
# Runs one step file in a fresh fixture and prints the command, the raw
# output and the exit status.
run_case() {
  local label=$1 step=$2 name=$3 layout=$4 sub=$5
  shift 5
  local file=$W/runs/$label.step$step.sh
  local d=$W/fx/$label-step$step-$name
  fixture "$d" "$layout"
  local sha
  sha=$(g "$d" rev-parse HEAD)
  local path="$FAKEGIT:$FAKEGO:$SYS"
  local extra=()
  for kv in "$@"; do
    case "$kv" in
      NOGO) path="$FAKEGIT:$SYS" ;;
      DIRTY) echo junk > "$d/untracked" ;;
      SHA=*) sha=${kv#SHA=} ;;
      *) extra+=("$kv") ;;
    esac
  done
  local shown=$sha
  [ "$sha" = "$(g "$d" rev-parse HEAD)" ] && shown="<fixture HEAD>"
  local gopath="fixture go on PATH"
  [ "$path" = "$FAKEGIT:$SYS" ] && gopath="no go on PATH"
  echo "## $label step $step: $name"
  echo "\$ (cd <fixture>/$sub && env GITHUB_SHA=$shown ${extra[*]} bash --noprofile --norc -eo pipefail $label.step$step.sh)  # $gopath"
  (
    cd "$d/$sub" || exit 99
    env PATH="$path" GITHUB_SHA="$sha" GITHUB_REF=refs/heads/fixture \
      GITHUB_WORKFLOW_REF=fixture/c9-evidence.yml@refs/heads/fixture GITHUB_WORKFLOW_SHA="$sha" \
      "${extra[@]}" bash --noprofile --norc -eo pipefail "$file"
  ) 2>&1
  echo "exit $?"
  echo
}

echo "bash: $BASH_VERSION"
echo "git: $("$REALGIT" --version)"
echo
for label in "$@"; do
  # Identity step.
  run_case "$label" 2 ok full .
  run_case "$label" 2 sha-mismatch full . SHA=0000000000000000000000000000000000000000
  run_case "$label" 2 dirty full . DIRTY
  run_case "$label" 2 no-subtree no-subtree .
  run_case "$label" 2 status-read-fails full . "FAKE_GIT_FAIL=status --porcelain"
  run_case "$label" 2 tree-read-empty full . "FAKE_GIT_EMPTY=rev-parse HEAD^{tree}"
  run_case "$label" 2 blob-read-fails full . "FAKE_GIT_FAIL=rev-parse HEAD:.github/workflows/c9-evidence.yml"
  run_case "$label" 2 no-go full . NOGO
  # LE-4.
  run_case "$label" 3 ok full .
  run_case "$label" 3 no-gomod no-gomod .
  run_case "$label" 3 gosum-read-empty full . "FAKE_GIT_EMPTY=rev-parse HEAD:go.sum"
  run_case "$label" 3 status-read-fails full . "FAKE_GIT_FAIL=status --porcelain"
  run_case "$label" 3 download-dirties full . FAKE_GO_DIRTY=1
  # Collect (working-directory prototype/c9evidence).
  run_case "$label" 5 ok full prototype/c9evidence
  run_case "$label" 5 no-go full prototype/c9evidence NOGO
done
