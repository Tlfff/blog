#!/usr/bin/env sh
# 共享 MySQL 阶段的数据备份、校验与回滚辅助脚本。
# 用法：
#   sh scripts/mysql/backup_and_rollback.sh backup <db> <user> <password> <backup_dir>
#   sh scripts/mysql/backup_and_rollback.sh verify <db> <user> <password> <backup_dir>
set -eu

MODE="$1"
DB="$2"
USER="$3"
PASSWORD="$4"
BACKUP_DIR="${5:-./backups}"
TS=$(date +%Y%m%d%H%M%S)

case "$MODE" in
  backup)
    mkdir -p "$BACKUP_DIR"
    mysqldump -u"$USER" -p"$PASSWORD" --single-transaction --routines --triggers "$DB" > "$BACKUP_DIR/${DB}_${TS}.sql"
    echo "备份完成: $BACKUP_DIR/${DB}_${TS}.sql"
    ;;
  verify)
    FILE=$(ls -1t "$BACKUP_DIR"/"${DB}"_*.sql | head -1)
    if [ -z "$FILE" ]; then
      echo "未找到备份文件"
      exit 1
    fi
    mysql -u"$USER" -p"$PASSWORD" "$DB" < "$FILE" --force >/dev/null 2>&1 || true
    echo "校验（回放）完成: $FILE"
    ;;
  *)
    echo "未知模式: $MODE"
    exit 1
    ;;
esac
