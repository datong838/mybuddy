# Dify 本机起服（三端）

## Windows

```powershell
cd mybuddy-v01
powershell -File .\scripts\win\start-dockerd.ps1
powershell -File .\scripts\win\start-dify.ps1
# 打开 http://127.0.0.1/install
```

使用 `docker-compose.yaml` + `docker-compose.wsl-volumes.yaml`（WSL 命名卷）。

## macOS

```bash
cd mybuddy-v01/dify/docker
cp -n .env.example .env
# 安装并启动 Docker Desktop
docker compose up -d
```

或：`scripts/mac/start-dify.sh`

## Linux

```bash
cd mybuddy-v01/dify/docker
cp -n .env.example .env
sudo docker compose up -d
```

或：`scripts/linux/start-dify.sh`

## 上传语料

将 `docs/sample-corpus/` 下文本上传到知识库 `enterprise-kb`，系统提示参考 `docs/prompts/refuse-and-cite.md`。

## 故障：点「设置」后出现 This page couldn't load

**现象：** `POST /console/api/setup` 返回 500；API 日志 `PermissionDenied` 写 `privkeys/.../private.pem`。

**原因：** WSL 命名卷 `docker_dify_wsl_app` 属主为 root，API 进程用户为 `dify` (uid 1001) 无法写存储。

**修复（已写入 `scripts/win/start-dify.ps1`；也可手工执行）：**

```powershell
docker run --rm -v docker_dify_wsl_app:/storage alpine sh -c "chown -R 1001:1001 /storage; chmod -R u+rwX /storage"
```

然后用**系统浏览器**打开 http://127.0.0.1/install 再点「设置」（勿在桌面 WebView 里完成初始化）。库中若尚无管理员账号可直接重试；账号表为空时无需清库。
