#!/bin/bash

echo "=== Stopping all NOTAM consumers ==="

# توقف همه containerهای مربوطه
docker stop $(docker ps -q --filter "name=notam") 2>/dev/null || true
docker stop $(docker ps -q --filter "ancestor=notam-consumer") 2>/dev/null || true

# حذف همه
docker rm $(docker ps -aq --filter "name=notam") 2>/dev/null || true
docker rm $(docker ps -aq --filter "ancestor=notam-consumer") 2>/dev/null || true

# توقف فرآیندهای host
pkill -f "notam-consumer" 2>/dev/null || true

echo "=== Starting fresh ==="
docker-compose up -d

sleep 3

echo "=== Status ==="
docker-compose ps

echo "=== Logs (last 10 lines) ==="
docker-compose logs --tail=10
