## OpenOcta 安装与快速访问指南

本文件位于打包产物的 `dist/README.md`，用于快速了解不同操作系统下如何安装、启动和访问 OpenOcta。

详细配置说明请参考归档中 `docs/` 目录下的文档，尤其是：

- `docs/configuration.md`：完整配置说明
- `docs/channels*.md`：各渠道配置
- `docs/tools*.md`：工具与集成

---

## 一、发行产物说明

在 `dist/` 目录中，你通常会看到：

- `openocta_<version>_linux_<arch>.tar.gz`：Linux 压缩包（GoReleaser）
- `openocta_<version>_linux_<arch>.deb` / `.rpm`：Linux 安装包
- `OpenOcta-<version>.dmg`：macOS 磁盘镜像（拖拽安装到「应用程序」）
- `OpenOcta-amd64-installer.exe`：Windows 安装器（Wails 构建，`./build.sh wails-nsis`）
- `openocta-docs_<version>.tar.gz`：仅包含文档的压缩包

Linux 压缩包包含：`openocta` 可执行文件、`docs/` 目录。Windows/macOS 使用 Wails 构建的桌面应用，以原生窗口运行。

---

## 二、Linux 安装与启动

### 1. 使用 .deb / .rpm 安装（推荐生产环境）

1. 安装包：
   - Debian / Ubuntu：
     - `sudo dpkg -i openocta_<version>_linux_amd64.deb`
   - RHEL / CentOS / Fedora：
     - `sudo rpm -ivh openocta_<version>_linux_amd64.rpm`
2. 安装后，systemd 服务会注册为 `openocta`，服务文件大致如下：
   - `/lib/systemd/system/openocta.service`
3. 安装后会自动：
   - `daemon-reload`
   - `enable openocta`（开机自启）
   - `start openocta`（立即启动）
4. 默认监听端口见配置文档（通常为 `gateway.port`，例如 18900）：
   - 在浏览器中访问 `http://<服务器IP>:<端口>` 即可。

### 2. 使用 tar.gz 手动运行（开发 / 测试）

1. 解压：
   - `tar -xzf openocta_<version>_linux_amd64.tar.gz`
2. 进入目录并运行：
   - `cd openocta_<version>_linux_amd64`
   - `chmod +x ./openocta`
   - `./openocta gateway run`
3. 终端会输出实际监听地址和端口，在浏览器中打开该地址即可快速访问。

---

## 三、macOS 安装与启动

### 方式一：OpenOcta.dmg（推荐）

1. **获取安装包**：从发布页下载 `OpenOcta-<version>.dmg`，或本地执行 `./deploy/macos/build-app.sh` 或 `make wails-dmg` 生成。
2. **打开映像**：双击 `.dmg` 会挂载只读卷（如 **`/Volumes/OpenOcta-Installer`**），Finder 打开窗口，内含 **OpenOcta**、指向本机「应用程序」的 **Applications** 快捷方式，以及 **安装说明.txt**。
3. **安装（任选其一）**：
   - **拖拽**：将 `OpenOcta` 拖到右侧「应用程序」文件夹图标上；或
   - **对话框安装**：直接双击窗口中的 `OpenOcta` 启动，若系统弹出「是否安装到『应用程序』」对话框，选 **「安装」** 并输入管理员密码，完成后会自动从 `/Applications` 重新启动（推荐首次从 DMG 运行时使用）。
4. **启动**：安装后在「应用程序」或启动台中双击 **OpenOcta**（英文名，无空格），以原生窗口打开（无需浏览器）。
5. **首次运行**：若提示「无法验证开发者」，请右键 → 打开 → 再次确认打开。

**找不到 OpenOcta？**

- 在 **访达 → 应用程序** 或按 **⌘⇧G** 输入 `/Applications`，查找 **OpenOcta.app**；启动台图标可能略有延迟。
- **勿混淆**：`/Volumes/...` 下的是 **DMG 挂载卷（只读）**，不是安装结果；安装后应用在 **`/Applications/OpenOcta.app`**。不要在 `/Volumes` 里用 `rm -rf` 强删，应在访达侧栏 **推出** 映像，或：`hdiutil detach "/Volumes/OpenOcta-Installer"`（卷名以 `ls /Volumes` 为准）。

### 方式二：命令行运行（需 Linux 压缩包）

