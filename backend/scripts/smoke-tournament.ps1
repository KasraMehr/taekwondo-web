param(
    [string]$BaseUrl = "http://localhost:8080",
    [string]$Email = "smoke-$([DateTimeOffset]::UtcNow.ToUnixTimeMilliseconds())@example.test",
    [string]$Password = "smoke-test-password"
)

$ErrorActionPreference = "Stop"

function Invoke-Json {
    param(
        [Parameter(Mandatory)][string]$Method,
        [Parameter(Mandatory)][string]$Path,
        [object]$Body,
        [string]$Token,
        [hashtable]$ExtraHeaders = @{}
    )
    $headers = @{}
    if ($Token) { $headers.Authorization = "Bearer $Token" }
    foreach ($key in $ExtraHeaders.Keys) { $headers[$key] = $ExtraHeaders[$key] }
    $parameters = @{
        Method = $Method
        Uri = "$BaseUrl$Path"
        Headers = $headers
    }
    if ($null -ne $Body) {
        $parameters.ContentType = "application/json; charset=utf-8"
        $parameters.Body = $Body | ConvertTo-Json -Depth 20 -Compress
    }
    Invoke-RestMethod @parameters
}

$health = Invoke-Json GET "/health"
if ($health.status -ne "up") { throw "API health check failed" }

$auth = Invoke-Json POST "/api/v1/auth/register" @{
    name = "Smoke Organizer"
    email = $Email
    password = $Password
}
$token = $auth.token

$organization = Invoke-Json POST "/api/v1/organizations" @{ name = "Smoke Tournament Organization" } $token
$orgId = $organization.id
$root = "/api/v1/organizations/$orgId"

$tournament = Invoke-Json POST "$root/tournaments" @{
    name = "Smoke Grand Prix"
    date = "2026-10-01"
    courts = 3
    gender = "male"
    ageCategory = "بزرگسالان"
    format = "grandPrix"
} $token
$tournamentId = $tournament.id
$tournamentPath = "$root/tournaments/$tournamentId"

$settings = $tournament.settings
$settings.draw.type = "ranked"
$settings.schedule.mode = "split_halves"
$settings.schedule.dayAssignment = "alternating"
$settings.schedule.days = @(
    @{ day = 1; date = "2026-10-01"; label = "روز اول" },
    @{ day = 2; date = "2026-10-02"; label = "روز دوم" }
)
$settings.numbering.startAt = 101
$settings.numbering.scope = "tournament"
$settings.numbering.order = "rounds"
$tournament = Invoke-Json PUT "$tournamentPath/settings" $settings $token @{ "If-Match" = '"' + $tournament.revision + '"' }

$categories = @("-54", "-58")
$rank = 1
foreach ($category in $categories) {
    foreach ($index in 1..4) {
        $profile = Invoke-Json POST "$root/athletes" @{
            name = "ورزشکار $category-$index"
            gender = "male"
        } $token
        $entry = Invoke-Json POST "$tournamentPath/athletes" @{
            profileId = $profile.id
            weightCategory = $category
            ranking = $rank
        } $token
        $limit = if ($category -eq "-54") { 54.0 } else { 58.0 }
        $null = Invoke-Json POST "$tournamentPath/athletes/$($entry.id)/weigh-in" @{
            weightKg = $limit
        } $token
        $rank++
    }
    $rank = 1
}

$drawn = Invoke-Json POST "$tournamentPath/draw" @{ type = "ranked" } $token
$readiness = Invoke-Json GET "$tournamentPath/draw-readiness" $null $token
if (-not $readiness.ready) { throw "Tournament is not ready for numbering" }

$preview = Invoke-Json POST "$tournamentPath/numbering/preview" $null $token
$numbered = Invoke-Json POST "$tournamentPath/numbering" $null $token @{ "If-Match" = '"' + $drawn.revision + '"' }

$playable = @($numbered.matches | Where-Object { -not $_.isBye })
$days = @($playable.day | Sort-Object -Unique)
$numbers = @($playable.matchNumber | Sort-Object)

[pscustomobject]@{
    Api = $BaseUrl
    Email = $Email
    OrganizationId = $orgId
    TournamentId = $tournamentId
    Revision = $numbered.revision
    AthleteCount = @($numbered.athletes).Count
    PlayableMatchCount = $playable.Count
    EventDays = $days -join ", "
    FirstMatchNumber = $numbers[0]
    LastMatchNumber = $numbers[-1]
    PreviewMatches = @($preview.matches).Count
} | Format-List

Write-Host "Smoke test passed." -ForegroundColor Green
