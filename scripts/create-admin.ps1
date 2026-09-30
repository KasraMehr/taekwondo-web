param(
    [string]$Email,
    [Security.SecureString]$Password,
    [string]$Name = 'TKDHub Admin',
    [string]$Organization = 'TKDHub',
    [string]$BaseUrl
)
$ErrorActionPreference = 'Stop'
if ($BaseUrl) { throw 'Public account creation is disabled. Use SSH and sudo bash /opt/tkdhub/scripts/create-admin.sh on the server.' }
if (-not $Email) { $Email = Read-Host 'Admin email' }
if (-not $Password) { $Password = Read-Host 'Password (10-72 UTF-8 bytes)' -AsSecureString }
$plainPassword = [Net.NetworkCredential]::new('', $Password).Password
$previousEncoding = $OutputEncoding
Push-Location (Join-Path $PSScriptRoot '../backend')
try {
    $OutputEncoding = [Text.UTF8Encoding]::new($false)
    $plainPassword | & go run ./cmd/create-admin -email $Email -name $Name -organization $Organization
    if ($LASTEXITCODE -ne 0) { throw 'Account provisioning failed; see the error above.' }
    Write-Host 'Local admin created. Sign in at http://localhost:5173' -ForegroundColor Green
} finally {
    $plainPassword = $null
    $Password = $null
    $OutputEncoding = $previousEncoding
    Pop-Location
}