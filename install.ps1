# pigment installer for Windows
# Usage: irm https://raw.githubusercontent.com/developerAkX/pigment/main/install.ps1 | iex

$ErrorActionPreference = "Stop"
$Repo = "developerAkX/pigment"

# Config via environment:
#   $env:PIGMENT_SKILLS_AGENT  agent(s) to install skills for (default: opencode; '*' = all)
#   $env:PIGMENT_NO_SKILLS=1   skip `npx skills add` skill installation
$SkillsAgent = if ($env:PIGMENT_SKILLS_AGENT) { $env:PIGMENT_SKILLS_AGENT } else { "opencode" }

function Main {
    Write-Host "Installing pigment..." -ForegroundColor Cyan
    Write-Host ""

    $arch = Get-Arch
    $version = Get-LatestVersion
    $asset = "pigment_${version}_windows_${arch}.zip"
    $baseUrl = "https://github.com/$Repo/releases/download/v$version"

    $tmpDir = New-TemporaryFile | ForEach-Object {
        Remove-Item $_; New-Item -ItemType Directory -Path $_
    }

    try {
        # Download
        Write-Host "Downloading $asset..."
        Invoke-WebRequest -Uri "$baseUrl/$asset" -OutFile "$tmpDir\$asset"
        Invoke-WebRequest -Uri "$baseUrl/checksums.txt" -OutFile "$tmpDir\checksums.txt"

        # Verify checksum
        Write-Host "Verifying checksum..."
        $expected = (Get-Content "$tmpDir\checksums.txt" |
            Where-Object { $_ -match $asset } |
            ForEach-Object { ($_ -split '\s+')[0] })

        if (-not $expected) {
            throw "Checksum not found for $asset"
        }

        $actual = (Get-FileHash -Path "$tmpDir\$asset" -Algorithm SHA256).Hash.ToLower()
        if ($actual -ne $expected) {
            throw "Checksum mismatch! Expected: $expected, Got: $actual"
        }
        Write-Host "Checksum OK." -ForegroundColor Green

        # Extract
        Write-Host "Extracting..."
        Expand-Archive -Path "$tmpDir\$asset" -DestinationPath "$tmpDir\extracted"

        # Install
        $installDir = "$env:LOCALAPPDATA\pigment\bin"
        New-Item -ItemType Directory -Path $installDir -Force | Out-Null
        Copy-Item "$tmpDir\extracted\pigment.exe" "$installDir\pigment.exe" -Force

        # Add to PATH if needed
        $userPath = [Environment]::GetEnvironmentVariable("Path", "User")
        if ($userPath -notlike "*$installDir*") {
            [Environment]::SetEnvironmentVariable(
                "Path", "$userPath;$installDir", "User")
            Write-Host "Added $installDir to user PATH." -ForegroundColor Yellow
            Write-Host "Restart your terminal for PATH changes to take effect."
        }

        Write-Host ""
        Write-Host "Installed binary to $installDir\pigment.exe" -ForegroundColor Green

        Install-Skills -BinPath "$installDir\pigment.exe"

        Write-Host ""
        Write-Host "Done! Next steps:"
        Write-Host "  1. Authenticate your ChatGPT subscription:  codex login"
        Write-Host "  2. Verify everything is ready:              pigment doctor"
        Write-Host "  3. Generate your first image:               pigment gen `"a red bicycle`""
        Write-Host ""
        Write-Host "Skills installed for '$SkillsAgent'. Re-run any time with:"
        Write-Host "  npx skills add $Repo --agent <agent> --global --yes"
    }
    finally {
        Remove-Item -Recurse -Force $tmpDir -ErrorAction SilentlyContinue
    }
}

# Install the agent skills via the skills.sh registry (`npx skills add`).
# Falls back to the binary's embedded installer if npx is unavailable.
function Install-Skills {
    param([string]$BinPath)

    Write-Host ""
    if ($env:PIGMENT_NO_SKILLS -eq "1") {
        Write-Host "Skipping skill installation (PIGMENT_NO_SKILLS=1)."
        return
    }

    Write-Host "Installing agent skills (agent: $SkillsAgent)..." -ForegroundColor Cyan
    if (Get-Command npx -ErrorAction SilentlyContinue) {
        try {
            & npx -y skills@latest add $Repo --skill '*' --agent $SkillsAgent --global --yes
            if ($LASTEXITCODE -eq 0) {
                Write-Host "Skills installed via 'npx skills add'." -ForegroundColor Green
                return
            }
            Write-Host "WARN: 'npx skills add' failed; falling back to the embedded installer." -ForegroundColor Yellow
        }
        catch {
            Write-Host "WARN: 'npx skills add' errored; falling back to the embedded installer." -ForegroundColor Yellow
        }
    }
    else {
        Write-Host "npx/Node not found; using the embedded skill installer."
    }

    if (Test-Path $BinPath) {
        & $BinPath skill install --force
    }
}

function Get-Arch {
    $arch = [System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture
    switch ($arch) {
        "X64"   { return "amd64" }
        "Arm64" { return "arm64" }
        default { throw "Unsupported architecture: $arch" }
    }
}

function Get-LatestVersion {
    $release = Invoke-RestMethod -Uri "https://api.github.com/repos/$Repo/releases/latest"
    $tag = $release.tag_name
    if (-not $tag) { throw "Failed to determine latest version" }
    return $tag.TrimStart("v")
}

Main
