#!/usr/bin/env sh
set -eu

ROOT=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)
cd "$ROOT"

if [ -z "${GOCACHE:-}" ]; then
	GOCACHE="$ROOT/.go-build"
fi
mkdir -p "$GOCACHE"
export GOCACHE

FORBIDDEN='github.com/gin-gonic/gin github.com/gin-contrib gorm.io/gorm github.com/redis/go-redis go.mongodb.org/mongo-driver github.com/segmentio/kafka-go github.com/minio/minio-go google.golang.org/grpc'

check_layer() {
	layer="$1"
	packages=$(go list "./$layer/..." 2>/dev/null || true)
	if [ -z "$packages" ]; then
		echo "[skip] $layer 目录尚未建立"
		return 0
	fi

	failed=0
	for pkg in $packages; do
		deps=$(go list -f '{{join .Imports "\n"}}' "$pkg")
		for dep in $deps; do
			for banned in $FORBIDDEN; do
				case "$dep" in
					"$banned" | "$banned"/*)
						echo "架构违规: $pkg 依赖禁止包 $dep"
						failed=1
						;;
				esac
			done
		done
	done
	return "$failed"
}

status=0
check_layer internal/domain || status=1
check_layer internal/application || status=1

if [ "$status" -eq 0 ]; then
	echo "架构依赖检查通过"
fi
exit "$status"
