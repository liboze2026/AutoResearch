#!/bin/sh
set -eu

HOST_SSH_DIR="${HOST_SSH_DIR:-/host-ssh}"
TARGET_SSH_DIR="${HOME:-/root}/.ssh"

rewrite_ssh_config() {
  config_file="$1"
  tmp_file="${config_file}.tmp"

  awk -v target_dir="$TARGET_SSH_DIR" '
    {
      original = $0
      sub(/\r$/, "", original)
      trimmed = $0
      sub(/^[[:space:]]+/, "", trimmed)
      sub(/\r$/, "", trimmed)

      if (trimmed ~ /^IdentityFile[[:space:]]+[A-Za-z]:\\Users\\[^\\]+\\.ssh\\[^[:space:]]+$/) {
        match(original, /^[[:space:]]*/)
        indent = substr(original, RSTART, RLENGTH)
        file = trimmed
        sub(/^IdentityFile[[:space:]]+[A-Za-z]:\\Users\\[^\\]+\\.ssh\\/, "", file)
        printf "%sIdentityFile %s/%s\n", indent, target_dir, file
        next
      }

      print original
    }
  ' "$config_file" > "$tmp_file"

  mv "$tmp_file" "$config_file"
}

if [ -d "$HOST_SSH_DIR" ]; then
  mkdir -p "$TARGET_SSH_DIR"
  cp -R "$HOST_SSH_DIR"/. "$TARGET_SSH_DIR"/ 2>/dev/null || true

  if [ -f "$TARGET_SSH_DIR/config" ]; then
    rewrite_ssh_config "$TARGET_SSH_DIR/config"
  fi

  chmod 700 "$TARGET_SSH_DIR" 2>/dev/null || true
  find "$TARGET_SSH_DIR" -type d -exec chmod 700 {} \; 2>/dev/null || true
  find "$TARGET_SSH_DIR" -type f ! -name "*.pub" ! -name "known_hosts" -exec chmod 600 {} \; 2>/dev/null || true

  if [ -f "$TARGET_SSH_DIR/known_hosts" ]; then
    chmod 644 "$TARGET_SSH_DIR/known_hosts" 2>/dev/null || true
  fi

  find "$TARGET_SSH_DIR" -type f -name "*.pub" -exec chmod 644 {} \; 2>/dev/null || true
fi

exec /app/mrag-server
