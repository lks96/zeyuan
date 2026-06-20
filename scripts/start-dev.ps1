param(
  [switch]$SkipInstall,
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
    exit 1
  }

  $envContent = Get-Content $EnvFile -Raw
  if ($envContent -match "DB_PASSWORD\s*=\s*change_me" -and [string]::IsNullOrWhiteSpace($env:DB_PASSWORD)) {
    Write-Warning "DB_PASSWORD in backend\.env is still change_me. Please set the real password."
    exit 1
  }
}

Initialize-Console
Use-LocalTools
Ensure-BackendEnv

if (-not $SkipInstall -and -not (Test-Path (Join-Path $Frontend "node_modules"))) {
  Write-Host "frontend\node_modules not found. Installing frontend dependencies..."
  Push-Location $Frontend
  try {
    npm install
  } finally {
    Pop-Location
  }
}

if (-not $SkipMigrate) {
  Write-Host "Running database migrations..."
  Push-Location $Backend
  try {
    go run ./cmd/migrate
  } finally {
    Pop-Location
  }
}

$backendScript = {
  param($BackendPath, $ActivatePath, $ToolsRoot)
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
  Set-Location $BackendPath
  if (Test-Path $ActivatePath) {
    . $ActivatePath
    $goPath = Join-Path $ToolsRoot "go-path"
    $goModCache = Join-Path $ToolsRoot "go-pkg-mod"
    $goCache = Join-Path $ToolsRoot "go-cache"
    if (Test-Path $goPath) { $env:GOPATH = $goPath }
    if (Test-Path $goModCache) { $env:GOMODCACHE = $goModCache }
    if (Test-Path $goCache) { $env:GOCACHE = $goCache }
  }
  & go run ./cmd/server 2>&1
}

$frontendScript = {
  param($FrontendPath, $ActivatePath)
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
  Set-Location $FrontendPath
  if (Test-Path $ActivatePath) {
    . $ActivatePath
  }
  & npm run dev -- --host 0.0.0.0 2>&1
}

$backendJob = $null
$frontendJob = $null

try {
  $backendJob = Start-Job -Name "temu-tools-backend" -ScriptBlock $backendScript -ArgumentList $Backend, $BundledActivate, $BundledTools
  $frontendJob = Start-Job -Name "temu-tools-frontend" -ScriptBlock $frontendScript -ArgumentList $Frontend, $BundledActivate

  Write-Host ""
  Write-Host "Temu Tools is starting..."
  Write-Host "Backend API: http://localhost:8080"
  Write-Host "Frontend: http://localhost:5173"
  Write-Host "Press Ctrl+C to stop both services."
  Write-Host ""

  while ($true) {
    Receive-Job -Job $backendJob, $frontendJob
    if ($backendJob.State -ne "Running" -or $frontendJob.State -ne "Running") {
      Receive-Job -Job $backendJob, $frontendJob
      throw "Backend or frontend exited. Check MySQL connectivity and the logs above."
    }
    Start-Sleep -Milliseconds 500
  }
} finally {
  if ($backendJob) {
    Stop-Job $backendJob -ErrorAction SilentlyContinue
    Remove-Job $backendJob -Force -ErrorAction SilentlyContinue
  }
  if ($frontendJob) {
    Stop-Job $frontendJob -ErrorAction SilentlyContinue
    Remove-Job $frontendJob -Force -ErrorAction SilentlyContinue
  }
}
