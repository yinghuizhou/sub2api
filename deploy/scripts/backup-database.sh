#!/bin/bash
# 数据库自动备份脚本
# 用法: ./backup-database.sh

set -e

# 配置
BACKUP_DIR="/opt/sub2api/backups"
RETENTION_DAYS=7
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
BACKUP_FILE="backup_${TIMESTAMP}.sql.gz"

# 创建备份目录
mkdir -p "$BACKUP_DIR"

# 执行备份
echo "[$(date)] 开始备份数据库..."
docker exec sub2api-postgres pg_dump -U sub2api sub2api | gzip > "${BACKUP_DIR}/${BACKUP_FILE}"

# 检查备份是否成功
if [ -f "${BACKUP_DIR}/${BACKUP_FILE}" ]; then
    SIZE=$(du -h "${BACKUP_DIR}/${BACKUP_FILE}" | cut -f1)
    echo "[$(date)] 备份成功: ${BACKUP_FILE} (${SIZE})"
else
    echo "[$(date)] 备份失败！"
    exit 1
fi

# 清理旧备份
echo "[$(date)] 清理 ${RETENTION_DAYS} 天前的备份..."
find "$BACKUP_DIR" -name "backup_*.sql.gz" -mtime +${RETENTION_DAYS} -delete

# 统计备份文件
BACKUP_COUNT=$(ls -1 "$BACKUP_DIR"/backup_*.sql.gz 2>/dev/null | wc -l)
echo "[$(date)] 当前保留 ${BACKUP_COUNT} 个备份文件"
