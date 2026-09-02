#!/bin/bash
set -e

# 1. 校验 Canal 账号配置，避免初始化出空用户名或空密码账号
if [[ -z "${CANAL_DB_USERNAME:-}" || -z "${CANAL_DB_PASSWORD:-}" ]]; then
    echo "Canal 数据库账号或密码未配置" >&2
    exit 1
fi
if [[ ! "${CANAL_DB_USERNAME}" =~ ^[A-Za-z0-9_]+$ ]]; then
    echo "Canal 数据库账号只能包含字母、数字和下划线" >&2
    exit 1
fi
if [[ "${CANAL_DB_PASSWORD}" == *"'"* ]]; then
    echo "Canal 数据库密码不能包含单引号" >&2
    exit 1
fi

# 2. 根据执行环境选择连接方式：MySQL 首次初始化走本地 Socket，独立任务走容器网络
mysql_args=(--protocol=tcp -h "${MYSQL_HOST:-mysql}")
if [[ -S /var/run/mysqld/mysqld.sock ]]; then
    mysql_args=(--protocol=socket)
fi

# 3. 创建 Canal 复制账号并授予读取 binlog 所需的最小权限
mysql "${mysql_args[@]}" -uroot -p"${MYSQL_ROOT_PASSWORD}" <<-EOSQL
    CREATE USER IF NOT EXISTS '${CANAL_DB_USERNAME}'@'%' IDENTIFIED BY '${CANAL_DB_PASSWORD}';
    ALTER USER '${CANAL_DB_USERNAME}'@'%' IDENTIFIED BY '${CANAL_DB_PASSWORD}';
    GRANT SELECT, REPLICATION SLAVE, REPLICATION CLIENT ON *.* TO '${CANAL_DB_USERNAME}'@'%';
    FLUSH PRIVILEGES;
EOSQL
