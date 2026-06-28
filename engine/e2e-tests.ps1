@'
============================================================
  md2pdf — End-to-End Test Suite
  Tests: CLI binary, all fixtures, edge cases, config variants
============================================================
'@

function Write-Result {
    param($Label, $Ok, $Detail)
    $color = if ($Ok) { "Green" } else { "Red" }
    $icon  = if ($Ok) { "[PASS]" } else { "[FAIL]" }
    Write-Host ("{0,-8} {1,-35} {2}" -f $icon, $Label, $Detail) -ForegroundColor $color
}

$root = Split-Path -Parent $PSScriptRoot
$engine = Join-Path $root "engine"
$bin   = Join-Path $engine "md2pdf-test.exe"
$fixtures = Join-Path $engine "fixtures"

if (-not (Test-Path $bin)) {
    Write-Host "ERROR: CLI binary not found. Run go build first." -ForegroundColor Red
    exit 1
}

$total = 0
$passed = 0
$failed = 0

$results = @()

# ============================================================
# 1. VALID PDF OUTPUT — All fixtures
# ============================================================
Write-Host "`n===== 1. FIXTURE TESTS (valid PDF output) =====" -ForegroundColor Cyan

$allFixtures = Get-ChildItem -LiteralPath $fixtures -Filter "*.md" | Sort-Object Name

foreach ($f in $allFixtures) {
    $total++
    $name = $f.BaseName
    $out = Join-Path $env:TEMP "$name.pdf"
    $start = Get-Date
    & $bin $f.FullName $out 2>&1 | Out-Null
    $elapsed = [math]::Round(((Get-Date) - $start).TotalMilliseconds)

    if ($LASTEXITCODE -ne 0) {
        Write-Result $name $false "exit code $LASTEXITCODE"
        $failed++
        continue
    }

    if (-not (Test-Path $out)) {
        Write-Result $name $false "no output file"
        $failed++
        continue
    }

    $bytes = [System.IO.File]::ReadAllBytes($out)
    $header = [System.Text.Encoding]::ASCII.GetString($bytes, 0, 8)
    $isValid = $header -eq "%PDF-1.7" -or $header.StartsWith("%PDF")
    $sizeKb = [math]::Round($bytes.Length / 1024, 1)

    if ($isValid) {
        Write-Result $name $true "${sizeKb}KB in ${elapsed}ms"
        $passed++
    } else {
        Write-Result $name $false "invalid header: $header"
        $failed++
    }

    $results += @{
        Name = $name
        Valid = $isValid
        Size = $bytes.Length
        Time = $elapsed
    }

    Remove-Item -LiteralPath $out -ErrorAction SilentlyContinue
}

# ============================================================
# 2. CONFIGURATION VARIANTS
# ============================================================
Write-Host "`n===== 2. CONFIG VARIANT TESTS =====" -ForegroundColor Cyan

$comprehensiveMd = Join-Path $fixtures "comprehensive.md"
$variants = @(
    @{ label = "Letter/Normal/10pt";  opts = @{ pageSize="letter"; marginMm=25; baseFontPt=10; theme="light" } },
    @{ label = "A4/Narrow/12pt";     opts = @{ pageSize="a4";     marginMm=15; baseFontPt=12; theme="light" } },
    @{ label = "A4/Wide/11pt";       opts = @{ pageSize="a4";     marginMm=35; baseFontPt=11; theme="light" } },
    @{ label = "Letter/Narrow/11pt"; opts = @{ pageSize="letter"; marginMm=15; baseFontPt=11; theme="draft" } }
)

