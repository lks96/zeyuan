param(
  [string]$BaseUrl = "http://localhost:8080"
)

$ErrorActionPreference = "Stop"

function Assert-True {
  param(
    [bool]$Condition,
    [string]$Message
  )

  if (-not $Condition) {
    throw "Assertion failed: $Message"
  }
}

function Invoke-Json {
  param(
    [string]$Method,
    [string]$Path,
    [hashtable]$Headers = @{},
    [object]$Body = $null
  )

  $params = @{
    Method = (Convert-Method $Method)
    Uri = "$BaseUrl$Path"
    Headers = $Headers
  }

  if ($null -ne $Body) {
    $params.ContentType = "application/json"
    $params.Body = ($Body | ConvertTo-Json -Depth 8)
  }

  Invoke-RestMethod @params
}

function Invoke-Status {
  param(
    [string]$Method,
    [string]$Path,
    [hashtable]$Headers = @{},
    [object]$Body = $null
  )

  try {
    $params = @{
      Method = (Convert-Method $Method)
      Uri = "$BaseUrl$Path"
      Headers = $Headers
      ErrorAction = "Stop"
    }
    if ($null -ne $Body) {
      $params.ContentType = "application/json"
      $params.Body = ($Body | ConvertTo-Json -Depth 8)
    }
    $response = Invoke-WebRequest @params
    [int]$response.StatusCode
  } catch {
    [int]$_.Exception.Response.StatusCode
  }
}

function Convert-Method {
  param([string]$Method)

  switch ($Method.ToUpperInvariant()) {
    "GET" { "Get" }
    "POST" { "Post" }
    "PUT" { "Put" }
    "PATCH" { "Patch" }
    "DELETE" { "Delete" }
    default { $Method }
  }
}

$health = Invoke-Json -Method GET -Path "/api/health"
Assert-True ($health.data.database -eq "ok") "database health should be ok"

$adminLogin = Invoke-Json -Method POST -Path "/api/auth/login" -Body @{
  username = "admin"
  password = "admin123"
}
$operatorLogin = Invoke-Json -Method POST -Path "/api/auth/login" -Body @{
  username = "operator_a"
  password = "operator123"
}

$adminHeaders = @{ Authorization = "Bearer $($adminLogin.data.token)" }
$operatorHeaders = @{ Authorization = "Bearer $($operatorLogin.data.token)" }
$testRunId = [DateTimeOffset]::UtcNow.ToUnixTimeMilliseconds()

Assert-True ($adminLogin.data.permissions -contains "users:view") "admin should have users:view"
Assert-True ($adminLogin.data.permissions -contains "settings:update") "admin should have settings:update"
Assert-True ($operatorLogin.data.permissions -contains "shops:view") "operator should have shops:view"
Assert-True (-not ($operatorLogin.data.permissions -contains "users:view")) "operator should not have users:view"

$operatorUsersStatus = Invoke-Status -Method GET -Path "/api/users" -Headers $operatorHeaders
Assert-True ($operatorUsersStatus -eq 403) "operator /api/users should be forbidden"

$me = Invoke-Json -Method GET -Path "/api/me" -Headers $adminHeaders
Assert-True ($me.data.user.username -eq "admin") "me should return admin"

$summary = Invoke-Json -Method GET -Path "/api/tenant/summary" -Headers $adminHeaders
Assert-True ($summary.data.totalUsers -ge 2) "summary should include seeded users"

$modules = Invoke-Json -Method GET -Path "/api/modules" -Headers $adminHeaders
Assert-True ($modules.data.Count -ge 4) "modules should include seeded modules"

$testModuleId = "api-test-module"
$savedModule = Invoke-Json -Method POST -Path "/api/modules" -Headers $adminHeaders -Body @{
  id = $testModuleId
  name = "API Test Module"
  description = "Created by integration test"
  status = "planning"
  sortOrder = 999
}
Assert-True ($savedModule.data.id -eq $testModuleId) "module should be saved"

$updatedModule = Invoke-Json -Method PUT -Path "/api/modules/$testModuleId" -Headers $adminHeaders -Body @{
  name = "API Test Module Updated"
  description = "Updated by integration test"
  status = "paused"
  sortOrder = 998
}
Assert-True ($updatedModule.data.status -eq "paused") "module should be updated"