1. 解压：`tar -xzf openocta_<version>_linux_amd64.tar.gz`（Linux 包在 macOS 上也可运行）
2. 运行：`cd openocta_<version>_linux_<arch>` → `chmod +x ./openocta` → `./openocta gateway run`
3. 在浏览器中访问终端输出的地址。

### 开机自启（可选）

如需开机自启，可安装 LaunchAgent（示例见 `deploy/macos/com.openocta.launcher.plist`）：

- `cp deploy/macos/com.openocta.launcher.plist ~/Library/LaunchAgents/`
- `launchctl load ~/Library/LaunchAgents/com.openocta.launcher.plist`

如需持久化运行（非 App 形态），也可自行编写 LaunchAgent/LaunchDaemon，ExecStart 命令与 `openocta.service` 类似：

- `openocta gateway run --port 18900`

---

## 四、Windows 安装与启动

### 方式一：安装器（推荐，Wails 桌面应用）

1. **获取安装包**：从发布页下载 `OpenOcta-amd64-installer.exe`，或本地执行 `./build.sh wails-nsis`（需在 Windows 上运行）生成。
2. **安装**：双击安装器 → 阅读版权声明并同意 → 选择安装目录 → 完成安装。
3. **启动**：从开始菜单或桌面快捷方式打开 `OpenOcta`，应用以原生窗口打开（无需浏览器）。

### 方式二：便携版（若有 .zip 压缩包）

1. 解压：`Expand-Archive .\openocta_<version>_windows_amd64.zip`
2. 运行：双击 `OpenOcta.exe` 或 `.\openocta.exe gateway run`
3. 在浏览器中访问终端输出的地址。

---

## 五、首次配置与快速上手

1. 默认配置文件路径：
   - `~/.openocta/openocta.json`
2. 建议：
   - 参考 `docs/configuration.md` 中的「快速开始」部分，复制示例 JSON 到 `~/.openocta/openocta.json`
   - 配置至少一个 `agent` 和一个 `channel`（例如 WhatsApp / Telegram / Slack 等）
3. 重启或重新运行 `openocta gateway run` 后生效。

## 大模型快速配置

这里以阿里云百炼模型为例，配置文件如下：
```shell
{
  "meta": {
    "lastTouchedVersion": "2026.2.9",
    "lastTouchedAt": "2026-02-11T09:19:35.523Z"
  },
  "auth": {
    "profiles": {
      "qwen-portal:default": {
        "provider": "qwen-portal",
        "mode": "oauth"
      }
    }
  },
  "skills": {
    "entries": {
      "prometheus": {
        "enabled": true
      }
    }
  },
  "plugins": {
    "entries": {
      "feishu": {
        "enabled": true
      },
      "qwen-portal-auth": {
        "enabled": true
      }
    }
  },
  "models": {
    "mode": "merge",
    "providers": {
      "bailian": {
        "baseUrl": "https://dashscope.aliyuncs.com/compatible-mode/v1",
        "apiKey": "xxxx",
        "api": "openai-completions",
        "models": [
          {
            "id": "qwen3.5-plus",
            "name": "qwen3.5-plus",
            "reasoning": false,
            "input": [
              "text"
            ],
            "cost": {
              "input": 0,
              "output": 0,
              "cacheRead": 0,
              "cacheWrite": 0
            },
            "contextWindow": 262144,
            "maxTokens": 65536
          }
        ]
      }
    }
  },
  "agents": {
    "defaults": {
      "model": {
        "primary": "bailian/qwen3.5-plus"
      },
      "maxConcurrent": 4,
      "subagents": {
        "maxConcurrent": 8
      }
    }
  },
  "messages": {
    "ackReactionScope": "group-mentions"
  },
  "commands": {
    "native": true,
    "nativeSkills": true,
    "bash": true
  },
  "channels": {
  },
  "hooks": {
    "enabled": true,
    "path": "/hooks",
    "token": "9fbe5742732ad7762201408cf22a97c559efc8106e616318"
  },
  "gateway": {
    "port": 18900,
    "mode": "local",
    "bind": "loopback",
    "auth": {
      "mode": "token",
      "token": "edc146993b5ae0b1544c3137cc888f94436cf11e1952cff6"
    },
    "tailscale": {
      "mode": "off",
      "resetOnExit": false
    }
  }
}
```

访问 `http://127.0.0.1:18900`，复制配置文件到配置框中，保存即可
<img src=".">


更多细节请阅读归档中的 `docs/` 文档或访问在线文档站点。