$variantIdx = 0
foreach ($v in $variants) {
    $total++
    $label = $v.label
    $variantIdx++
    $out = Join-Path $env:TEMP "variant-$variantIdx.pdf"
    $o = $v.opts
    & $bin --pageSize $o.pageSize --margin $o.marginMm --fontSize $o.baseFontPt --theme $o.theme $comprehensiveMd $out 2>&1 | Out-Null

    if ($LASTEXITCODE -ne 0) {
        Write-Result $label $false "exit $LASTEXITCODE"
        $failed++
        continue
    }

    $bytes = [System.IO.File]::ReadAllBytes($out)
    $header = [System.Text.Encoding]::ASCII.GetString($bytes, 0, 8)
    $isValid = $header -eq "%PDF-1.7"
    $sizeKb = [math]::Round($bytes.Length / 1024, 1)

    if ($isValid) {
        Write-Result $label $true "${sizeKb}KB valid PDF"
        $passed++
    } else {
        Write-Result $label $false "bad header: $header"
        $failed++
    }

    Remove-Item -LiteralPath $out -ErrorAction SilentlyContinue
}

# ============================================================
# 3. EDGE CASE TESTS (PRD Section 5.3)
# ============================================================
Write-Host "`n===== 3. PRD EDGE CASE TESTS =====" -ForegroundColor Cyan

function Test-Md {
    param($Label, $Markdown, [scriptblock]$Validate = { $args[0] -and $args[0].Length -gt 30 })
    $total++
    $out = Join-Path $env:TEMP "edge-test.pdf"
    $mdFile = Join-Path $env:TEMP "edge-test.md"
    $Markdown | Set-Content -LiteralPath $mdFile -Encoding ASCII -NoNewline
    & $bin $mdFile $out 2>&1 | Out-Null

    $ok = $LASTEXITCODE -eq 0
    $detail = "exit $LASTEXITCODE"
    if ($ok -and (Test-Path $out)) {
        $bytes = [System.IO.File]::ReadAllBytes($out)
        $header = [System.Text.Encoding]::ASCII.GetString($bytes, 0, 8)
        $ok = $header.StartsWith("%PDF")
        $detail = if ($ok) { "$([math]::Round($bytes.Length/1024,1))KB OK" } else { "bad header: $header" }
        if ($ok) {
            $ok = & $Validate $bytes
            if (-not $ok) { $detail = "validation failed" }
        }
    }

    if ($ok) { Write-Result $Label $true $detail; $passed++ }
    else     { Write-Result $Label $false $detail; $failed++ }

    Remove-Item -LiteralPath $out,$mdFile -ErrorAction SilentlyContinue
}

# Edge case: empty doc (no newlines)
Test-Md "Empty (truly empty)" ""

# Edge case: only a code block (fenced)
Test-Md "Only code block (fenced)" "``````powershell`r`nWrite-Host `"hi`"`r`n``````"

# Edge case: only a code block (indented)
Test-Md "Only code block (indented)" "    code line`r`n    another line"

# Edge case: 200-char URL
$longUrl = "https://example.com/" + ("x" * 180)
Test-Md "Long unbroken string" $longUrl

# Edge case: URL longer than page width
$veryLongUrl = "https://very-long-domain-name-that-exceeds-page-width.example.com/" + ("y" * 100)
Test-Md "Very long URL wrap" $veryLongUrl

# Edge case: consecutive headings with no spacing
Test-Md "Consecutive headings" "# H1`r`n## H2`r`n### H3"

# Edge case: list item with multiple paragraphs
Test-Md "List multi-para" "- First para.`r`n`r`n  Second para.`r`n`r`n- Another item"

# Edge case: deeply nested list (4+ levels)
Test-Md "Deep nesting (4 levels)" "- 1`r`n  - 2`r`n    - 3`r`n      - 4"

# Edge case: table with many columns
Test-Md "Wide table (7 cols)" "|A|B|C|D|E|F|G|`r`n|-|-|-|-|-|-|-|`r`n|1|2|3|4|5|6|7|"

# Edge case: mixed ordered/unordered
Test-Md "Mixed ordered/unordered" "1. One`r`n   - A`r`n   - B`r`n1. Two"

# Edge case: blockquote with list inside
Test-Md "Blockquote+list" "> - item in quote`r`n> - another item"

# Edge case: task list
Test-Md "Task list" "- [x] done`r`n- [ ] not done" 

# Edge case: typographic replacements
Test-Md "Typographic" "(c) 2024 -- test..."

# Edge case: autolink (bare URL)
Test-Md "Autolink" "Visit https://example.com/foo today"

# Edge case: strikethrough only
Test-Md "Strikethrough only" "~~struck~~"

# Edge case: bold+italic nested
Test-Md "Bold+Italic nested" "***bold and italic*** text"

# Edge case: image with remote URL
Test-Md "Remote image" "![img](https://example.com/image.png)"

# ============================================================
# 4. CONFIG CLI PARSING
# ============================================================
Write-Host "`n===== 4. CLI ARG PARSING =====" -ForegroundColor Cyan

