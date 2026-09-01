$ErrorActionPreference = 'Stop'

try {
    $exe = Join-Path $PSScriptRoot 'cc-connect.exe'
    if (-not (Test-Path -LiteralPath $exe)) {
        throw "cc-connect.exe was not found in $PSScriptRoot"
    }
    $process = Start-Process -FilePath $exe -ArgumentList 'desktop' -WindowStyle Hidden -Wait -PassThru
    if ($process.ExitCode -ne 0) {
        throw "cc-connect exited with code $($process.ExitCode)."
    }
} catch {
    $logPath = Join-Path ([Environment]::GetFolderPath('UserProfile')) '.cc-connect\logs\desktop.log'
    $message = "$($_.Exception.Message)`n`nLog: $logPath"
    $shell = New-Object -ComObject WScript.Shell
    $null = $shell.Popup($message, 0, 'CC-Connect could not start', 16)
    exit 1
}
