param(
  [int]$AuthPort = 9091,
  [int]$ProcPort = 8080,
  [int]$ConfigPort = 7963,
  [string]$Env = "staging",
  [switch]$Loki,
  [switch]$NoBuild
)

$root = Split-Path -Parent $MyInvocation.MyCommand.Definition
$logDir = "$root\log"
$null = New-Item -ItemType Directory -Path $logDir -Force

function Log { param([string]$msg) Write-Host "[$(Get-Date -Format HH:mm:ss)] $msg" }

# ── 1. MySQL ──
function Start-MySQL {
  $existing = docker ps --filter name=mysql --format "{{.Names}}" 2>$null
  if ($existing -eq "mysql") { Log "MySQL already running"; return }
  Log "Starting MySQL..."
  docker rm -f mysql 2>$null
  docker run -d --name mysql -e MYSQL_ROOT_PASSWORD=123456 -e MYSQL_DATABASE=ttuser -p 3306:3306 mysql:8.0 --character-set-server=utf8mb4 --collation-server=utf8mb4_unicode_ci
  Start-Sleep -Seconds 10
  docker exec mysql mysql -u root -p123456 -e "SELECT 1" 2>&1 | Out-Null
  if ($LASTEXITCODE -eq 0) { Log "MySQL ready" } else { Log "MySQL failed"; exit 1 }
}

# ── 2. Loki Stack ──
function Start-Loki {
  Log "Starting Loki stack..."
  Set-Location "$root\deploy\loki"
  docker compose up -d 2>&1
  if ($LASTEXITCODE -eq 0) { Log "Loki stack ready" } else { Log "Loki stack failed"; exit 1 }
}

# ── 3. Build ──
function Build-All {
  $svcs = @(
    @{Name="config-server"; Dir="$root\config-server"},
    @{Name="auth-server";   Dir="$root\auth-server"},
    @{Name="proc";          Dir="$root\proc"}
  )
  foreach ($s in $svcs) {
    Log "Building $($s.Name)..."
    Set-Location $s.Dir
    $out = go build -o "$root\bin\$($s.Name).exe" ./cmd/server/ 2>&1
    if ($LASTEXITCODE -ne 0) { Log "Build $($s.Name) failed:`n$out"; exit 1 }
  }
  Log "All builds done"
}

# ── 4. Start services ──
$procs = @()
function Start-Service {
  param([string]$Name, [string]$ArgList, [int]$WaitSeconds = 3)
  $Exe = "$root\bin\$Name.exe"
  $logFile = "$logDir\$Name.log"
  Set-Location $root
  $p = Start-Process -NoNewWindow -FilePath $Exe -ArgumentList $ArgList `
    -WorkingDirectory $root `
    -RedirectStandardOutput "$logFile" -RedirectStandardError "$logDir\$Name.err.log" -PassThru
  $script:procs += $p
  Log "Started $Name (PID=$($p.Id)), log=$logFile"
  Start-Sleep -Seconds $WaitSeconds
  if ($p.HasExited) { Log "ERROR: $Name exited immediately, check $logFile"; $script:failed=1 }
}

function Start-All {
  $env:SERVICE_NAME   = "config-server"
  $env:SERVICE_PORT   = "$ConfigPort"
  $env:ENV            = $Env
  Start-Service -Name "config-server" `
    -ArgList "--name=config-server --port=$ConfigPort --env=$Env --config-dir=config-server\config-center" -WaitSeconds 4
  if ($failed) { exit 1 }

  $env:SERVICE_NAME   = "auth-server"
  $env:SERVICE_PORT   = "$AuthPort"
  Start-Service -Name "auth-server" `
    -ArgList "--name=auth-server --port=$AuthPort --env=$Env" -WaitSeconds 5
  if ($failed) { exit 1 }

  $env:SERVICE_NAME   = "proc"
  $env:SERVICE_PORT   = "$ProcPort"
  Start-Service -Name "proc" `
    -ArgList "--name=proc --port=$ProcPort --env=$Env" -WaitSeconds 3
}

# ── 5. Stop ──
function Stop-All {
  Log "Stopping all services..."
  foreach ($p in $procs) {
    if ($p -and !$p.HasExited) {
      Stop-Process -Id $p.Id -Force -ErrorAction SilentlyContinue
      Log "Stopped PID=$($p.Id)"
    }
  }
  if ($Loki) { Set-Location "$root\deploy\loki"; docker compose down 2>&1 | Out-Null }
}

# ── Main ──
try {
  Start-MySQL
  if ($Loki) { Start-Loki }
  if (-not $NoBuild) { Build-All }
  Start-All

  Write-Host "`n==== All services running ===="
  Write-Host "  config-server  http://127.0.0.1:$ConfigPort"
  Write-Host "  auth-server    grpc://127.0.0.1:$AuthPort  metrics http://127.0.0.1:$($AuthPort+100)"
  Write-Host "  proc           http://127.0.0.1:$ProcPort"
  if ($Loki) {
    Write-Host "  Grafana        http://127.0.0.1:3000  (admin/admin)"
    Write-Host "  Prometheus     http://127.0.0.1:9090"
    Write-Host "  Loki           http://127.0.0.1:3100"
  }
  Write-Host "Logs: $logDir"
  Write-Host "Press Ctrl+C to stop all.`n"

  Wait-Event -SourceIdentifier WaitEvent -Timeout ([System.Threading.Timeout]::Infinite)
} finally {
  Stop-All
}