$basicMd = Join-Path $fixtures "basic.md"
$cliTests = @(
    @{ args = @($basicMd, (Join-Path $env:TEMP "out.pdf"), "--pageSize", "letter"); label = "--pageSize letter" },
    @{ args = @($basicMd, (Join-Path $env:TEMP "out.pdf"), "--margin", "15");     label = "--margin 15" },
    @{ args = @($basicMd, (Join-Path $env:TEMP "out.pdf"), "--fontSize", "10");   label = "--fontSize 10" },
    @{ args = @($basicMd, (Join-Path $env:TEMP "out.pdf"), "--theme", "draft");   label = "--theme draft" }
)

foreach ($ct in $cliTests) {
    $total++
    & $bin $ct.args 2>&1 | Out-Null
    $ok = $LASTEXITCODE -eq 0 -and (Test-Path (Join-Path $env:TEMP "out.pdf"))
    $size = if ($ok) { "$([math]::Round((Get-Item (Join-Path $env:TEMP "out.pdf")).Length/1024,1))KB" } else { "FAIL" }
    Write-Result $ct.label $ok $size
    if ($ok) { $passed++ } else { $failed++ }
    Remove-Item (Join-Path $env:TEMP "out.pdf") -ErrorAction SilentlyContinue
}

# ============================================================
# 5. TIMING / BENCHMARK
# ============================================================
Write-Host "`n===== 5. TIMING (multi-page.md, 5 runs) =====" -ForegroundColor Cyan

$mpMd = Join-Path $fixtures "multi-page.md"
$times = @()
foreach ($i in 1..5) {
    $out = Join-Path $env:TEMP "timing.pdf"
    $start = Get-Date
    & $bin $mpMd $out 2>&1 | Out-Null
    $elapsed = ((Get-Date) - $start).TotalMilliseconds
    $times += $elapsed
    Remove-Item $out -ErrorAction SilentlyContinue
}
$avg = [math]::Round(($times | Measure-Object -Average).Average, 0)
Write-Host "  Average: ${avg}ms across 5 runs" -ForegroundColor Yellow

# ============================================================
# SUMMARY
# ============================================================
Write-Host "`n============================================================" -ForegroundColor Cyan
Write-Host "  RESULTS: $passed passed / $failed failed / $total total" -ForegroundColor $(if ($failed -eq 0) { "Green" } else { "Red" })
Write-Host "============================================================" -ForegroundColor Cyan

if ($failed -gt 0) {
    exit 1
}

# Optional: test the WASM binary via Node.js
Write-Host "`n===== (OPTIONAL) WASM TEST =====" -ForegroundColor Cyan
$nodeTest = Join-Path $engine "test-harness.js"
if ((Test-Path (Join-Path $engine "main.wasm")) -and (Get-Command "node" -ErrorAction SilentlyContinue)) {
    Write-Host "  Running WASM test harness..." -ForegroundColor DarkGray
    Push-Location $engine
    node test-harness.js 2>&1
    Pop-Location
} else {
    Write-Host "  SKIP: WASM binary or Node.js not available" -ForegroundColor DarkGray
}
