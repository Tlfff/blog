#!/usr/bin/env sh
set -eu

ROOT=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)
cd "$ROOT"

contexts="user article comment like notification"
forbidden_domain_app="github.com/gin-gonic/gin github.com/gin-contrib gorm.io/gorm gorm.io/driver github.com/redis/go-redis go.mongodb.org/mongo-driver github.com/segmentio/kafka-go github.com/minio/minio-go google.golang.org/grpc"
forbidden_context_interfaces="gorm.io/gorm gorm.io/driver github.com/redis/go-redis go.mongodb.org/mongo-driver github.com/segmentio/kafka-go github.com/minio/minio-go"
status=0

# check_imports 检查指定目录是否导入禁止包。
check_imports() {
	dir="$1"
	banned_list="$2"
	[ -d "$dir" ] || return 0
	for file in $(find "$dir" -type f -name '*.go' ! -name '*_test.go'); do
		for banned in $banned_list; do
			if grep -Eq "^[[:space:]]*\"$banned(/|\")" "$file"; then
				echo "架构违规: $file 依赖禁止包 $banned"
				status=1
			fi
		done
	done
}

# 1. 检查五个上下文的层级依赖。
for context in $contexts; do
	check_imports "internal/$context/domain" "$forbidden_domain_app"
	check_imports "internal/$context/application" "$forbidden_domain_app"
	check_imports "internal/$context/interfaces" "$forbidden_context_interfaces"

	for layer in domain application infrastructure interfaces; do
		dir="internal/$context/$layer"
		[ -d "$dir" ] || continue
		for file in $(find "$dir" -type f -name '*.go' ! -name '*_test.go'); do
			for other in $contexts; do
				[ "$other" = "$context" ] && continue
				if grep -Eq "^[[:space:]]*\"blog/internal/$other/(infrastructure|interfaces)(/|\")" "$file"; then
					echo "架构违规: $file 直接依赖 $other 的 Infrastructure 或 Interfaces"
					status=1
				fi
			done
		done
	done

done

# 2. 检查旧全局技术分层目录已经删除。
for legacy in auth common consts cron dto grpc handler middleware model mq repository routes service; do
	if [ -d "internal/$legacy" ]; then
		echo "架构违规: 旧全局目录 internal/$legacy 仍然存在"
		status=1
	fi
done

# 3. 检查内部技术代码不再通过旧 pkg 路径引用。
if [ -d pkg ]; then
	echo "架构违规: 旧 pkg 目录仍然存在"
	status=1
fi
if grep -R -n --include='*.go' '"blog/pkg/' cmd internal >/dev/null 2>&1; then
	echo "架构违规: 业务代码仍然引用 blog/pkg 路径"
	status=1
fi
if grep -R -n --include='*.go' '"blog/config"' cmd internal >/dev/null 2>&1; then
	echo "架构违规: 业务代码仍然引用旧根 config 包"
	status=1
fi

if [ "$status" -ne 0 ]; then
	exit "$status"
fi

echo "架构依赖检查通过"
