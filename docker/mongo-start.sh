#!/bin/sh
set -eu
# La clave interna persiste con el volumen. Convertir standalone a rs0 conserva
# datos y credenciales; no reinicia ni borra colecciones.
key=/data/db/.replica-key
if [ ! -s "$key" ]; then
    umask 077
    openssl rand -base64 756 > "$key"
fi
chown mongodb:mongodb "$key"
chmod 400 "$key"
exec /usr/local/bin/docker-entrypoint.sh mongod --replSet rs0 --bind_ip_all --keyFile "$key"
