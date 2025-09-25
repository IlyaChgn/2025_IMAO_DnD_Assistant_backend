#!/bin/bash
set -e

echo "Waiting for Elasticsearch to be ready..."
until curl -s -f -u "$ELASTIC_USERNAME:$ELASTIC_PASSWORD" "http://$ELASTIC_HOST:$ELASTIC_PORT/_cluster/health" >/dev/null; do
  echo "Elasticsearch is unavailable - sleeping"
  sleep 5
done

curl -X POST -u "$ELASTIC_USERNAME:$ELASTIC_PASSWORD" "http://$ELASTIC_HOST:$ELASTIC_PORT/_security/user/$KIBANA_USER" \
  -H "Content-Type: application/json" \
  -d "{
    \"password\": \"$KIBANA_PASSWORD\",
    \"roles\": [\"superuser\", \"kibana_system\"],
    \"full_name\": \"Kibana Service User\"
  }"

echo "Kibana user created successfully"
