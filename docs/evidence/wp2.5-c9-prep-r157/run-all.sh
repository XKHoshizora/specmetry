# Driver of the local checks of the Review 157 round (review-resolution
# §208/§209). It runs no Go, no tool module, no formal entry and no
# collector, and needs no network.
#
# usage (Git Bash on Windows): bash run-all.sh <repo> <workdir>
# It extracts c9-evidence.yml from master c34baf8, from dcc01bd (the
# reviewed HEAD) and from HEAD, and writes into outputs/:
#   yaml-check.out  prettier's YAML parser: settings and steps compared
#   syntax.out      bash -n and the PowerShell parser on every run block
#   bash-cases.out  the identity, LE-4 and Collect steps on fixtures
#   pwsh-cases.out  the Windows PATH step on fixtures
# Every output keeps the commands, the raw output and the exit status; the
# work directory and the user's home are replaced by <W> and <HOME>.
set -uo pipefail
export MSYS_NO_PATHCONV=1
R=$1
W=$2
RM=$(cygpath -m "$R")
E=$R/docs/evidence/wp2.5-c9-prep-r157
O=$E/outputs
rm -rf "$W"
mkdir -p "$W/runs" "$O"
WW=$(cygpath -w "$W")
WM=$(cygpath -m "$W")
HW=$(cygpath -w "$HOME")
HM=$(cygpath -m "$HOME")

scrub() {
  sed -e "s#$(printf '%s' "$WW" | sed 's/[\\.]/\\&/g')#<W>#g" \
      -e "s#$(printf '%s' "$WM" | sed 's/[.]/\\&/g')#<W>#g" \
      -e "s#$(printf '%s' "$W" | sed 's/[.]/\\&/g')#<W>#g" \
      -e "s#$(printf '%s' "$HW" | sed 's/[\\.]/\\&/g')#<HOME>#g" \
      -e "s#$(printf '%s' "$HM" | sed 's/[.]/\\&/g')#<HOME>#g" \
      -e "s#$(printf '%s' "$HOME" | sed 's/[.]/\\&/g')#<HOME>#g"
}

for v in master=c34baf8 dcc01bd=dcc01bd new=HEAD; do
  git -C "$RM" show "${v#*=}:.github/workflows/c9-evidence.yml" > "$W/${v%%=*}.yml"
done
PLUGIN=$(cygpath -w "$R/web/node_modules/prettier/plugins/yaml.mjs")

{
  echo "repo HEAD $(git -C "$RM" rev-parse HEAD)"
  for v in c34baf8 dcc01bd HEAD; do
    echo "c9-evidence.yml blob at $v: $(git -C "$RM" rev-parse "$v:.github/workflows/c9-evidence.yml")"
  done
  echo "node $(node --version); prettier $(sed -n 's/.*"version": "\(.*\)".*/\1/p' "$R/web/node_modules/prettier/package.json" | head -1)"
  echo "\$ node yaml-check.mjs <prettier yaml.mjs> <W>/runs master=master.yml dcc01bd=dcc01bd.yml new=new.yml"
  (cd "$W" && node "$(cygpath -w "$E/yaml-check.mjs")" "$PLUGIN" "$WW\\runs" master=master.yml dcc01bd=dcc01bd.yml new=new.yml) 2>&1
  echo "exit $?"
} | scrub > "$O/yaml-check.out"

{
  echo "bash $BASH_VERSION; pwsh $(pwsh -NoProfile -Command '$PSVersionTable.PSVersion.ToString()')"
  for f in "$W"/runs/*.sh; do
    echo "\$ bash -n runs/$(basename "$f")"
    bash -n "$f" 2>&1
    echo "exit $?"
  done
  for f in "$W"/runs/*.ps1; do
    echo "\$ PowerShell Parser.ParseFile runs/$(basename "$f")"
    pwsh -NoProfile -Command "\$t = \$null; \$e = \$null; [void][System.Management.Automation.Language.Parser]::ParseFile('$(cygpath -w "$f")', [ref]\$t, [ref]\$e); if (\$e.Count) { \$e | ForEach-Object { \$_.ToString() }; exit 1 } else { 'no parse errors' }" 2>&1
    echo "exit $?"
  done
} | scrub > "$O/syntax.out"

{
  echo "\$ bash bash-cases.sh <W> new dcc01bd"
  bash "$E/bash-cases.sh" "$W" new dcc01bd 2>&1
  echo "exit $?"
} | scrub > "$O/bash-cases.out"

{
  echo "\$ pwsh -NoProfile -File pwsh-cases.ps1 -W <W> -Labels new,dcc01bd"
  pwsh -NoProfile -File "$(cygpath -w "$E/pwsh-cases.ps1")" -W "$WW" -Labels new,dcc01bd 2>&1
  echo "exit $?"
} | scrub > "$O/pwsh-cases.out"

# Nothing of the user's paths may remain.
if grep -n -i -e "$(basename "$HOME")" "$O"/*.out; then
  echo "a user path remains in the outputs" >&2
  exit 1
fi
echo "outputs written"
