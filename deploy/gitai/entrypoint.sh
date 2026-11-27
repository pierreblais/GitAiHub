#!/usr/bin/env bash
# Renders gitai's config.toml from environment variables (12-factor) and starts the server.
#   GITAI_SHARED_TOKEN   operator token (required)
#   GITAI_AGENTS         comma-separated agent names; tokens are read from
#                        GITAI_TOKEN_<NAME> or generated into $DATA/tokens/<name>
#   GITAI_LISTEN_ADDR    TCP listen address (default 0.0.0.0:8075)
#   GITAI_PROTECT_MAIN   agent allowed to land on main (default: all agents allowed)
#   GITAI_EXTRA_CONFIG   raw TOML appended to the config (e.g. [[graft.review.rule]] blocks)
#   GITAI_EXTRA_CONFIG_FILE  path of a TOML file appended the same way
#   GITAI_DATA_DIR, GITAI_BIN_DIR  override /var/lib/gitai and /opt/gitai/bin
set -euo pipefail
DATA="${GITAI_DATA_DIR:-/var/lib/gitai}"
BIN="${GITAI_BIN_DIR:-/opt/gitai/bin}"
# The runtime directory only holds sockets and per-process state; a container
# that was killed leaves it behind and gitaly then refuses to start.
rm -rf "$DATA/run"
mkdir -p "$DATA"/{repos,run,tokens}
: "${GITAI_SHARED_TOKEN:?GITAI_SHARED_TOKEN is required}"
LISTEN="${GITAI_LISTEN_ADDR:-0.0.0.0:8075}"
AGENTS="${GITAI_AGENTS:-}"
CFG="$DATA/config.toml"
{
  echo "bin_dir = \"$BIN\""
  echo "socket_path = \"$DATA/run/gitaly.sock\""
  echo "listen_addr = \"$LISTEN\""
  echo "runtime_dir = \"$DATA/run\""
  echo "prometheus_listen_addr = \"0.0.0.0:9236\""
  echo "[auth]"
  echo "token = \"$GITAI_SHARED_TOKEN\""
  echo "[[storage]]"
  echo "name = \"default\""
  echo "path = \"$DATA/repos\""
  echo "[graft]"
  echo "standalone = true"
  IFS=',' read -ra names <<< "$AGENTS"
  for a in "${names[@]}"; do
    [ -z "$a" ] && continue
    var="GITAI_TOKEN_$(echo "$a" | tr 'a-z-' 'A-Z_')"
    tf="$DATA/tokens/$a"
    if [ -n "${!var:-}" ]; then printf '%s' "${!var}" > "$tf"; elif [ ! -s "$tf" ]; then openssl rand -hex 32 > "$tf"; fi
    chmod 0600 "$tf"
    echo "[[graft.agent]]"
    echo "name = \"$a\""
    echo "token_file = \"$tf\""
  done
  if [ -n "${GITAI_PROTECT_MAIN:-}" ]; then
    echo "[[graft.policy]]"; echo "actor = \"$GITAI_PROTECT_MAIN\""; echo 'refs = ["refs/heads/main"]'; echo "allow = true"
    echo "[[graft.policy]]"; echo 'refs = ["refs/heads/main"]'; echo "allow = false"
  fi
  echo "[[graft.policy]]"; echo 'refs = ["refs/**"]'; echo "allow = true"
  echo "[graft.changelog]"; echo "disabled = false"
  echo "[graft.derived]"; echo "disabled = false"; echo "eager = false"
  [ -n "${GITAI_EXTRA_CONFIG:-}" ] && printf '%s\n' "$GITAI_EXTRA_CONFIG"
  [ -n "${GITAI_EXTRA_CONFIG_FILE:-}" ] && cat "$GITAI_EXTRA_CONFIG_FILE"
  true
} > "$CFG"
exec "$BIN/gitaly" serve "$CFG"
