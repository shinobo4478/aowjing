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

if ($Down) {
    Push-Location $backend
    docker compose down
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
docker info *> $null
if ($LASTEXITCODE -ne 0) {
    throw "Docker Desktop isn't running. Start it (wait for the green whale) and retry."
}

# --- .env ---------------------------------------------------------------
$envFile = Join-Path $backend ".env"
if (-not (Test-Path $envFile)) {
    Copy-Item (Join-Path $backend ".env.example") $envFile
    Write-Host "Created backend/.env from .env.example"
}

# --- Postgres + Redis --------------------------------------------------
Write-Host "Starting Postgres + Redis..."
Push-Location $backend
docker compose up -d
if ($LASTEXITCODE -ne 0) { Pop-Location; throw "docker compose up failed." }

foreach ($svc in "backend-db-1", "backend-redis-1") {
    for ($i = 0; $i -lt 30; $i++) {
        if ((docker inspect -f '{{.State.Health.Status}}' $svc 2>$null) -eq "healthy") { break }
        Start-Sleep -Seconds 2
    }
}
Pop-Location

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
