# Stage3 validation entrypoint
# Execute from PowerShell at repository root:
#   Get-Content .\scripts\validate_stage3.sh -Raw | Invoke-Expression

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$RootDir = (Get-Location).Path
if (-not (Test-Path (Join-Path $RootDir "backend"))) {
    throw "Run this script from the MRAG repository root."
}

$Stamp = [DateTimeOffset]::UtcNow.ToUnixTimeSeconds()
$ApiBase = if ($env:API_BASE) { $env:API_BASE } else { "http://127.0.0.1:18080/api/v1" }
$ProbeApiBase = if ($env:PROBE_API_BASE) { $env:PROBE_API_BASE } else { "http://127.0.0.1:18081/api/v1" }
$FrontendBase = if ($env:FRONTEND_BASE) { $env:FRONTEND_BASE } else { "http://127.0.0.1:4173" }
$LogsDir = Join-Path $RootDir ".stage3-logs"
$WorkspaceRootHost = Join-Path $RootDir "workspace"
$ValidationDir = Join-Path $RootDir "workspace\validation\stage3"
$RuntimeDir = Join-Path $ValidationDir "runtime"
$TemplateHostPath = Join-Path $WorkspaceRootHost "templates\stage3_validation_paper.md"
$GOCACHE = Join-Path $RootDir ".gocache"
$GOMODCACHE = Join-Path $RootDir ".gomodcache"
$DbHost = if ($env:POSTGRES_HOST) { $env:POSTGRES_HOST } else { "127.0.0.1" }
$DbPort = if ($env:POSTGRES_PORT) { $env:POSTGRES_PORT } else { "5432" }
$DbUser = if ($env:POSTGRES_USER) { $env:POSTGRES_USER } else { "postgres" }
$DbPassword = if ($env:POSTGRES_PASSWORD) { $env:POSTGRES_PASSWORD } else { "root" }
$DbName = if ($env:POSTGRES_DB) { $env:POSTGRES_DB } else { "mrag_stage3_validation" }
$PsqlPath = if ($env:PSQL_BIN) { $env:PSQL_BIN } else { "D:\Postgre\bin\psql.exe" }
$BackendExe = Join-Path $RootDir "backend\go\stage3-validation-server.exe"
$ProbeBackendLog = Join-Path $LogsDir "backend-probe-$Stamp.log"
$MainBackendLog = Join-Path $LogsDir "backend-main-$Stamp.log"
$BackendContainer = "mrag-stage3-backend-validation"
$ProbeContainer = "mrag-stage3-probe-validation"
$FrontendLog = Join-Path $LogsDir "frontend-$Stamp.log"
$FrontendErrLog = Join-Path $LogsDir "frontend-$Stamp.err.log"

$script:CurrentStep = "initializing"
$script:LastLogPath = ""
$script:FrontendPid = $null
$script:ProbeBackendPid = $null
$script:MainBackendPid = $null

New-Item -ItemType Directory -Force -Path $LogsDir, $ValidationDir, $RuntimeDir, (Split-Path $TemplateHostPath -Parent), $GOCACHE, $GOMODCACHE | Out-Null

function Cleanup-ValidationProcesses {
    Stop-BackgroundProcess $script:ProbeBackendPid
    Stop-BackgroundProcess $script:MainBackendPid
    Stop-BackgroundProcess $script:FrontendPid
    Stop-ValidationBackendProcesses
}

trap {
    $message = $_.Exception.Message
    Cleanup-ValidationProcesses
    Write-Output "FAIL: stage3 validation failed"
    Write-Output "- step: $script:CurrentStep"
    Write-Output "- message: $message"
    if ($script:LastLogPath -and (Test-Path $script:LastLogPath)) {
        Write-Output "- log: $script:LastLogPath"
        Write-Output "- log_tail:"
        Get-Content $script:LastLogPath -Tail 40 | ForEach-Object { Write-Output "  $_" }
    }
    exit 1
}

function Log([string]$Message) {
    Write-Output "[stage3] $Message"
}

function Set-Step([string]$Name) {
    $script:CurrentStep = $Name
    Log $Name
}

function Assert-True($Condition, [string]$Message) {
    if (-not $Condition) {
        throw $Message
    }
}

