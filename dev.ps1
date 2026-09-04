<#
  Starts the full local ACMP stack, each process in its own PowerShell window:

    Postgres + Redis   docker compose (backend/docker-compose.yml)
    ACMP api           backend/  ->  go run ./cmd/api        (:8080)
    ACMP worker        backend/  ->  go run ./cmd/worker
    ACMP web           frontend/ ->  npm run dev             (:3000)

  Usage:
    .\dev.ps1          start everything
    .\dev.ps1 -Down    docker compose down (stop Postgres + Redis)

  Stop the app: close the three "ACMP *" windows, then run  .\dev.ps1 -Down
#>
param([switch]$Down)

$ErrorActionPreference = "Stop"
$root     = $PSScriptRoot
$backend  = Join-Path $root "backend"
$frontend = Join-Path $root "frontend"

# Run a docker command for its exit code only. `cmd /c` swallows stdout+stderr
# at the shell level so Docker's harmless stderr warnings (e.g. "No blkio
# throttle...") don't trip PowerShell's stop-on-native-stderr behaviour.
function Invoke-DockerQuiet($argString) {
    cmd /c "docker $argString >nul 2>nul"
    return $LASTEXITCODE
}

if ($Down) {
    Push-Location $backend
    cmd /c "docker compose down"
    Pop-Location
    return
}

# --- resolve go -----------------------------------------------------------
$go = (Get-Command go -ErrorAction SilentlyContinue).Source
if (-not $go) { $go = "C:\Program Files\Go\bin\go.exe" }
if (-not (Test-Path $go)) {
    throw "Go not found on PATH or at '$go'. Install Go, or open a fresh terminal."
}

# --- docker running? ----------------------------------------------------
if ((Invoke-DockerQuiet "info") -ne 0) {
    throw "Docker Desktop isn't running (or its engine is still starting). Start it, wait for the green whale, and retry."
}

# --- .env ---------------------------------------------------------------
$envFile = Join-Path $backend ".env"
if (-not (Test-Path $envFile)) {
    Copy-Item (Join-Path $backend ".env.example") $envFile
    Write-Host "Created backend/.env from .env.example"
}

# --- Postgres + Redis (compose --wait blocks until healthy) -----------
Write-Host "Starting Postgres + Redis..."
Push-Location $backend
cmd /c "docker compose up -d --wait"
$composeExit = $LASTEXITCODE
Pop-Location
if ($composeExit -ne 0) {
    throw "docker compose up failed (exit $composeExit)."
}

# --- app processes, each in its own window ----------------------------
function Start-DevWindow($title, $dir, $command) {
    $inner = "`$Host.UI.RawUI.WindowTitle = '$title'; Set-Location '$dir'; $command"
    Start-Process powershell -ArgumentList "-NoExit", "-Command", $inner
}

Start-DevWindow "ACMP api"    $backend  "& '$go' run ./cmd/api"
Start-DevWindow "ACMP worker" $backend  "& '$go' run ./cmd/worker"
Start-DevWindow "ACMP web"    $frontend "npm run dev"

Write-Host ""
Write-Host "Up:"
Write-Host "  API      http://localhost:8080/healthz"
Write-Host "  Frontend http://localhost:3000"
Write-Host ""
Write-Host "Stop: close the three 'ACMP *' windows, then  .\dev.ps1 -Down"
