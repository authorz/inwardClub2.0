#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
SERVER_DIR="${REPO_ROOT}/server"
DIST_DIR="${REPO_ROOT}/dist"
TARGET_OS="linux"
TARGET_ARCH="amd64"
PACKAGE_LABEL="${1:-$(date '+%Y%m%d-%H%M%S')}"

if [[ ! "${PACKAGE_LABEL}" =~ ^[A-Za-z0-9._-]+$ ]]; then
  echo "错误：打包标签只能包含字母、数字、点、下划线和连字符。" >&2
  exit 1
fi

PACKAGE_NAME="inwardclub-server-${PACKAGE_LABEL}-${TARGET_OS}-${TARGET_ARCH}"
PACKAGE_DIR="${DIST_DIR}/${PACKAGE_NAME}"
ARCHIVE_PATH="${DIST_DIR}/${PACKAGE_NAME}.tar.gz"
ARCHIVE_CHECKSUM_PATH="${ARCHIVE_PATH}.sha256"

if [[ -e "${PACKAGE_DIR}" || -e "${ARCHIVE_PATH}" || -e "${ARCHIVE_CHECKSUM_PATH}" ]]; then
  echo "错误：产物 ${PACKAGE_NAME} 已存在，请换一个标签后重试。" >&2
  exit 1
fi

mkdir -p "${DIST_DIR}"
STAGING_DIR="$(mktemp -d "${DIST_DIR}/.package-staging.XXXXXX")"

cleanup() {
  if [[ -d "${STAGING_DIR}" ]]; then
    find "${STAGING_DIR}" -depth -delete
  fi
}
trap cleanup EXIT

STAGED_PACKAGE_DIR="${STAGING_DIR}/${PACKAGE_NAME}"
STAGED_BIN_DIR="${STAGED_PACKAGE_DIR}/bin"
mkdir -p "${STAGED_BIN_DIR}"

sha256_files() {
  if command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "$@"
  elif command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$@"
  else
    echo "错误：系统缺少 shasum 或 sha256sum。" >&2
    return 1
  fi
}

echo "[1/4] 运行服务端测试"
(
  cd "${SERVER_DIR}"
  go test ./...
)

echo "[2/4] 构建 ${TARGET_OS}/${TARGET_ARCH} 静态二进制"
(
  cd "${SERVER_DIR}"
  CGO_ENABLED=0 GOOS="${TARGET_OS}" GOARCH="${TARGET_ARCH}" \
    go build -trimpath -ldflags="-s -w" -o "${STAGED_BIN_DIR}/inwardclub-api" ./cmd/api
  CGO_ENABLED=0 GOOS="${TARGET_OS}" GOARCH="${TARGET_ARCH}" \
    go build -trimpath -ldflags="-s -w" -o "${STAGED_BIN_DIR}/inwardclub-worker" ./cmd/worker
  CGO_ENABLED=0 GOOS="${TARGET_OS}" GOARCH="${TARGET_ARCH}" \
    go build -trimpath -ldflags="-s -w" -o "${STAGED_BIN_DIR}/inwardclub-migrate" ./cmd/migrate
)

COMMIT_SHA="$(git -C "${REPO_ROOT}" rev-parse HEAD)"
WORKTREE_DIRTY="false"
if [[ -n "$(git -C "${REPO_ROOT}" status --porcelain --untracked-files=no)" ]]; then
  WORKTREE_DIRTY="true"
fi

cat > "${STAGED_PACKAGE_DIR}/BUILD_INFO.txt" <<EOF
package=${PACKAGE_NAME}
commit=${COMMIT_SHA}
builtAt=$(date -u '+%Y-%m-%dT%H:%M:%SZ')
goVersion=$(go version)
target=${TARGET_OS}/${TARGET_ARCH}
dirtyWorktree=${WORKTREE_DIRTY}
EOF

(
  cd "${STAGED_PACKAGE_DIR}"
  sha256_files \
    bin/inwardclub-api \
    bin/inwardclub-worker \
    bin/inwardclub-migrate > SHA256SUMS
)

echo "[3/4] 校验二进制格式"
for binary in "${STAGED_BIN_DIR}"/*; do
  binary_type="$(file "${binary}")"
  echo "${binary_type}"
  if [[ "${binary_type}" != *"ELF 64-bit LSB executable"* || "${binary_type}" != *"x86-64"* ]]; then
    echo "错误：${binary} 不是预期的 Linux AMD64 可执行文件。" >&2
    exit 1
  fi
done

mv "${STAGED_PACKAGE_DIR}" "${PACKAGE_DIR}"

echo "[4/4] 生成压缩包与校验和"
tar -czf "${ARCHIVE_PATH}" -C "${DIST_DIR}" "${PACKAGE_NAME}"
sha256_files "${ARCHIVE_PATH}" > "${ARCHIVE_CHECKSUM_PATH}"
tar -tzf "${ARCHIVE_PATH}" >/dev/null

echo
echo "打包完成"
echo "目录：${PACKAGE_DIR}"
echo "压缩包：${ARCHIVE_PATH}"
echo "SHA-256：$(awk '{print $1}' "${ARCHIVE_CHECKSUM_PATH}")"