function Convert-WorkspacePathToHost([string]$PathValue) {
    if ([string]::IsNullOrWhiteSpace($PathValue)) {
        return ""
    }
    if ($PathValue.StartsWith("/app/workspace")) {
        $relative = $PathValue.Substring("/app/workspace".Length).TrimStart("/")
        if ([string]::IsNullOrWhiteSpace($relative)) {
            return $WorkspaceRootHost
        }
        return (Join-Path $WorkspaceRootHost ($relative -replace "/", "\"))
    }
    return $PathValue
}

function Get-HttpErrorDetail($ErrorRecord) {
    if ($null -eq $ErrorRecord) {
        return "unknown http error"
    }
    $parts = @()
    $exception = $ErrorRecord.Exception
    if ($null -ne $exception) {
        try {
            if ($null -ne $exception.Response -and $null -ne $exception.Response.StatusCode) {
                $parts += "status=$([int]$exception.Response.StatusCode)"
            }
        } catch {
        }
        if (-not [string]::IsNullOrWhiteSpace($exception.Message)) {
            $parts += $exception.Message
        }
    }
    $errorBody = ""
    try {
        $errorBody = [string]$ErrorRecord.ErrorDetails.Message
    } catch {
    }
    if ([string]::IsNullOrWhiteSpace($errorBody) -and $null -ne $exception -and $null -ne $exception.Response) {
        try {
            $stream = $exception.Response.GetResponseStream()
            if ($null -ne $stream) {
                $reader = New-Object System.IO.StreamReader($stream)
                try {
                    $errorBody = $reader.ReadToEnd()
                } finally {
                    $reader.Dispose()
                    $stream.Dispose()
                }
            }
        } catch {
        }
    }
    if (-not [string]::IsNullOrWhiteSpace($errorBody)) {
        $parts += "body=$errorBody"
    }
    if (@($parts).Count -eq 0) {
        return "unknown http error"
    }
    return ($parts -join " | ")
}

function Wait-Http([string]$Url, [int]$Retries = 90, [int]$SleepSeconds = 2) {
    for ($i = 0; $i -lt $Retries; $i++) {
        try {
            Invoke-WebRequest -UseBasicParsing -Uri $Url -Method Get -TimeoutSec 10 | Out-Null
            return
        } catch {
            Start-Sleep -Seconds $SleepSeconds
        }
    }
    throw "Timed out waiting for $Url"
}

function Invoke-ApiGet([string]$Path, [string]$BaseUrl = $ApiBase) {
    try {
        return Invoke-RestMethod -Method Get -Uri ($BaseUrl + $Path) -TimeoutSec 60
    } catch {
        throw "GET $Path failed: $(Get-HttpErrorDetail $_)"
    }
}

function Invoke-ApiPostJson([string]$Path, $Payload, [string]$BaseUrl = $ApiBase) {
    $json = $Payload | ConvertTo-Json -Depth 100 -Compress
    try {
        return Invoke-RestMethod -Method Post -Uri ($BaseUrl + $Path) -ContentType "application/json" -Body $json -TimeoutSec 120
    } catch {
        throw "POST $Path failed: $(Get-HttpErrorDetail $_)`nrequest=$json"
    }
}

function Stop-DockerContainer([string]$Name) {
    $null = & docker rm -f $Name 2>$null
    $global:LASTEXITCODE = 0
}

function Stop-BackgroundProcess([Nullable[int]]$PidValue) {
    if ($null -ne $PidValue -and $PidValue -gt 0) {
        Stop-Process -Id $PidValue -Force -ErrorAction SilentlyContinue
    }
}

function Stop-ValidationBackendProcesses {
    $validationExeName = [System.IO.Path]::GetFileName($BackendExe)
    Get-CimInstance Win32_Process -ErrorAction SilentlyContinue |
        Where-Object {
            $_.Name -eq $validationExeName -and
            [string]::Equals($_.ExecutablePath, $BackendExe, [System.StringComparison]::OrdinalIgnoreCase)
        } |
        ForEach-Object {
            Stop-Process -Id $_.ProcessId -Force -ErrorAction SilentlyContinue
        }
}

function Run-And-Log([string]$StepName, [scriptblock]$Action, [string]$LogName) {
    Set-Step $StepName
    $logPath = Join-Path $LogsDir $LogName
    $script:LastLogPath = $logPath
    if (Test-Path $logPath) {
        Remove-Item $logPath -Force
    }
    $global:LASTEXITCODE = 0
    $previousPreference = $global:ErrorActionPreference
    try {
        $global:ErrorActionPreference = "Continue"
        & $Action *> $logPath
    } finally {
        $global:ErrorActionPreference = $previousPreference
    }
    $exitCode = $LASTEXITCODE
    if ($null -ne $exitCode -and $exitCode -ne 0) {
        throw "$StepName failed with exit code $exitCode."
    }
}

function Stop-ProcessOnPort([int]$Port) {
    try {
        $connections = Get-NetTCPConnection -LocalPort $Port -State Listen -ErrorAction Stop | Select-Object -ExpandProperty OwningProcess -Unique
        foreach ($processId in $connections) {
            if ($processId -and $processId -ne 0) {
                Stop-Process -Id $processId -Force -ErrorAction SilentlyContinue
            }
        }
        return
    } catch {
        $netstat = netstat -ano | Select-String ":$Port"
        foreach ($line in $netstat) {
            $parts = ($line.ToString() -split "\s+") | Where-Object { $_ -ne "" }
            if ($parts.Length -ge 5) {
                $processId = [int]$parts[-1]
                if ($processId -gt 0) {
                    Stop-Process -Id $processId -Force -ErrorAction SilentlyContinue
                }
            }
        }
    }
}

function Get-AgentJob([string]$JobId) {
    return Invoke-ApiGet "/agents/jobs/$JobId"
}

function Assert-AgentJobHealthy([string]$JobId, [string]$AgentType) {
    $jobResp = Get-AgentJob $JobId
    Assert-True ($jobResp.data.agent_type -eq $AgentType) "Expected agent job $JobId to be of type $AgentType."
    Assert-True ($jobResp.data.status -eq "succeeded") "Expected $AgentType job $JobId to succeed, got $($jobResp.data.status)."
    Assert-True ($jobResp.data.validation_status -eq "succeeded") "Expected $AgentType job $JobId validation_status=succeeded."
    Assert-True ($jobResp.data.repair_status -ne "failed") "Expected $AgentType job $JobId repair_status to not be failed."
    return $jobResp
}

function New-RuntimeContract([string]$PathValue, [string]$ExecutionMode, [string]$WorkspaceDir, [string]$JobId) {
    $contract = @{
        job_id = $JobId
        agent_type = "reader"
        execution_mode = $ExecutionMode
        model_provider = "codex"
        model_name = "stage3-runtime-validation"
        prompt_version = "v1"
        input_refs = @()
        output_schema_ref = "schemas/reader-output-v1.json"
        skill_refs = @()
        tool_refs = @()
        memory_refs = @()
        workspace_dir = $WorkspaceDir
        metadata = @{
            research_direction = "stage3 validation retrieval"
            keywords = @("retrieval", "validation")
            source_scope = "arxiv"
            time_range = @{ year = 2026 }
            max_papers = 1
        }
    }
    $contract | ConvertTo-Json -Depth 50 | Set-Content -Encoding UTF8 -Path $PathValue
}

function Start-BackendProcess([int]$Port, [string]$SshMode, [string]$LogPath) {
    Stop-ValidationBackendProcesses
    Stop-ProcessOnPort $Port
    if (Test-Path $LogPath) {
        Remove-Item $LogPath -Force
    }
    $dsn = "postgres://$DbUser`:$DbPassword@$DbHost`:$DbPort/${DbName}?sslmode=disable"
    $backendGoDir = Join-Path $RootDir "backend\go"
    $cmd = @(
        "set `"APP_PORT=$Port`"",
        "set `"POSTGRES_DSN=$dsn`"",
        "set `"PYTHON_EXEC=python`"",
        "set `"PYTHON_AGENTS_DIR=$RootDir\backend\python_agents`"",
        "set `"PYTHON_TEMPLATES_DIR=$RootDir\backend\python_templates`"",
        "set `"WORKSPACE_ROOT=$WorkspaceRootHost`"",
        "set `"CODEX_CLI_BIN=definitely-missing-codex-cli`"",
        "set `"SSH_CLIENT_MODE=$SshMode`"",
        "set `"SSH_DIAL_TIMEOUT_SEC=4`"",
        "set `"SSH_COMMAND_TIMEOUT_SEC=20`"",
        "set `"REMOTE_EXECUTION_MODE=mock`"",
        "set `"REMOTE_WORK_ROOT=/tmp/mrag`"",
        "set `"REMOTE_RUNNER_ENTRYPOINT=./bin/mrag-remote-runner`"",
        "set `"REMOTE_DATASET_RUNNER_ENTRYPOINT=./bin/mrag-dataset-runner`"",
        "set `"DATASET_SCAN_MODE=mock`"",
        "set `"DATASET_INDEX_MODE=mock`"",
        "set `"OVERVIEW_STATS_MODE=mock`"",
        "set `"SERVER_HEARTBEAT_INTERVAL_SEC=0`"",
        "set `"GPU_SNAPSHOT_INTERVAL_SEC=0`"",
        "cd /d `"$backendGoDir`"",
        "`"$BackendExe`" 1> `"$LogPath`" 2>&1"
    ) -join " && "
    return Start-Process -FilePath "cmd.exe" -ArgumentList "/d", "/c", $cmd -WindowStyle Hidden -PassThru
}

Run-And-Log "go stage3 package tests" {
    Push-Location (Join-Path $RootDir "backend\go")
    try {
        $env:GOCACHE = $GOCACHE
        $env:GOMODCACHE = $GOMODCACHE
        go test -buildvcs=false ./internal/toolregistry ./internal/skillregistry ./internal/agentmemory ./internal/agentpipeline ./internal/readeragent ./internal/insightagent ./internal/datasetagent ./internal/ideaagent ./internal/planneragent ./internal/codingagent ./internal/writeragent ./internal/handler
    } finally {
        Pop-Location
    }
} "go-stage3-tests.log"

Run-And-Log "python stage3 runtime unit tests" {
    Push-Location $RootDir
    try {
        python -m unittest discover -s backend/python_agents/runtime/tests -p "test_*.py"
    } finally {
        Pop-Location
    }
} "python-stage3-runtime-tests.log"

Set-Step "runtime runner mock and codex fallback checks"
$runtimeMockDir = Join-Path $RuntimeDir "mock"
$runtimeCodexDir = Join-Path $RuntimeDir "codex_cli"
New-Item -ItemType Directory -Force -Path $runtimeMockDir, $runtimeCodexDir | Out-Null
$runtimeMockInput = Join-Path $runtimeMockDir "input.json"
$runtimeMockOutput = Join-Path $runtimeMockDir "output.json"
$runtimeCodexInput = Join-Path $runtimeCodexDir "input.json"
$runtimeCodexOutput = Join-Path $runtimeCodexDir "output.json"
New-RuntimeContract -PathValue $runtimeMockInput -ExecutionMode "mock" -WorkspaceDir $runtimeMockDir -JobId "stage3-runtime-mock"
New-RuntimeContract -PathValue $runtimeCodexInput -ExecutionMode "codex_cli" -WorkspaceDir $runtimeCodexDir -JobId "stage3-runtime-codex"

$script:LastLogPath = Join-Path $LogsDir "runtime-runner-mock.log"
Push-Location $RootDir
try {
    python backend/python_agents/runtime/runner.py --input $runtimeMockInput --output $runtimeMockOutput *> $script:LastLogPath
    if ($LASTEXITCODE -ne 0) {
        throw "runtime runner mock execution failed"
    }
    $env:CODEX_CLI_BIN = "definitely-missing-codex-cli"
    $script:LastLogPath = Join-Path $LogsDir "runtime-runner-codex.log"
    python backend/python_agents/runtime/runner.py --input $runtimeCodexInput --output $runtimeCodexOutput *> $script:LastLogPath
    if ($LASTEXITCODE -ne 0) {
        throw "runtime runner codex fallback execution failed"
    }
} finally {
    Remove-Item Env:CODEX_CLI_BIN -ErrorAction SilentlyContinue
    Pop-Location
}
$runtimeMockResult = Get-Content $runtimeMockOutput -Raw | ConvertFrom-Json
$runtimeCodexResult = Get-Content $runtimeCodexOutput -Raw | ConvertFrom-Json
Assert-True ($runtimeMockResult.status -eq "succeeded") "Expected runtime mock runner to succeed."
Assert-True ($runtimeMockResult.normalized_payload.execution_mode_used -eq "mock") "Expected runtime mock runner to use mock."
Assert-True ($runtimeCodexResult.status -eq "succeeded") "Expected runtime codex runner to succeed."
Assert-True ($runtimeCodexResult.normalized_payload.execution_mode_requested -eq "codex_cli") "Expected runtime codex runner to request codex_cli."
Assert-True ($runtimeCodexResult.normalized_payload.execution_mode_used -eq "mock") "Expected runtime codex runner to fall back to mock."
Assert-True ((@($runtimeCodexResult.warnings | Where-Object { $_ -match "falling back to mock executor" }) | Measure-Object).Count -ge 1) "Expected runtime codex runner to record fallback warning."

Run-And-Log "frontend typecheck" {
    Push-Location $RootDir
    try {
        npm.cmd run typecheck
    } finally {
        Pop-Location
    }
} "frontend-stage3-typecheck.log"

Set-Step "prepare local validation database"
Assert-True ((Test-NetConnection $DbHost -Port ([int]$DbPort)).TcpTestSucceeded) "Expected local PostgreSQL to be reachable at $DbHost`:$DbPort."
if (-not (Test-Path $PsqlPath)) {
    $PsqlPath = "psql"
}
$script:LastLogPath = Join-Path $LogsDir "postgres-prepare.log"
$env:PGPASSWORD = $DbPassword
try {
    $dbExists = & $PsqlPath -h $DbHost -p $DbPort -U $DbUser -d postgres -tAc "SELECT 1 FROM pg_database WHERE datname = '$DbName';" 2>&1
    if ($LASTEXITCODE -ne 0) {
        $dbExists | Set-Content -Encoding UTF8 -Path $script:LastLogPath
        throw "Failed to query PostgreSQL databases."
    }
    if (($dbExists | Out-String).Trim() -ne "1") {
        $createDbOutput = & $PsqlPath -h $DbHost -p $DbPort -U $DbUser -d postgres -c "CREATE DATABASE $DbName;" 2>&1
        $createDbOutput | Set-Content -Encoding UTF8 -Path $script:LastLogPath
        if ($LASTEXITCODE -ne 0) {
            throw "Failed to create validation database $DbName."
        }
    } else {
        "database $DbName already exists" | Set-Content -Encoding UTF8 -Path $script:LastLogPath
    }
} finally {
    Remove-Item Env:PGPASSWORD -ErrorAction SilentlyContinue
}

Run-And-Log "build local backend executable" {
    Push-Location (Join-Path $RootDir "backend\go")
    try {
        $env:GOCACHE = $GOCACHE
        $env:GOMODCACHE = $GOMODCACHE
        go build -buildvcs=false -o stage3-validation-server.exe ./cmd/server
    } finally {
        Pop-Location
    }
} "go-stage3-build.log"

Set-Step "start stage3 probe backend"
Stop-BackgroundProcess $script:ProbeBackendPid
$probeProcess = Start-BackendProcess -Port 18081 -SshMode "real" -LogPath $ProbeBackendLog
$script:ProbeBackendPid = $probeProcess.Id
$script:LastLogPath = $ProbeBackendLog
Wait-Http "http://127.0.0.1:18081/healthz" 120 2

Set-Step "probe shenzhenvlab availability"
$realProbeMode = "missing"
$realProbeMessage = "shenzhenvlab record not found"
$realServerId = ""
$realServerAvailableGPU = 0
try {
    $serversResp = Invoke-ApiGet "/servers" $ProbeApiBase
    $servers = @($serversResp.data)
    $shenzhen = $servers | Where-Object { $_.name -eq "shenzhenvlab" } | Select-Object -First 1
    if ($null -ne $shenzhen) {
        $realServerId = $shenzhen.id
        try {
            $heartbeatResp = Invoke-ApiPostJson "/servers/$realServerId/heartbeat" @{} $ProbeApiBase
            $gpuResp = Invoke-ApiPostJson "/servers/$realServerId/check-gpu" @{} $ProbeApiBase
            $realServerAvailableGPU = [int]$gpuResp.data.availableGpuCount
            if ($realServerAvailableGPU -gt 0) {
                $realProbeMode = "real_available"
                $realProbeMessage = "shenzhenvlab is reachable and has idle GPU."
            } else {
                $realProbeMode = "mock_fallback"
                $realProbeMessage = "shenzhenvlab is reachable but has no idle GPU."
            }
            $null = $heartbeatResp
        } catch {
            $realProbeMode = "mock_fallback"
            $realProbeMessage = "real probe failed: $($_.Exception.Message)"
        }
    }
} finally {
    Stop-BackgroundProcess $script:ProbeBackendPid
    $script:ProbeBackendPid = $null
}
Write-Output "[stage3] real_probe_result: $realProbeMode"
Write-Output "[stage3] real_probe_message: $realProbeMessage"

Set-Step "start main stage3 validation backend"
$mainProcess = Start-BackendProcess -Port 18080 -SshMode "mock" -LogPath $MainBackendLog
$script:MainBackendPid = $mainProcess.Id
$script:LastLogPath = $MainBackendLog
Wait-Http "http://127.0.0.1:18080/healthz" 120 2

Set-Step "start frontend dev server"
Stop-ProcessOnPort 4173
if (Test-Path $FrontendLog) { Remove-Item $FrontendLog -Force }
if (Test-Path $FrontendErrLog) { Remove-Item $FrontendErrLog -Force }
Assert-True (Test-Path (Join-Path $RootDir "dist\index.html")) "Expected dist/index.html to exist for frontend preview."
$frontendServerScript = Join-Path $ValidationDir "frontend_spa_server.py"
@"
from http.server import SimpleHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path
from urllib.parse import urlparse
import os
import sys

root = Path(sys.argv[1]).resolve()
port = int(sys.argv[2])
os.chdir(root)

class SPAHandler(SimpleHTTPRequestHandler):
    def do_GET(self):
        parsed = urlparse(self.path)
        route_path = parsed.path or "/"
        candidate = root / route_path.lstrip("/")
        if route_path == "/" or (not candidate.exists() and not Path(route_path).suffix):
            self.path = "/index.html"
        return super().do_GET()

server = ThreadingHTTPServer(("127.0.0.1", port), SPAHandler)
server.serve_forever()
"@ | Set-Content -Encoding UTF8 -Path $frontendServerScript
$frontendDistDir = Join-Path $RootDir "dist"
$frontendCmd = "cd /d `"$RootDir`" && python `"$frontendServerScript`" `"$frontendDistDir`" 4173 1> `"$FrontendLog`" 2> `"$FrontendErrLog`""
$frontendProc = Start-Process -FilePath "cmd.exe" -ArgumentList "/c", $frontendCmd -WindowStyle Hidden -PassThru
$script:FrontendPid = $frontendProc.Id
$script:LastLogPath = $FrontendErrLog
Wait-Http $FrontendBase 120 2

Set-Step "create validation mock server"
$script:LastLogPath = $MainBackendLog
$mockServerName = "stage3-mock-server-$Stamp"
$mockServerResp = Invoke-ApiPostJson "/servers" @{
    name = $mockServerName
    host = "mock-online-$Stamp"
    sshPort = 22
    username = "demo"
    authType = "ssh_config"
    remoteRoot = "/tmp/mrag"
    taskWorkdir = "/tmp/mrag/tasks"
    config = @{
        profile = "stage3-validation"
    }
}
$mockServerId = $mockServerResp.data.id
Assert-True (-not [string]::IsNullOrWhiteSpace($mockServerId)) "Expected validation mock server id."
$null = Invoke-ApiPostJson "/servers/$mockServerId/heartbeat" @{}
$null = Invoke-ApiPostJson "/servers/$mockServerId/gpu-snapshot" @{}
$preferredServerName = if ($realProbeMode -eq "real_available") { "shenzhenvlab" } else { $mockServerName }

Set-Step "register validation tool and skill"
$script:LastLogPath = $MainBackendLog
$toolRegisterResp = Invoke-ApiPostJson "/tools/register" @{
    name = "stage3 validation tool $Stamp"
    owner_agent_type = "reader"
    description = "Validation-only tool registration for stage3 acceptance."
    usage_md = "Invoke as a py_compile-safe placeholder during stage3 validation."
    input_schema = @{
        type = "object"
        properties = @{
            text = @{ type = "string" }
        }
    }
    output_schema = @{
        type = "object"
        properties = @{
            ok = @{ type = "boolean" }
        }
    }
    version = "v1"
    script_name = "validate_stage3_tool.py"
    script_content = "def validation_tool():`n    return {'ok': True}`n"
}
$toolId = $toolRegisterResp.data.tool_id
Assert-True (-not [string]::IsNullOrWhiteSpace($toolId)) "Expected tool registry to return tool_id."
$toolListResp = Invoke-ApiGet "/tools"
Assert-True ((@($toolListResp.data) | Where-Object { $_.tool_id -eq $toolId } | Measure-Object).Count -eq 1) "Expected registered tool to appear in tool registry list."

$skillRegisterResp = Invoke-ApiPostJson "/skills/register" @{
    name = "stage3 validation skill $Stamp"
    description = "Validation-only skill registration for stage3 acceptance."
    entrypoint = "SKILL.md"
    dependencies = @()
    entry_content = "# Stage3 Validation Skill`n`n- Used to verify persistent skill registration.`n"
}
$skillId = $skillRegisterResp.data.skill_id
Assert-True (-not [string]::IsNullOrWhiteSpace($skillId)) "Expected skill registry to return skill_id."
$skillListResp = Invoke-ApiGet "/skills"
Assert-True ((@($skillListResp.data) | Where-Object { $_.skill_id -eq $skillId } | Measure-Object).Count -eq 1) "Expected registered skill to appear in skill registry list."

Set-Step "prepare writer template"
@"
# Stage3 Validation Paper Template

## Abstract
{{abstract}}

## Introduction
{{introduction}}

## Method
{{method}}

## Experiments
{{experiments}}

## Conclusion
{{conclusion}}
"@ | Set-Content -Encoding UTF8 -Path $TemplateHostPath

Set-Step "reader agent minimal test"
$script:LastLogPath = $MainBackendLog
$readerResp = Invoke-ApiPostJson "/agents/reader/run" @{
    research_direction = "stage3 validation retrieval"
    keywords = @("retrieval", "validation")
    source_scope = "arxiv"
    max_papers = 1
    execution_mode = "codex_cli"
    model_provider = "codex"
    model_name = "stage3-reader-validation"
    prompt_version = "v1"
    skill_refs = @($skillId)
    tool_refs = @($toolId)
}
$readerJobId = $readerResp.data.job.id
Assert-True (-not [string]::IsNullOrWhiteSpace($readerJobId)) "Expected reader job id."
Assert-True (@($readerResp.data.imported_papers).Count -ge 1) "Expected reader agent to import at least one paper."
$paperId = $readerResp.data.imported_papers[0].result.paper.id
Assert-True (-not [string]::IsNullOrWhiteSpace($paperId)) "Expected reader result to contain paper id."
$readerJobResp = Assert-AgentJobHealthy $readerJobId "reader"
Assert-True ($readerJobResp.data.normalized_payload.execution_mode_requested -eq "codex_cli") "Expected reader execution_mode_requested=codex_cli."
Assert-True ($readerJobResp.data.normalized_payload.execution_mode_used -eq "mock") "Expected reader to fall back to mock."
$readerDetailResp = Invoke-ApiGet "/agents/reader/jobs/$readerJobId"
Assert-True (@($readerDetailResp.data.imported_papers).Count -ge 1) "Expected reader job detail imported papers."
$parsedContentRef = Join-Path $WorkspaceRootHost "papers\parsed\$paperId\parsed.md"
Assert-True (Test-Path (Convert-WorkspacePathToHost $parsedContentRef)) "Expected parsed paper markdown to exist."

Set-Step "insight agent minimal test"
$script:LastLogPath = $MainBackendLog
$insightResp = Invoke-ApiPostJson "/agents/insight/run" @{
    paper_id = $paperId
    parsed_content_ref = $parsedContentRef
    focus = "method"
    execution_mode = "codex_cli"
    model_provider = "codex"
    model_name = "stage3-insight-validation"
    prompt_version = "v1"
}
$insightJobId = $insightResp.data.job.id
$insightId = $insightResp.data.insight.id
Assert-True (-not [string]::IsNullOrWhiteSpace($insightJobId)) "Expected insight job id."
Assert-True (-not [string]::IsNullOrWhiteSpace($insightId)) "Expected insight id."
$insightJobResp = Assert-AgentJobHealthy $insightJobId "insight"
Assert-True ($insightJobResp.data.normalized_payload.execution_mode_used -eq "mock") "Expected insight to fall back to mock."
$insightSummaryHostPath = Convert-WorkspacePathToHost $insightResp.data.summary_path
Assert-True (Test-Path $insightSummaryHostPath) "Expected insight summary file to exist."

Set-Step "dataset agent minimal test"
$script:LastLogPath = $MainBackendLog
$datasetResp = Invoke-ApiPostJson "/agents/dataset/run" @{
    research_direction = "stage3 validation retrieval"
    task_type = "retrieval"
    keywords = @("retrieval", "benchmark")
    target_server_preference = $preferredServerName
    execution_mode = "codex_cli"
    model_provider = "codex"
    model_name = "stage3-dataset-validation"
    prompt_version = "v1"
}
$datasetJobId = $datasetResp.data.job.id
$datasetAssetId = $datasetResp.data.dataset_asset.asset.id
$baselineId = $datasetResp.data.baseline.baseline.id
$evalPlanPath = $datasetResp.data.eval_plan.evalplan_path
Assert-True (-not [string]::IsNullOrWhiteSpace($datasetJobId)) "Expected dataset job id."
Assert-True (-not [string]::IsNullOrWhiteSpace($datasetAssetId)) "Expected dataset asset id."
Assert-True (-not [string]::IsNullOrWhiteSpace($baselineId)) "Expected baseline id."
Assert-True (-not [string]::IsNullOrWhiteSpace($evalPlanPath)) "Expected eval plan path."
$datasetJobResp = Assert-AgentJobHealthy $datasetJobId "dataset"
Assert-True ($datasetJobResp.data.normalized_payload.execution_mode_used -eq "mock") "Expected dataset to fall back to mock."
$evalPlanResp = Invoke-ApiGet "/dataset-assets/$datasetAssetId/evalplan"
Assert-True ($evalPlanResp.data.eval_plan.dataset_asset_id -eq $datasetAssetId) "Expected evalplan endpoint to return the dataset asset."
Assert-True (Test-Path (Convert-WorkspacePathToHost $evalPlanPath)) "Expected dataset evalplan file to exist."
$memoryResp = Invoke-ApiGet "/agents/dataset/memory"
$datasetMemory = @($memoryResp.data) | Where-Object { $_.source_ref -eq "dataset_asset:$datasetAssetId" } | Select-Object -First 1
Assert-True ($null -ne $datasetMemory) "Expected dataset memory record for validation dataset asset."

Set-Step "idea generator agent minimal test"
$script:LastLogPath = $MainBackendLog
$ideaResp = Invoke-ApiPostJson "/agents/idea-generator/run" @{
    paper_insight_refs = @($insightId)
    dataset_asset_refs = @($datasetAssetId)
    human_hints = @("keep the validation pipeline small and controlled")
    execution_mode = "codex_cli"
    model_provider = "codex"
    model_name = "stage3-idea-validation"
    prompt_version = "v1"
}
$ideaJobId = $ideaResp.data.job.id
$ideaId = $ideaResp.data.idea.idea.id
Assert-True (-not [string]::IsNullOrWhiteSpace($ideaJobId)) "Expected idea generator job id."
Assert-True (-not [string]::IsNullOrWhiteSpace($ideaId)) "Expected idea id."
$ideaJobResp = Assert-AgentJobHealthy $ideaJobId "idea_generator"
Assert-True ($ideaJobResp.data.normalized_payload.execution_mode_used -eq "mock") "Expected idea generator to fall back to mock."

Set-Step "planner agent minimal test"
$script:LastLogPath = $MainBackendLog
$plannerResp = Invoke-ApiPostJson "/agents/planner/run" @{
    idea_id = $ideaId
    dataset_asset_refs = @($datasetAssetId)
    eval_protocol_refs = @($evalPlanPath)
    baseline_refs = @($baselineId)
    human_hints = @("prefer auditable template-bound experiments")
    execution_mode = "codex_cli"
    model_provider = "codex"
    model_name = "stage3-planner-validation"
    prompt_version = "v1"
}
$plannerJobId = $plannerResp.data.job.id
$experimentId = $plannerResp.data.experiment.experiment.id
$planPath = $plannerResp.data.plan.plan_path
Assert-True (-not [string]::IsNullOrWhiteSpace($plannerJobId)) "Expected planner job id."
Assert-True (-not [string]::IsNullOrWhiteSpace($experimentId)) "Expected planner to create experiment."
Assert-True (-not [string]::IsNullOrWhiteSpace($planPath)) "Expected planner plan path."
$plannerJobResp = Assert-AgentJobHealthy $plannerJobId "planner"
Assert-True ($plannerJobResp.data.normalized_payload.execution_mode_used -eq "mock") "Expected planner to fall back to mock."
$planResp = Invoke-ApiGet "/experiments/$experimentId/plan"
Assert-True ($planResp.data.plan.experiment_id -eq $experimentId) "Expected persisted experiment plan."
Assert-True (Test-Path (Convert-WorkspacePathToHost $planPath)) "Expected experiment plan file to exist."

Set-Step "coding evaluator agent minimal test"
$script:LastLogPath = $MainBackendLog
$codingResp = Invoke-ApiPostJson "/agents/coding/run" @{
    experiment_id = $experimentId
    train_template_ref = "mock_train_template"
    execution_mode = "codex_cli"
    model_provider = "codex"
    model_name = "stage3-coding-validation"
    prompt_version = "v1"
}
$codingJobId = $codingResp.data.job.id
$runId = $codingResp.data.run.id
Assert-True (-not [string]::IsNullOrWhiteSpace($codingJobId)) "Expected coding job id."
Assert-True (-not [string]::IsNullOrWhiteSpace($runId)) "Expected coding/evaluator run id."
$codingJobResp = Assert-AgentJobHealthy $codingJobId "coding"
Assert-True ($codingJobResp.data.normalized_payload.execution_mode_used -eq "mock") "Expected coding agent to fall back to mock."
Assert-True (-not [string]::IsNullOrWhiteSpace([string]$codingJobResp.data.normalized_payload.result_archive_id)) "Expected coding/evaluator result archive id."
$resultArchiveId = [string]$codingJobResp.data.normalized_payload.result_archive_id
Assert-True ($codingResp.data.run.runStatus -eq "succeeded") "Expected coding/evaluator run to succeed."
Assert-True (@($codingResp.data.patch_manifest).Count -ge 1) "Expected coding agent patch manifest."
$comparisonListResp = Invoke-ApiGet "/experiments/$experimentId/comparisons"
Assert-True (@($comparisonListResp.data).Count -ge 1) "Expected coding/evaluator comparison records."

Set-Step "writer agent minimal test"
$script:LastLogPath = $MainBackendLog
$writerResp = Invoke-ApiPostJson "/agents/writer/run" @{
    paper_template_ref = $TemplateHostPath
    idea_refs = @($ideaId)
    experiment_result_refs = @($runId)
    comparison_refs = @($experimentId)
    citation_refs = @("paper:$paperId")
    execution_mode = "codex_cli"
    model_provider = "codex"
    model_name = "stage3-writer-validation"
    prompt_version = "v1"
}
$writerJobId = $writerResp.data.job.id
$draftId = $writerResp.data.draft.draft_id
$draftPath = $writerResp.data.draft.draft_path
Assert-True (-not [string]::IsNullOrWhiteSpace($writerJobId)) "Expected writer job id."
Assert-True (-not [string]::IsNullOrWhiteSpace($draftId)) "Expected draft id."
$writerJobResp = Assert-AgentJobHealthy $writerJobId "writer"
Assert-True ($writerJobResp.data.normalized_payload.execution_mode_used -eq "mock") "Expected writer to fall back to mock."
$draftResp = Invoke-ApiGet "/drafts/$draftId"
Assert-True ($draftResp.data.draft_id -eq $draftId) "Expected draft retrieval endpoint to return validation draft."
Assert-True (Test-Path (Convert-WorkspacePathToHost $draftPath)) "Expected draft json to exist."

Set-Step "agent admin and frontend accessibility checks"
$script:LastLogPath = $MainBackendLog
$agentListResp = Invoke-ApiGet "/agents"
$jobListResp = Invoke-ApiGet "/agents/jobs?limit=100"
$eventListResp = Invoke-ApiGet "/agent-events?limit=100"
Assert-True (@($agentListResp.data).Count -ge 6) "Expected agent admin list to expose controlled agents."
Assert-True (@($jobListResp.data).Count -ge 7) "Expected agent admin job list to contain validation jobs."
Assert-True (@($eventListResp.data).Count -ge 1) "Expected agent event list to contain pipeline events."

foreach ($route in @("/", "/agents", "/agents/jobs", "/agents/catalog", "/agents/events")) {
    $script:LastLogPath = $FrontendErrLog
    $response = Invoke-WebRequest -UseBasicParsing -Uri ($FrontendBase + $route) -Method Get -TimeoutSec 30
    Assert-True ($response.StatusCode -eq 200) "Expected frontend route $route to return HTTP 200."
}

$summary = [ordered]@{
    reader_job_id = $readerJobId
    paper_id = $paperId
    insight_job_id = $insightJobId
    insight_id = $insightId
    dataset_job_id = $datasetJobId
    dataset_asset_id = $datasetAssetId
    dataset_memory_id = $datasetMemory.id
    idea_job_id = $ideaJobId
    idea_id = $ideaId
    planner_job_id = $plannerJobId
    experiment_id = $experimentId
    coding_job_id = $codingJobId
    run_id = $runId
    result_archive_id = $resultArchiveId
    writer_job_id = $writerJobId
    draft_id = $draftId
    tool_id = $toolId
    skill_id = $skillId
    shenzhenvlab_probe = $realProbeMode
    shenzhenvlab_probe_message = $realProbeMessage
}
$summaryPath = Join-Path $ValidationDir "stage3_validation_summary.json"
$summary | ConvertTo-Json -Depth 20 | Set-Content -Encoding UTF8 -Path $summaryPath

Cleanup-ValidationProcesses
Write-Output "PASS: stage3 validation passed"
Write-Output "- runtime_runner_mock: succeeded"
Write-Output "- runtime_runner_codex_fallback: succeeded"
Write-Output "- schema_validator_repair: succeeded"
Write-Output "- tool_registry: $toolId"
Write-Output "- skill_registry: $skillId"
Write-Output "- memory_store: $($datasetMemory.id)"
Write-Output "- reader_job_id: $readerJobId"
Write-Output "- insight_job_id: $insightJobId"
Write-Output "- dataset_job_id: $datasetJobId"
Write-Output "- idea_job_id: $ideaJobId"
Write-Output "- planner_job_id: $plannerJobId"
Write-Output "- coding_job_id: $codingJobId"
Write-Output "- writer_job_id: $writerJobId"
Write-Output "- experiment_id: $experimentId"
Write-Output "- run_id: $runId"
Write-Output "- draft_id: $draftId"
Write-Output "- shenzhenvlab_probe: $realProbeMode"
Write-Output "- summary_path: $summaryPath"
