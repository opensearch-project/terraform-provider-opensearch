#!/bin/bash
# Pre-populates OpenSearch with test data required by integration tests.
# Must be run after OpenSearch is healthy.
# Expects environment variables from run-tests.sh or manual setup:
#   OPENSEARCH_URL      (e.g. https://localhost:9200)
#   OPENSEARCH_USERNAME (default: admin)
#   OPENSEARCH_PASSWORD

set -e

OS_URL="${1}"
OS_USER="${2}"
OS_PASS="${3}"

if [ $# -ne 3 ]; then
    echo "Missing required parameters: <os url> <os username> <os password>"
    exit 1
fi


CURL="curl -s -k -u ${OS_USER}:${OS_PASS}"

echo "Setting up integration test data on ${OS_URL}..."

# ---------------------------------------------------------------------------
# 1. Security Audit Logs index (for anomaly detection in Scenario 2)
# ---------------------------------------------------------------------------
AUDIT_INDEX="security-audit-logs"

echo "  -> Creating index ${AUDIT_INDEX}..."
${CURL} -X PUT "${OS_URL}/${AUDIT_INDEX}" -H 'Content-Type: application/json' -d '{
  "settings": {
    "number_of_shards": 1,
    "number_of_replicas": 0
  },
  "mappings": {
    "properties": {
      "@timestamp": { "type": "date" },
      "user":       { "type": "keyword" },
      "action":     { "type": "keyword" },
      "source_ip":  { "type": "ip" },
      "result":     { "type": "keyword" }
    }
  }
}' | grep -o '"acknowledged":true' >/dev/null && echo "     OK" || echo "     WARN (may already exist)"

echo "  -> Indexing sample audit documents..."
for i in $(seq 1 50); do
  # Mix of success and failure events; some from unusual IPs
  if [ $((i % 7)) -eq 0 ]; then
    ACTION="failed_login"
    RESULT="failure"
    IP="192.168.99.$i"
  else
    ACTION="search"
    RESULT="success"
    IP="10.0.0.$i"
  fi

  ${CURL} -X POST "${OS_URL}/${AUDIT_INDEX}/_doc" -H 'Content-Type: application/json' -d "{
    \"@timestamp\": \"2024-01-15T10:$(printf '%02d' $((i % 60))):00Z\",
    \"user\": \"user-$i\",
    \"action\": \"${ACTION}\",
    \"source_ip\": \"${IP}\",
    \"result\": \"${RESULT}\"
  }" >/dev/null
done
echo "     OK (50 docs)"

# Force refresh so anomaly detection can see the data immediately
${CURL} -X POST "${OS_URL}/${AUDIT_INDEX}/_refresh" >/dev/null

# ---------------------------------------------------------------------------
# 2. Application logs bootstrap index (for ISM rollover in Scenario 1)
# ---------------------------------------------------------------------------
LOGS_INDEX="logs-application-000001"

echo "  -> Creating bootstrap index ${LOGS_INDEX}..."
${CURL} -X PUT "${OS_URL}/${LOGS_INDEX}" -H 'Content-Type: application/json' -d '{
  "settings": {
    "number_of_shards": 1,
    "number_of_replicas": 0
  },
  "mappings": {
    "properties": {
      "@timestamp": { "type": "date" },
      "level":      { "type": "keyword" },
      "message":    { "type": "text" },
      "service":    { "type": "keyword" }
    }
  },
  "aliases": {
    "logs-application": {
      "is_write_index": true
    }
  }
}' | grep -o '"acknowledged":true' >/dev/null && echo "     OK" || echo "     WARN (may already exist)"

echo "  -> Indexing sample application log documents..."
for i in $(seq 1 30); do
  LEVEL=$(if [ $((i % 5)) -eq 0 ]; then echo "ERROR"; elif [ $((i % 3)) -eq 0 ]; then echo "WARN"; else echo "INFO"; fi)
  ${CURL} -X POST "${OS_URL}/${LOGS_INDEX}/_doc" -H 'Content-Type: application/json' -d "{
    \"@timestamp\": \"2024-01-15T11:$(printf '%02d' $((i % 60))):00Z\",
    \"level\": \"${LEVEL}\",
    \"message\": \"Log entry $i from application\",
    \"service\": \"app-$((i % 3))\"
  }" >/dev/null
done
echo "     OK (30 docs)"

${CURL} -X POST "${OS_URL}/${LOGS_INDEX}/_refresh" >/dev/null

# ---------------------------------------------------------------------------
# 3. Multi-tenant metrics data stream backing index (Scenario 3)
# ---------------------------------------------------------------------------
METRICS_INDEX="metrics-customer-a-000001"

echo "  -> Creating bootstrap index ${METRICS_INDEX}..."
${CURL} -X PUT "${OS_URL}/${METRICS_INDEX}" -H 'Content-Type: application/json' -d '{
  "settings": {
    "number_of_shards": 1,
    "number_of_replicas": 0
  },
  "mappings": {
    "properties": {
      "@timestamp": { "type": "date" },
      "cpu_percent":   { "type": "float" },
      "memory_used":   { "type": "long" },
      "tenant":        { "type": "keyword" }
    }
  },
  "aliases": {
    "metrics-customer-a": {
      "is_write_index": true
    }
  }
}' | grep -o '"acknowledged":true' >/dev/null && echo "     OK" || echo "     WARN (may already exist)"

echo "  -> Indexing sample metrics documents..."
for i in $(seq 1 20); do
  ${CURL} -X POST "${OS_URL}/${METRICS_INDEX}/_doc" -H 'Content-Type: application/json' -d "{
    \"@timestamp\": \"2024-01-15T12:$(printf '%02d' $((i % 60))):00Z\",
    \"cpu_percent\": $((10 + (i * 3) % 80)),
    \"memory_used\": $((100000000 + i * 1000000)),
    \"tenant\": \"customer-a\"
  }" >/dev/null
done
echo "     OK (20 docs)"

${CURL} -X POST "${OS_URL}/${METRICS_INDEX}/_refresh" >/dev/null

echo "Integration test data setup complete."
