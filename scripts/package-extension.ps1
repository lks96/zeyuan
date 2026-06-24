param(
  [string]$ProjectRoot = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path,
  [string]$ApiBase = $(if ($env:TEMU_TOOLS_EXTENSION_API_BASE) { $env:TEMU_TOOLS_EXTENSION_API_BASE } else { "http://localhost:8080/api" })
)

$extensionDir = Join-Path $ProjectRoot "chrome-extension"
$downloadDir = Join-Path $ProjectRoot "frontend\public\downloads"
$zipPath = Join-Path $downloadDir "temu-seller-sync-extension.zip"
$tempRoot = Join-Path ([System.IO.Path]::GetTempPath()) "temu-tools-extension-package"
$tempDir = Join-Path $tempRoot ([System.Guid]::NewGuid().ToString("N"))

if (-not (Test-Path -LiteralPath $extensionDir -PathType Container)) {
  throw "Extension directory not found: $extensionDir"
}

New-Item -ItemType Directory -Path $downloadDir -Force | Out-Null
New-Item -ItemType Directory -Path $tempDir -Force | Out-Null

if (Test-Path -LiteralPath $zipPath) {
  Remove-Item -LiteralPath $zipPath -Force
}

try {
  Copy-Item -Path (Join-Path $extensionDir "*") -Destination $tempDir -Recurse -Force

  $encodedApiBase = $ApiBase.TrimEnd("/")
  $jsApiBase = $encodedApiBase.Replace("\", "\\").Replace("'", "\'")
  $configContent = @"
globalThis.TEMU_TOOLS_EXTENSION_CONFIG = {
  apiBase: '$jsApiBase',
}
"@
  Set-Content -LiteralPath (Join-Path $tempDir "config.js") -Value $configContent -Encoding UTF8

  $items = Get-ChildItem -LiteralPath $tempDir -Force | Where-Object {
    $_.Name -notin @(".DS_Store", "Thumbs.db")
  }

  Compress-Archive -Path $items.FullName -DestinationPath $zipPath -CompressionLevel Optimal
  Write-Host "Created $zipPath"
  Write-Host "Configured API base: $encodedApiBase"
} finally {
  $resolvedTempRoot = [System.IO.Path]::GetFullPath($tempRoot)
  $resolvedTempDir = [System.IO.Path]::GetFullPath($tempDir)
  if ($resolvedTempDir.StartsWith($resolvedTempRoot, [System.StringComparison]::OrdinalIgnoreCase) -and (Test-Path -LiteralPath $tempDir)) {
    Remove-Item -LiteralPath $tempDir -Recurse -Force
  }
}
