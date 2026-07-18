#!/bin/sh
# RPM/DEB 安装后：重载 systemd，注册并启用/启动服务
if command -v systemctl >/dev/null 2>&1; then
  systemctl daemon-reload || true
  systemctl enable openocta || true
  systemctl start openocta || true
fi
