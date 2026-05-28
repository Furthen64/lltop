$path = Join-Path $env:USERPROFILE ".config\lltop"

if (Test-Path $path) {
    Remove-Item $path -Recurse -Force
    Write-Output "Removed: $path"
} else {
    Write-Output "Folder not found: $path"
}