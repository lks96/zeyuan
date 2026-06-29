param(
  [string]$Message = "",
  [string]$Proxy = "http://127.0.0.1:7897"
)

$ErrorActionPreference = "Stop"

$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$Root = Split-Path -Parent $ScriptDir

Push-Location $Root
try {
  $status = git status --porcelain

  if ($status) {
    git add -A

    if ([string]::IsNullOrWhiteSpace($Message)) {
      $Message = "Update workspace changes"
    }

    git commit -m $Message
  } else {
    Write-Host "No local changes to commit."
  }

  git -c "http.proxy=$Proxy" -c "https.proxy=$Proxy" push
} finally {
  Pop-Location
}
