param (
    [string]$option
)

if ($option -eq "ai") {
    Set-Location backend\components\AIChat
    go run AIchat.go
} elseif ($option -eq "api") {
    Set-Location backend\ApiGateway
    go run ApiGateway.go
} else {
    Write-Output "Usage: ./run_backend.ps1 [ai|api]"
}
