param(
  [switch]$SkipMigrate
)

$ErrorActionPreference = "Stop"

function Initialize-Console {
  try {
    chcp.com 65001 > $null
    [Console]::InputEncoding = [System.Text.UTF8Encoding]::new()
    [Console]::OutputEncoding = [System.Text.UTF8Encoding]::new()
    $script:OutputEncoding = [System.Text.UTF8Encoding]::new()
  } catch {
  }

  $env:NO_COLOR = "1"
  $env:FORCE_COLOR = "0"
  $env:npm_config_color = "false"
  $env:VITE_CJS_IGNORE_WARNING = "true"
}

$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$Root = Split-Path -Parent $ScriptDir
$Backend = Join-Path $Root "backend"
$Frontend = Join-Path $Root "frontend"
$EnvFile = Join-Path $Backend ".env"
$EnvExample = Join-Path $Backend ".env.example"
$BundledTools = if ($env:TEMU_TOOLS_RUNTIME) { $env:TEMU_TOOLS_RUNTIME } else { "E:\tools" }
$BundledActivate = Join-Path $BundledTools "activate.ps1"

function Use-LocalTools {
  if (Test-Path $BundledActivate) {
    . $BundledActivate
    if (Test-Path (Join-Path $BundledTools "go-path")) {
      $env:GOPATH = Join-Path $BundledTools "go-path"
    }
    if (Test-Path (Join-Path $BundledTools "go-pkg-mod")) {
      $env:GOMODCACHE = Join-Path $BundledTools "go-pkg-mod"
    }
    if (Test-Path (Join-Path $BundledTools "go-cache")) {
      $env:GOCACHE = Join-Path $BundledTools "go-cache"
    }
  }
}

function Ensure-BackendEnv {
  if (-not (Test-Path $EnvFile)) {
    Copy-Item $EnvExample $EnvFile
    Write-Warning "Created backend\.env. Please fill DB_PASSWORD before starting."
    return $false
  }

  $envContent = Get-Content $EnvFile -Raw
  if ($envContent -match "DB_PASSWORD\s*=\s*change_me" -and [string]::IsNullOrWhiteSpace($env:DB_PASSWORD)) {
    Write-Warning "DB_PASSWORD in backend\.env is still change_me. Please set the real password."
    return $false
  }

  return $true
}

Initialize-Console
Use-LocalTools

Write-Host "Checking Go / Node / npm..."
go version
node --version
npm --version

$envReady = Ensure-BackendEnv

Write-Host "Installing frontend dependencies..."
Push-Location $Frontend
try {
  npm install
} finally {
  Pop-Location
}

Write-Host "Downloading backend dependencies..."
Push-Location $Backend
try {
  go mod download
  if ($envReady -and -not $SkipMigrate) {
    Write-Host "Preparing database schema..."
    go run ./cmd/dbprepare
  }
} finally {
  Pop-Location
}

if (-not $envReady) {
  Write-Host ""
  Write-Host "Setup is incomplete. Edit backend\.env, then run .\scripts\setup-dev.ps1 again."
  exit 1
}

Write-Host ""
Write-Host "Setup complete. Start the app with: .\scripts\start-dev.ps1"
