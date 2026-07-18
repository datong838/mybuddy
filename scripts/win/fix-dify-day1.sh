#!/bin/bash
set -e
docker info >/dev/null 2>&1 || { nohup dockerd >/var/log/dockerd.log 2>&1 & sleep 5; }
docker exec docker-db_postgres-1 psql -U postgres -tc "SELECT 1 FROM pg_database WHERE datname='dify_plugin'" | grep -q 1 \
  || docker exec docker-db_postgres-1 psql -U postgres -c "CREATE DATABASE dify_plugin;"
docker restart docker-plugin_daemon-1 docker-api-1 docker-worker-1 docker-nginx-1 || true
sleep 10
cd /mnt/c/work/projects/wchat/mybuddy-v01/dify/docker
docker compose -f docker-compose.yaml -f docker-compose.wsl-volumes.yaml ps
echo "=== curl from wsl ==="
curl -sI --max-time 10 http://127.0.0.1/ | head -20 || true
echo "=== nginx logs ==="
docker logs docker-nginx-1 2>&1 | tail -25
echo "=== api health ==="
docker inspect --format='{{.State.Health.Status}}' docker-api-1 2>/dev/null || true
