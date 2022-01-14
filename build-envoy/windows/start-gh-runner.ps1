param (
    [string]$Token = $(throw "-Token is required."),
    [string]$LogonAccount = "installer",
    [string]$LogonPassword = $(throw "-LogonPassword is required."),
    [string]$RepositoryURL = $(throw "-RepositoryURL is required."),
    [string]$RunnerPath = "C:\actions-runner",
    [string]$RunnerZipURI = "https://github.com/actions/runner/releases/download/v2.285.1/actions-runner-win-x64-2.285.1.zip",
    [string]$Labels = "envoy"
)

If(!(Test-Path $RunnerPath)) {
  New-Item -ItemType Directory -Force -Path $RunnerPath
}

Set-Location -Path $RunnerPath

Invoke-WebRequest `
  -Uri $RunnerZipURI `
  -OutFile actions-runner-win-x64.zip

$wantHash = 'f79dbb6dfae9d42d0befb8cff30a145dd32c9b1df6ff280c9935c46884b001f3'.ToUpper()
$hash = $(Get-FileHash -Path actions-runner-win-x64.zip -Algorithm SHA256).Hash.ToUpper()

if($hash -ne $wantHash) {
  throw 'Computed checksum did not match'
}

Add-Type -AssemblyName System.IO.Compression.FileSystem; [System.IO.Compression.ZipFile]::ExtractToDirectory("$PWD/actions-runner-win-x64.zip", "$PWD")

./config.cmd `
  --url $RepositoryURL `
  --token $Token `
  --windowslogonaccount $LogonAccount `
  --windowslogonpassword $LogonPassword `
  --labels $Labels `
  --runasservice `
  --unattended
