param(
    [Parameter(Mandatory = $true)][string]$BackupFile,
    [switch]$Force,
    [switch]$SkipChecksum
)

if (-not $Force) { throw "Restore is destructive. Re-run with -Force after verifying the target database." }
$resolvedBackup = (Resolve-Path -LiteralPath $BackupFile -ErrorAction Stop).Path
$required = @("MYSQL_HOST", "MYSQL_PORT", "MYSQL_USER", "MYSQL_PASSWORD", "MYSQL_DATABASE")
foreach ($name in $required) {
    if ([string]::IsNullOrWhiteSpace([Environment]::GetEnvironmentVariable($name))) { throw "Missing required environment variable: $name" }
}
$checksumFile = "$resolvedBackup.sha256"
if (-not (Test-Path -LiteralPath $checksumFile -PathType Leaf)) {
    if (-not $SkipChecksum) {
        throw "Backup checksum file is required. Use -SkipChecksum only after independently verifying a legacy backup."
    }
    Write-Warning "Backup has no checksum sidecar. Verification was explicitly skipped."
}
else {
    $expected = (Get-Content -LiteralPath $checksumFile -Raw).Trim().ToLowerInvariant()
    if ($expected -notmatch '^[0-9a-f]{64}$') { throw "Backup checksum file is invalid." }
    $actual = (Get-FileHash -LiteralPath $resolvedBackup -Algorithm SHA256).Hash.ToLowerInvariant()
    if ($expected -ne $actual) { throw "Backup checksum verification failed." }
}
$env:MYSQL_PWD = $env:MYSQL_PASSWORD
try {
    $OutputEncoding = [System.Text.UTF8Encoding]::new($false)
    Get-Content -LiteralPath $resolvedBackup -Encoding UTF8 -ErrorAction Stop |
        & mysql --binary-mode --default-character-set=utf8mb4 --host=$env:MYSQL_HOST --port=$env:MYSQL_PORT --user=$env:MYSQL_USER $env:MYSQL_DATABASE
    if ($LASTEXITCODE -ne 0) { throw "mysql restore failed with exit code $LASTEXITCODE" }
    Write-Output "Restore completed for database $env:MYSQL_DATABASE"
}
finally {
    Remove-Item Env:MYSQL_PWD -ErrorAction SilentlyContinue
}