$deleteModule = Invoke-Json -Method DELETE -Path "/api/modules/$testModuleId" -Headers $adminHeaders
Assert-True ($deleteModule.data.deleted -eq $true) "module should be deleted"

$extractBatches = Invoke-Json -Method GET -Path "/api/tools/delivery-extractions" -Headers $adminHeaders
Assert-True ($null -ne $extractBatches.data) "delivery extract batches should be listed"

$importExtract = Invoke-Json -Method POST -Path "/api/tools/delivery-extractions/import-source" -Headers $adminHeaders
Assert-True ($importExtract.data.extractedTotal -ge 1) "delivery source should be imported"
Assert-True ($importExtract.data.data.Count -eq $importExtract.data.extractedTotal) "import response should include extracted rows"
Assert-True ($importExtract.data.data[0].id -ge 1) "extracted row should include auto increment id"

$latestExtract = Invoke-Json -Method GET -Path "/api/tools/delivery-extractions/latest" -Headers $adminHeaders
Assert-True ($latestExtract.data.id -eq $importExtract.data.id) "latest delivery extract should match imported batch"

$settings = Invoke-Json -Method GET -Path "/api/settings" -Headers $adminHeaders
Assert-True ($settings.data.Count -ge 4) "settings should include seeded keys"

$updatedSettings = Invoke-Json -Method PUT -Path "/api/settings" -Headers $adminHeaders -Body @{
  values = @{
    sync_interval = "30 分钟"
    shop_alias = "integration-test"
  }
}
Assert-True (($updatedSettings.data | Where-Object { $_.key -eq "shop_alias" }).value -eq "integration-test") "settings should update"

$permissions = Invoke-Json -Method GET -Path "/api/permissions" -Headers $adminHeaders
Assert-True ($permissions.data.Count -ge 10) "permissions should be listed"

$rolePermissions = Invoke-Json -Method GET -Path "/api/roles/user/permissions" -Headers $adminHeaders
Assert-True ($rolePermissions.data -contains "shops:view") "user role should have shops:view"

$shops = Invoke-Json -Method GET -Path "/api/shops" -Headers $adminHeaders
Assert-True ($shops.data.Count -ge 2) "shops should include seeded shops"

$testShopCode = "api-test-shop-$testRunId"
$createdShop = Invoke-Json -Method POST -Path "/api/shops" -Headers $adminHeaders -Body @{
  shopName = "API Test Shop"
  platform = "temu"
  externalCode = $testShopCode
  status = "active"
}
Assert-True ($createdShop.data.externalCode -eq $testShopCode) "shop should be created"

$updatedShop = Invoke-Json -Method PUT -Path "/api/shops/$($createdShop.data.id)" -Headers $adminHeaders -Body @{
  shopName = "API Test Shop Updated"
  platform = "temu"
  externalCode = $testShopCode
  status = "paused"
}
Assert-True ($updatedShop.data.status -eq "paused") "shop should be updated"

$users = Invoke-Json -Method GET -Path "/api/users" -Headers $adminHeaders
Assert-True ($users.data.Count -ge 2) "users should include seeded users"

$assignShop = Invoke-Json -Method POST -Path "/api/users/2/shops" -Headers $adminHeaders -Body @{
  shopId = $createdShop.data.id
  shopRole = "viewer"
}
Assert-True ($assignShop.data.assigned -eq $true) "shop should be assigned"

$userShops = Invoke-Json -Method GET -Path "/api/users/2/shops" -Headers $adminHeaders
Assert-True (($userShops.data | Where-Object { $_.shopId -eq $createdShop.data.id }).shopRole -eq "viewer") "assignment should be visible"

$removeAssignment = Invoke-Json -Method DELETE -Path "/api/users/2/shops/$($createdShop.data.id)" -Headers $adminHeaders
Assert-True ($removeAssignment.data.removed -eq $true) "assignment should be removed"

$closeShop = Invoke-Json -Method DELETE -Path "/api/shops/$($createdShop.data.id)" -Headers $adminHeaders
Assert-True ($closeShop.data.closed -eq $true) "shop should be closed"

Write-Host "All API integration tests passed."
