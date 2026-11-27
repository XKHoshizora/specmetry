# Local fixture cases for the Windows PATH step of c9-evidence.yml (Review
# 157 N-157-1; review-resolution §208/§209). Not a runner execution.
#
# usage: pwsh -NoProfile -File pwsh-cases.ps1 -W <workdir> -Labels new,dcc01bd
# <workdir>\runs\<label>.step4.ps1 is the run block extracted by
# yaml-check.mjs. Each case runs it as GitHub runs a pwsh step: the script
# is preceded by $ErrorActionPreference = 'stop' and followed by the exit
# code check. "go.exe" is a copy of a Windows system program and is never
# run; only its bytes are hashed.
#
# The real cases use a real junction and a plain directory. A real
# junction always stores a fully qualified target, so the other target
# strings are given through a function named Get-Item defined before the
# step body; in PowerShell a function takes precedence over the cmdlet of
# the same name. The double only answers LinkType and Target.
#
# Display only: the output is UTF-8 without color codes, and PowerShell's
# and .NET's messages are in English (en-US), as on the runner. The child
# wrapper sets the same before the step body; this changes no behaviour of
# the step.
param([Parameter(Mandatory)][string]$W, [Parameter(Mandatory)][string]$Labels)
$ErrorActionPreference = 'stop'
$utf8 = [System.Text.UTF8Encoding]::new($false)
[Console]::OutputEncoding = $utf8
$OutputEncoding = $utf8
$PSStyle.OutputRendering = 'PlainText'
$display = "[Console]::OutputEncoding = [System.Text.UTF8Encoding]::new(`$false); `$PSStyle.OutputRendering = 'PlainText'; [System.Threading.Thread]::CurrentThread.CurrentCulture = 'en-US'; [System.Threading.Thread]::CurrentThread.CurrentUICulture = 'en-US'`n"

$fx = Join-Path $W 'pfx'
if (Test-Path -LiteralPath $fx) { Remove-Item -LiteralPath $fx -Recurse -Force }
foreach ($d in 'real\x64\bin', 'plain\x64\bin', 'other\x64\bin', 'nobin\x64', 'link', 'relative\bin') {
  New-Item -ItemType Directory -Force (Join-Path $fx $d) | Out-Null
}
Copy-Item "$env:SystemRoot\System32\whoami.exe" (Join-Path $fx 'real\x64\bin\go.exe')
# For the case where a drive-relative target resolves against the current
# directory: <fx>\relative\bin\go.exe has the bytes of the go.exe on PATH.
Copy-Item "$env:SystemRoot\System32\whoami.exe" (Join-Path $fx 'relative\bin\go.exe')
Copy-Item "$env:SystemRoot\System32\whoami.exe" (Join-Path $fx 'plain\x64\bin\go.exe')
Copy-Item "$env:SystemRoot\System32\hostname.exe" (Join-Path $fx 'other\x64\bin\go.exe')
New-Item -ItemType Junction -Path (Join-Path $fx 'link\x64') -Target (Join-Path $fx 'real\x64') | Out-Null
$real = Join-Path $fx 'real\x64'

function Show([string]$s) { $s.Replace($W, '<W>') }

# One case: write the wrapper, run it in a child pwsh with the given PATH
# head, print the command, the raw output, the exit status and what was
# written to GITHUB_PATH.
function RunCase([string]$label, [string]$name, [string]$pathHead, [string]$double, [string]$cwd = '') {
  $body = Get-Content -Raw -LiteralPath (Join-Path $W "runs\$label.step4.ps1")
  $pre = $display + "`$ErrorActionPreference = 'stop'`n"
  if ($double) {
    $pre += "function Get-Item { param([Parameter(Position = 0)][string]`$Path, [string]`$LiteralPath, [switch]`$Force) [pscustomobject]@{ LinkType = 'Junction'; Target = $double } }`n"
  }
  $wrapper = Join-Path $W "pcase-$label-$name.ps1"
  Set-Content -LiteralPath $wrapper -Value ($pre + $body + "`nif ((Test-Path -LiteralPath variable:\LASTEXITCODE)) { exit `$LASTEXITCODE }`n") -NoNewline
  $gp = Join-Path $W "GITHUB_PATH-$label-$name.txt"
  Set-Content -LiteralPath $gp -Value '' -NoNewline
  "## $label PATH step: $name"
  "`$ pwsh -NoProfile -File pcase-$label-$name.ps1   # PATH head: $(Show $pathHead)" + $(if ($double) { "; Get-Item double: Target = $(Show $double)" } else { '' }) + $(if ($cwd) { "; current directory: $(Show $cwd)" } else { '' })
  $saved = $env:PATH
  $env:PATH = "$pathHead;$env:PATH"
  $env:GITHUB_PATH = $gp
  $eap = $ErrorActionPreference
  $ErrorActionPreference = 'continue'
  if ($cwd) { Push-Location -LiteralPath $cwd }
  try {
    $out = & pwsh -NoProfile -File $wrapper 2>&1
    $code = $LASTEXITCODE
  } finally {
    if ($cwd) { Pop-Location }
    $ErrorActionPreference = $eap
    $env:PATH = $saved
  }
  $out | ForEach-Object { Show (($_ | Out-String).TrimEnd() -replace "`e\[[0-9;]*m", '') }
  "exit $code"
  "GITHUB_PATH: [$(Show (Get-Content -Raw -LiteralPath $gp))]"
  ''
}

"pwsh: $($PSVersionTable.PSVersion)"
''
$linkBin = Join-Path $fx 'link\x64\bin'
$plainBin = Join-Path $fx 'plain\x64\bin'
$q = { param($s) "'" + $s.Replace("'", "''") + "'" }
foreach ($label in $Labels.Split(',')) {
  RunCase $label 'real-junction' $linkBin ''
  RunCase $label 'no-junction' $plainBin ''
  RunCase $label 'target-drive-path' $plainBin (& $q $real)
  RunCase $label 'target-device-path' $plainBin (& $q "\\?\$real")
  RunCase $label 'target-null' $plainBin '$null'
  RunCase $label 'target-empty' $plainBin "''"
  RunCase $label 'target-drive-relative' $plainBin "'C:relative'"
  RunCase $label 'target-drive-relative-resolvable' $plainBin "'$($fx.Substring(0, 2))relative'" $fx
  RunCase $label 'target-root-relative' $plainBin "'\relative'"
  RunCase $label 'target-relative' $plainBin "'relative'"
  RunCase $label 'two-targets' $plainBin "@($(& $q $real), $(& $q $real))"
  RunCase $label 'target-other-bytes' $plainBin (& $q (Join-Path $fx 'other\x64'))
  RunCase $label 'target-without-go' $plainBin (& $q (Join-Path $fx 'nobin\x64'))
}
