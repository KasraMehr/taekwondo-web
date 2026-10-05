#!/usr/bin/env bash
set -Eeuo pipefail

# Run on the Ubuntu server as root:
#   sudo bash /opt/tkdhub/scripts/deploy.sh
#
# Optional overrides:
#   sudo DEPLOY_BRANCH=master FRONTEND_DIR=/var/www/tkdhub \
#     bash /opt/tkdhub/scripts/deploy.sh

APP_DIR="${APP_DIR:-/opt/tkdhub}"
DEPLOY_BRANCH="${1:-${DEPLOY_BRANCH:-balanced-courts-bulk-weighin}}"
SERVICE_NAME="${SERVICE_NAME:-tkdhub}"
FRONTEND_DIR="${FRONTEND_DIR:-/var/www/tkdhub}"
HEALTH_URL="${HEALTH_URL:-http://127.0.0.1:8080/health}"
GO_IMAGE="${GO_IMAGE:-golang:1.27.1}"
NODE_IMAGE="${NODE_IMAGE:-node:22-alpine}"

if [[ ${EUID} -ne 0 ]]; then
  echo "Run this script as root (use sudo)." >&2
  exit 1
fi

for command in git docker curl systemctl tar; do
  command -v "$command" >/dev/null 2>&1 || {
    echo "Required command is missing: $command" >&2
    exit 1
  }
done

[[ -d "$APP_DIR/.git" ]] || {
  echo "Git checkout not found at $APP_DIR" >&2
  exit 1
}

git check-ref-format --branch "$DEPLOY_BRANCH" >/dev/null
cd "$APP_DIR"

if [[ -n "$(git status --porcelain)" ]]; then
  echo "Deployment stopped: $APP_DIR has uncommitted changes." >&2
  exit 1
fi

echo "Fetching origin/$DEPLOY_BRANCH ..."
git fetch --prune origin "$DEPLOY_BRANCH"
if git show-ref --verify --quiet "refs/heads/$DEPLOY_BRANCH"; then
  git switch "$DEPLOY_BRANCH"
  git merge --ff-only "origin/$DEPLOY_BRANCH"
else
  git switch --track -c "$DEPLOY_BRANCH" "origin/$DEPLOY_BRANCH"
fi

build_dir="$(mktemp -d /tmp/tkdhub-deploy.XXXXXX)"
frontend_new="${FRONTEND_DIR}.new.$$"
frontend_previous="${FRONTEND_DIR}.previous.$$"
api_path="$APP_DIR/bin/api"
api_previous="$APP_DIR/bin/api.previous.$$"

cleanup() {
  rm -rf "$build_dir" "$frontend_new"
}
trap cleanup EXIT

echo "Building backend ..."
mkdir -p "$build_dir/bin" "$APP_DIR/bin"
docker run --rm \
  -v "$APP_DIR/backend:/src:ro" \
  -v "$build_dir/bin:/out" \
  -w /src \
  -e CGO_ENABLED=0 \
  "$GO_IMAGE" \
  sh -c 'go build -trimpath -o /out/api ./cmd/api && go build -trimpath -o /out/create-admin ./cmd/create-admin'

echo "Building frontend ..."
mkdir -p "$build_dir/frontend"
tar -C "$APP_DIR/frontend" \
  --exclude='./node_modules' --exclude='./dist' \
  -cf - . | tar -C "$build_dir/frontend" -xf -
docker run --rm \
  -v "$build_dir/frontend:/app" \
  -w /app \
  "$NODE_IMAGE" \
  sh -c 'npm ci && npm run build'

echo "Installing release ..."
install -m 0755 "$build_dir/bin/create-admin" "$APP_DIR/bin/create-admin"
install -d "$frontend_new"
cp -a "$build_dir/frontend/dist/." "$frontend_new/"

if [[ -f "$api_path" ]]; then
  cp -a "$api_path" "$api_previous"
fi
install -m 0755 "$build_dir/bin/api" "${api_path}.new"
mv -f "${api_path}.new" "$api_path"

if [[ -d "$FRONTEND_DIR" ]]; then
  mv "$FRONTEND_DIR" "$frontend_previous"
fi
mv "$frontend_new" "$FRONTEND_DIR"

echo "Restarting $SERVICE_NAME ..."
systemctl restart "$SERVICE_NAME"

healthy=false
for _ in $(seq 1 30); do
  if curl --fail --silent "$HEALTH_URL" >/dev/null; then
    healthy=true
    break
  fi
  sleep 1
done

if [[ "$healthy" != true ]]; then
  echo "Health check failed; restoring the previous release." >&2
  if [[ -f "$api_previous" ]]; then
    mv -f "$api_previous" "$api_path"
  fi
  rm -rf "$FRONTEND_DIR"
  if [[ -d "$frontend_previous" ]]; then
    mv "$frontend_previous" "$FRONTEND_DIR"
  fi
  systemctl restart "$SERVICE_NAME"
  exit 1
fi

rm -f "$api_previous"
rm -rf "$frontend_previous"
echo "Deployment completed: $(git rev-parse --short HEAD) from $DEPLOY_BRANCH"
echo "Health check passed: $HEALTH_URL"
