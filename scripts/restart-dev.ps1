param(
  [switch]$SkipInstall,
  [switch]$SkipMigrate
)

$ErrorActionPreference = "Stop"

$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$Root = Split-Path -Parent $ScriptDir
$Logs = Join-Path $Root "logs"
$StartScript = Join-Path $ScriptDir "start-dev.ps1"
$Ports = @(8080, 5173)

function Stop-DevPorts {
  $connections = Get-NetTCPConnection -LocalPort $Ports -State Listen -ErrorAction SilentlyContinue
  $processIds = $connections | Select-Object -ExpandProperty OwningProcess -Unique

  foreach ($processId in $processIds) {
    if (-not $processId -or $processId -eq 0) {
      continue
    }

    $process = Get-Process -Id $processId -ErrorAction SilentlyContinue
    if ($process) {
      Write-Host "Stopping $($process.ProcessName) on pid $processId..."
      Stop-Process -Id $processId -Force -ErrorAction SilentlyContinue
    }
  }
}

function Wait-ForPortsToClose {
  $deadline = (Get-Date).AddSeconds(10)
  while ((Get-Date) -lt $deadline) {
    $open = Get-NetTCPConnection -LocalPort $Ports -State Listen -ErrorAction SilentlyContinue
    if (-not $open) {
      return
    }
    Start-Sleep -Milliseconds 500
  }
}

function Start-DevServices {
  New-Item -ItemType Directory -Force -Path $Logs | Out-Null

  $outLog = Join-Path $Logs "dev.out.log"
  $errLog = Join-Path $Logs "dev.err.log"
  $arguments = @(
    "-NoProfile",
    "-ExecutionPolicy", "Bypass",
    "-File", $StartScript
  )

  if ($SkipInstall) {
    $arguments += "-SkipInstall"
  }
  if ($SkipMigrate) {
    $arguments += "-SkipMigrate"
  }

  $launcher = Start-Process `
    -FilePath "powershell.exe" `
    -ArgumentList $arguments `
    -WorkingDirectory $Root `
    -WindowStyle Hidden `
    -RedirectStandardOutput $outLog `
    -RedirectStandardError $errLog `
    -PassThru

  Write-Host "Started launcher pid $($launcher.Id)."
  Write-Host "Output log: $outLog"
  Write-Host "Error log: $errLog"
}

function Wait-ForServices {
  $deadline = (Get-Date).AddSeconds(30)
  while ((Get-Date) -lt $deadline) {
    $listeningPorts = Get-NetTCPConnection -LocalPort $Ports -State Listen -ErrorAction SilentlyContinue |
      Select-Object -ExpandProperty LocalPort -Unique

    if (($listeningPorts -contains 8080) -and ($listeningPorts -contains 5173)) {
      return
    }

    Start-Sleep -Seconds 1
  }

  throw "Services did not start within 30 seconds. Check logs in $Logs."
}

Write-Host "Restarting Temu Tools dev services..."
Stop-DevPorts
Wait-ForPortsToClose
Start-DevServices
Wait-ForServices

Write-Host ""
Write-Host "Temu Tools restarted."
Write-Host "Backend API: http://localhost:8080"
Write-Host "Frontend: http://localhost:5173"
