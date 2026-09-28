$ErrorActionPreference = 'Stop'

$pgCtl = 'C:\Program Files\PostgreSQL\17\bin\pg_ctl.exe'
$psql = 'C:\Program Files\PostgreSQL\17\bin\psql.exe'
$dataDirectory = Join-Path $PSScriptRoot '..\.test-postgres'
$logFile = Join-Path $PSScriptRoot '..\pg-local.log'

if (-not (Test-Path -LiteralPath $pgCtl)) {
    throw 'PostgreSQL 17 was not found in C:\Program Files\PostgreSQL\17.'
}
if (-not (Test-Path -LiteralPath $dataDirectory)) {
    throw "Project database directory was not found: $dataDirectory"
}

& $pgCtl status -D $dataDirectory *> $null
if ($LASTEXITCODE -ne 0) {
    & $pgCtl start -D $dataDirectory -l $logFile -o '-p 55432 -h 127.0.0.1'
    if ($LASTEXITCODE -ne 0) {
        throw "PostgreSQL could not start. Check $logFile"
    }
}

$deadline = (Get-Date).AddSeconds(45)
do {
    Start-Sleep -Milliseconds 500
    & $psql -h 127.0.0.1 -p 55432 -U tkd_test -d taekwondo_web -X -tAc 'SELECT 1' *> $null
    $ready = $LASTEXITCODE -eq 0
} until ($ready -or (Get-Date) -ge $deadline)

if (-not $ready) {
    throw 'PostgreSQL started but the taekwondo_web database did not become ready.'
}

Write-Host 'Taekwondo PostgreSQL is ready on 127.0.0.1:55432.' -ForegroundColor Green
