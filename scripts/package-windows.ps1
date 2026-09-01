param(
    [Parameter(Mandatory = $true)][string]$Version,
    [Parameter(Mandatory = $true)][string]$ExePath,
    [string]$OutputDir = (Join-Path $PSScriptRoot '..\release')
)

$ErrorActionPreference = 'Stop'
$root = (Resolve-Path (Join-Path $PSScriptRoot '..')).Path
$staging = Join-Path $OutputDir "cc-connect-$Version-windows-amd64"
$zip = "$staging.zip"

if (Test-Path -LiteralPath $staging) { throw "Staging directory already exists: $staging" }
if (Test-Path -LiteralPath $zip) { throw "Release archive already exists: $zip" }

$null = New-Item -ItemType Directory -Path $staging -Force
Copy-Item -LiteralPath $ExePath -Destination (Join-Path $staging 'cc-connect.exe')
Copy-Item -LiteralPath (Join-Path $root 'scripts\install-windows.ps1') -Destination $staging
Copy-Item -LiteralPath (Join-Path $root 'scripts\cc-connect-launch.ps1') -Destination $staging
Copy-Item -LiteralPath (Join-Path $root 'README.md') -Destination $staging
Copy-Item -LiteralPath (Join-Path $root 'LICENSE') -Destination $staging

Compress-Archive -LiteralPath $staging -DestinationPath $zip -CompressionLevel Optimal
$hash = Get-FileHash -LiteralPath $zip -Algorithm SHA256
"$($hash.Hash.ToLowerInvariant())  $([IO.Path]::GetFileName($zip))" | Set-Content -LiteralPath "$zip.sha256" -Encoding ascii
Write-Output $zip
