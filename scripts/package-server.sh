#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
SERVER_DIR="${REPO_ROOT}/server"
ADMIN_CONSOLE_DIR="${REPO_ROOT}/admin-console"
STORE_CONSOLE_DIR="${REPO_ROOT}/store-console"
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
STAGED_ADMIN_CONSOLE_DIR="${STAGED_PACKAGE_DIR}/web/admin-console"
STAGED_STORE_CONSOLE_DIR="${STAGED_PACKAGE_DIR}/web/store-console"
mkdir -p "${STAGED_BIN_DIR}" "${STAGED_ADMIN_CONSOLE_DIR}" "${STAGED_STORE_CONSOLE_DIR}"

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

echo "[1/6] 运行服务端测试"
(
  cd "${SERVER_DIR}"
  go test ./...
)

echo "[2/6] 构建总后台和门店后台"
(
  cd "${ADMIN_CONSOLE_DIR}"
  npm run build
)
(
  cd "${STORE_CONSOLE_DIR}"
  npm run build
)

echo "[3/6] 构建 ${TARGET_OS}/${TARGET_ARCH} 静态二进制"
(
  cd "${SERVER_DIR}"
  CGO_ENABLED=0 GOOS="${TARGET_OS}" GOARCH="${TARGET_ARCH}" \
    go build -trimpath -ldflags="-s -w" -o "${STAGED_BIN_DIR}/inwardclub-api" ./cmd/api
  CGO_ENABLED=0 GOOS="${TARGET_OS}" GOARCH="${TARGET_ARCH}" \
    go build -trimpath -ldflags="-s -w" -o "${STAGED_BIN_DIR}/inwardclub-worker" ./cmd/worker
  CGO_ENABLED=0 GOOS="${TARGET_OS}" GOARCH="${TARGET_ARCH}" \
    go build -trimpath -ldflags="-s -w" -o "${STAGED_BIN_DIR}/inwardclub-migrate" ./cmd/migrate
)

echo "[4/6] 复制后台静态文件"
cp -R "${ADMIN_CONSOLE_DIR}/dist/." "${STAGED_ADMIN_CONSOLE_DIR}/"
cp -R "${STORE_CONSOLE_DIR}/dist/." "${STAGED_STORE_CONSOLE_DIR}/"

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
adminConsole=web/admin-console
storeConsole=web/store-console
EOF

(
  cd "${STAGED_PACKAGE_DIR}"
  find BUILD_INFO.txt bin web -type f | LC_ALL=C sort | while IFS= read -r packaged_file; do
    sha256_files "${packaged_file}"
  done > SHA256SUMS
)

echo "[5/6] 校验二进制格式和后台入口"
for binary in "${STAGED_BIN_DIR}"/*; do
  binary_type="$(file "${binary}")"
  echo "${binary_type}"
  if [[ "${binary_type}" != *"ELF 64-bit LSB executable"* || "${binary_type}" != *"x86-64"* ]]; then
    echo "错误：${binary} 不是预期的 Linux AMD64 可执行文件。" >&2
    exit 1
  fi
done
for frontend_entry in \
  "${STAGED_ADMIN_CONSOLE_DIR}/index.html" \
  "${STAGED_STORE_CONSOLE_DIR}/index.html"; do
  if [[ ! -s "${frontend_entry}" ]]; then
    echo "错误：后台入口 ${frontend_entry} 不存在或为空。" >&2
    exit 1
  fi
done

mv "${STAGED_PACKAGE_DIR}" "${PACKAGE_DIR}"

echo "[6/6] 生成压缩包与校验和"
tar -czf "${ARCHIVE_PATH}" -C "${DIST_DIR}" "${PACKAGE_NAME}"
sha256_files "${ARCHIVE_PATH}" > "${ARCHIVE_CHECKSUM_PATH}"
tar -tzf "${ARCHIVE_PATH}" >/dev/null

echo
echo "打包完成"
echo "目录：${PACKAGE_DIR}"
echo "压缩包：${ARCHIVE_PATH}"
echo "SHA-256：$(awk '{print $1}' "${ARCHIVE_CHECKSUM_PATH}")"
