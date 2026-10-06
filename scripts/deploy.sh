#!/usr/bin/env bash
set -Eeuo pipefail
umask 077

# Run on the Ubuntu server as root:
#   sudo bash /opt/tkdhub/scripts/deploy.sh
#
# The Nginx document root should point to /opt/tkdhub/frontend/dist.
# Optional first argument selects another branch; master is the production default.

APP_DIR="${APP_DIR:-/opt/tkdhub}"
DEPLOY_BRANCH="${1:-${DEPLOY_BRANCH:-master}}"
SERVICE_NAME="${SERVICE_NAME:-tkdhub}"
FRONTEND_DIR="${FRONTEND_DIR:-$APP_DIR/frontend/dist}"
ENV_FILE="${ENV_FILE:-/etc/tkdhub.env}"
BACKUP_DIR="${BACKUP_DIR:-/var/backups/tkdhub}"
HEALTH_URL="${HEALTH_URL:-http://127.0.0.1:8080/health}"
GO_IMAGE="${GO_IMAGE:-golang:1.27.1}"
NODE_IMAGE="${NODE_IMAGE:-node:22-bookworm}"

if [[ ${EUID} -ne 0 ]]; then
  echo "Run this script as root (use sudo)." >&2
  exit 1
fi

for command in git docker curl systemctl tar pg_dump; do
  command -v "$command" >/dev/null 2>&1 || {
    echo "Required command is missing: $command" >&2
    exit 1
  }
done

[[ -d "$APP_DIR/.git" ]] || {
  echo "Git checkout not found at $APP_DIR" >&2
  exit 1
}
[[ -f "$ENV_FILE" ]] || {
  echo "Database environment file not found: $ENV_FILE" >&2
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
frontend_parent="$(dirname -- "$FRONTEND_DIR")"
frontend_name="$(basename -- "$FRONTEND_DIR")"
mkdir -p "$frontend_parent" "$APP_DIR/bin" "$BACKUP_DIR"
chmod 0700 "$BACKUP_DIR"
frontend_new="$(mktemp -d "$frontend_parent/.${frontend_name}.new.XXXXXX")"
frontend_previous="$(mktemp -d "$frontend_parent/.${frontend_name}.previous.XXXXXX")"
rmdir "$frontend_previous"
api_path="$APP_DIR/bin/api"
api_previous="$APP_DIR/bin/api.previous.$$"
had_api=false
had_frontend=false
release_started=false
healthy=false
database_backup=""

rollback_release() {
  local status=$?
  trap - EXIT
  set +e
  rm -rf "$build_dir" "$frontend_new" "${api_path}.new"

  if [[ "$release_started" == true && "$healthy" != true ]]; then
    echo "Deployment failed; restoring the previous application files." >&2
    if [[ "$had_api" == true && -f "$api_previous" ]]; then
      mv -f "$api_previous" "$api_path"
    else
      rm -f "$api_path"
    fi
    rm -rf "$FRONTEND_DIR"
    if [[ "$had_frontend" == true && -d "$frontend_previous" ]]; then
      mv "$frontend_previous" "$FRONTEND_DIR"
    fi
    systemctl restart "$SERVICE_NAME" || true
    if [[ -n "$database_backup" ]]; then
      echo "Database migrations are not automatically reversed. Backup: $database_backup" >&2
    fi
  fi

  rm -rf "$frontend_previous"
  exit "$status"
}
trap rollback_release EXIT

echo "Building backend (API, migration runner, admin tool) ..."
docker run --rm \
  -v "$APP_DIR/backend:/src:ro" \
  -v "$build_dir:/out" \
  -w /src \
  -e CGO_ENABLED=0 \
  "$GO_IMAGE" \
  sh -c 'go build -trimpath -o /out/api ./cmd/api && go build -trimpath -o /out/migrate ./cmd/migrate && go build -trimpath -o /out/create-admin ./cmd/create-admin'

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
[[ -s "$build_dir/frontend/dist/index.html" ]] || {
  echo "Frontend build did not produce dist/index.html." >&2
  exit 1
}
cp -a "$build_dir/frontend/dist/." "$frontend_new/"

echo "Backing up the database ..."
database_backup="$BACKUP_DIR/tkdhub-$(date -u +%Y%m%dT%H%M%SZ)-$$.dump"
(
  set -a
  # This trusted, root-owned server file contains the live DATABASE_URL.
  source "$ENV_FILE"
  set +a
  : "${DATABASE_URL:?DATABASE_URL is missing from $ENV_FILE}"
  pg_dump --format=custom --file="$database_backup" --dbname="$DATABASE_URL"
)
chmod 0600 "$database_backup"
echo "Database backup saved: $database_backup"

echo "Applying all pending database migrations ..."
(
  set -a
  source "$ENV_FILE"
  set +a
  "$build_dir/migrate"
)

echo "Installing release ..."
install -m 0755 "$build_dir/create-admin" "$APP_DIR/bin/create-admin"
if [[ -f "$api_path" ]]; then
  cp -a "$api_path" "$api_previous"
  had_api=true
fi
install -m 0755 "$build_dir/api" "${api_path}.new"

if [[ -d "$FRONTEND_DIR" ]]; then
  mv "$FRONTEND_DIR" "$frontend_previous"
  had_frontend=true
fi
release_started=true
mv -f "${api_path}.new" "$api_path"
mv "$frontend_new" "$FRONTEND_DIR"

echo "Restarting $SERVICE_NAME ..."
systemctl restart "$SERVICE_NAME"

for _ in $(seq 1 30); do
  if curl --fail --silent "$HEALTH_URL" >/dev/null; then
    healthy=true
    break
  fi
  sleep 1
done

if [[ "$healthy" != true ]]; then
  echo "Health check failed: $HEALTH_URL" >&2
  exit 1
fi

rm -rf "$api_previous" "$frontend_previous" "$build_dir"
trap - EXIT
echo "Deployment completed: $(git rev-parse --short HEAD) from $DEPLOY_BRANCH"
echo "Frontend installed at: $FRONTEND_DIR"
echo "Health check passed: $HEALTH_URL"
