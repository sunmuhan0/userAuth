#!/bin/bash
# Promtail 一键安装脚本（Linux amd64）
# 使用方法：bash install-promtail.sh

set -e

PROMTAIL_VERSION="2.9.0"
PROMTAIL_URL="https://github.com/grafana/loki/releases/download/v${PROMTAIL_VERSION}/promtail-linux-amd64.zip"

echo "=== 1. 下载 Promtail ${PROMTAIL_VERSION} ==="
cd /tmp
wget -q "${PROMTAIL_URL}" -O promtail.zip
unzip -o promtail.zip
chmod +x promtail-linux-amd64
mv promtail-linux-amd64 /usr/local/bin/promtail

echo "=== 2. 创建配置目录 ==="
mkdir -p /etc/promtail
mkdir -p /var/lib/promtail
mkdir -p /home/work/log

echo "=== 3. 复制配置文件 ==="
cp promtail-config.yml /etc/promtail/config.yml

echo "=== 4. 安装 systemd 服务 ==="
cp promtail.service /etc/systemd/system/promtail.service
systemctl daemon-reload
systemctl enable promtail
systemctl start promtail

echo "=== 5. 检查状态 ==="
systemctl status promtail --no-pager

echo ""
echo "=== 安装完成 ==="
echo "配置文件：/etc/promtail/config.yml"
echo "日志采集目录：/home/work/log/**/*.log"
echo "查看日志：journalctl -u promtail -f"
echo ""
echo "注意：请修改 /etc/promtail/config.yml 中的 Loki 地址为实际生产地址"
