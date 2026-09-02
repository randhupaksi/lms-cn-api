param(
    [Parameter(Mandatory = $true)][string]$BackupFile,
    [switch]$Force
)

if (-not $Force) { throw "Restore is destructive. Re-run with -Force after verifying the target database." }
$resolvedBackup = (Resolve-Path -LiteralPath $BackupFile -ErrorAction Stop).Path
$required = @("MYSQL_HOST", "MYSQL_PORT", "MYSQL_USER", "MYSQL_PASSWORD", "MYSQL_DATABASE")
foreach ($name in $required) {
    if ([string]::IsNullOrWhiteSpace([Environment]::GetEnvironmentVariable($name))) { throw "Missing required environment variable: $name" }
}
$checksumFile = "$resolvedBackup.sha256"
if (Test-Path -LiteralPath $checksumFile) {
    $expected = (Get-Content -LiteralPath $checksumFile -Raw).Trim().ToLowerInvariant()
    $actual = (Get-FileHash -LiteralPath $resolvedBackup -Algorithm SHA256).Hash.ToLowerInvariant()
    if ($expected -ne $actual) { throw "Backup checksum verification failed." }
}
$env:MYSQL_PWD = $env:MYSQL_PASSWORD
try {
    Get-Content -LiteralPath $resolvedBackup -Raw | & mysql --host=$env:MYSQL_HOST --port=$env:MYSQL_PORT --user=$env:MYSQL_USER $env:MYSQL_DATABASE
    if ($LASTEXITCODE -ne 0) { throw "mysql restore failed with exit code $LASTEXITCODE" }
    Write-Output "Restore completed for database $env:MYSQL_DATABASE"
}
finally {
    Remove-Item Env:MYSQL_PWD -ErrorAction SilentlyContinue
}

