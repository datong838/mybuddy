# Apply Chinese UI labels to running Dify web console
# See: docs/palantier/10_v01/10g-交付面去Dify品牌说明.md

$ErrorActionPreference = "Stop"
$PatchJs = Join-Path $PSScriptRoot "whitelabel\aos-zh-ui-patch.js"

Write-Host "==> Rename workspace in DB"
docker exec docker-db_postgres-1 psql -U postgres -d dify -c "UPDATE tenants SET name = E'\u6816\u6708\u6c47\u7684\u5de5\u4f5c\u533a' WHERE id = '7ae527da-ab07-4f3c-9cce-148bae909b69'; SELECT id, name FROM tenants;"

Write-Host "==> Patch UI strings inside web container (node)"
$b64 = [Convert]::ToBase64String([System.IO.File]::ReadAllBytes($PatchJs))
$chunkSize = 6000
$tmp = "/tmp/_aos_zh_ui.js.b64"
docker exec docker-web-1 sh -c "rm -f $tmp /tmp/_aos_zh_ui.js"
for ($i = 0; $i -lt $b64.Length; $i += $chunkSize) {
  $len = [Math]::Min($chunkSize, $b64.Length - $i)
  $part = $b64.Substring($i, $len)
  docker exec docker-web-1 sh -c "printf %s '$part' >> $tmp"
}
docker exec docker-web-1 sh -c "base64 -d $tmp > /tmp/_aos_zh_ui.js && node /tmp/_aos_zh_ui.js"

Write-Host "==> Restart web"
docker restart docker-web-1 | Out-Null
Write-Host "Done. Hard-refresh http://127.0.0.1/apps"
