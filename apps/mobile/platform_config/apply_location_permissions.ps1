# Applies T3.1.1 location permission fragments after `flutter create .`
# Usage: from apps/mobile → .\platform_config\apply_location_permissions.ps1

$ErrorActionPreference = 'Stop'
$root = Split-Path -Parent $PSScriptRoot

$androidManifest = Join-Path $root 'android\app\src\main\AndroidManifest.xml'
$iosPlist = Join-Path $root 'ios\Runner\Info.plist'

$androidSnippet = @'
    <!-- Location — Gas Tam Đệ T3.1.1 (when-in-use) -->
    <uses-permission android:name="android.permission.ACCESS_COARSE_LOCATION" />
    <uses-permission android:name="android.permission.ACCESS_FINE_LOCATION" />
'@

$iosSnippet = @'
	<key>NSLocationWhenInUseUsageDescription</key>
	<string>Gas Tam Đệ cần vị trí để xác định địa chỉ giao gas.</string>
'@

if (-not (Test-Path $androidManifest)) {
    Write-Warning "Missing $androidManifest — run: flutter create . --platforms=web,android,ios"
} elseif ((Get-Content $androidManifest -Raw) -match 'ACCESS_FINE_LOCATION') {
    Write-Host "Android: ACCESS_FINE_LOCATION already present."
} else {
    $xml = Get-Content $androidManifest -Raw
    if ($xml -notmatch '<manifest[\s\S]*?>') {
        throw "Could not find <manifest> in $androidManifest"
    }
    $xml = [regex]::Replace(
        $xml,
        '(<manifest[^>]*>)',
        "`$1`r`n$androidSnippet",
        1
    )
    Set-Content -Path $androidManifest -Value $xml -NoNewline
    Write-Host "Android: inserted location permissions into AndroidManifest.xml"
}

if (-not (Test-Path $iosPlist)) {
    Write-Warning "Missing $iosPlist — run: flutter create . --platforms=web,android,ios"
} elseif ((Get-Content $iosPlist -Raw) -match 'NSLocationWhenInUseUsageDescription') {
    Write-Host "iOS: NSLocationWhenInUseUsageDescription already present."
} else {
    $plist = Get-Content $iosPlist -Raw
    if ($plist -notmatch '</dict>\s*</plist>') {
        throw "Could not find closing </dict></plist> in $iosPlist"
    }
    $plist = [regex]::Replace(
        $plist,
        '</dict>\s*</plist>',
        "$iosSnippet`r`n</dict>`r`n</plist>",
        1
    )
    Set-Content -Path $iosPlist -Value $plist -NoNewline
    Write-Host "iOS: inserted NSLocationWhenInUseUsageDescription into Info.plist"
}

Write-Host "Web: no file patch needed (browser Geolocation / HTTPS or localhost)."
Write-Host "Done."
