param(
    [string]$ApiUrl = "http://localhost:8080"
)

Write-Host "Running SDK smoke test against $ApiUrl"

Push-Location "$PSScriptRoot\..\examples"
try {
    go mod tidy
    go build .
    Write-Host "Smoke build OK"
}
finally {
    Pop-Location
}
