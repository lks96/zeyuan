param(
  [string]$Message = "",
  [string]$Proxy = "http://127.0.0.1:7897"
)

$ErrorActionPreference = "Stop"

$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$Root = Split-Path -Parent $ScriptDir

Push-Location $Root
try {
  if (-not [string]::IsNullOrWhiteSpace($Proxy)) {
    $env:HTTP_PROXY = $Proxy
    $env:HTTPS_PROXY = $Proxy
    $env:ALL_PROXY = $Proxy
    $env:NO_PROXY = "localhost,127.0.0.1"
  }

  Write-Host "Repository: $Root"
  Write-Host "Branch: $(git branch --show-current)"

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

  Write-Host "Latest commit: $(git log --oneline -1)"
  Write-Host "Pushing with proxy: $Proxy"
  git -c "http.proxy=$Proxy" -c "https.proxy=$Proxy" push
} finally {
  Pop-Location
}
