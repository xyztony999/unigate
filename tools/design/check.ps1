$ErrorActionPreference = 'Stop'
$contractCheck = Join-Path $PSScriptRoot 'check_contract.py'
& python -X utf8 $contractCheck @args
exit $LASTEXITCODE
