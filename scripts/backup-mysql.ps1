param()

$required = @("MYSQL_HOST", "MYSQL_PORT", "MYSQL_USER", "MYSQL_PASSWORD", "MYSQL_DATABASE", "BACKUP_DIRECTORY")
foreach ($name in $required) {
    if ([string]::IsNullOrWhiteSpace([Environment]::GetEnvironmentVariable($name))) {
        throw "Missing required environment variable: $name"
    }
}

$backupDirectory = [System.IO.Path]::GetFullPath($env:BACKUP_DIRECTORY)
[System.IO.Directory]::CreateDirectory($backupDirectory) | Out-Null
$timestamp = [DateTime]::UtcNow.ToString("yyyyMMdd-HHmmss")
$backupFile = Join-Path $backupDirectory "lms-cn-$timestamp.sql"
$checksumFile = "$backupFile.sha256"
$env:MYSQL_PWD = $env:MYSQL_PASSWORD

try {
    & mysqldump --host=$env:MYSQL_HOST --port=$env:MYSQL_PORT --user=$env:MYSQL_USER --single-transaction --routines --triggers --set-gtid-purged=OFF --result-file=$backupFile $env:MYSQL_DATABASE
    if ($LASTEXITCODE -ne 0) { throw "mysqldump failed with exit code $LASTEXITCODE" }
    $hash = (Get-FileHash -LiteralPath $backupFile -Algorithm SHA256).Hash.ToLowerInvariant()
    Set-Content -LiteralPath $checksumFile -Value $hash -Encoding ascii
    Write-Output "Backup completed: $backupFile"
    Write-Output "Checksum: $checksumFile"
}
finally {
    Remove-Item Env:MYSQL_PWD -ErrorAction SilentlyContinue
}

