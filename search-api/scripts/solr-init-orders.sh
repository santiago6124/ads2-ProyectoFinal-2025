#!/bin/bash
# Script para inicializar el índice de SolR para Orders

SOLR_URL="${SOLR_URL:-http://localhost:8983/solr}"
COLLECTION="${COLLECTION:-orders_search}"

echo "Inicializando colección SolR para Orders: $COLLECTION"

# Add field types and fields to schema
echo "Adding field definitions to schema..."

# Add date field type if not exists
curl -X POST -H 'Content-type:application/json' --data-binary '{
  "add-field-type": {
    "name": "pdate",
    "class": "solr.DatePointField",
    "docValues": true
  }
}' "$SOLR_URL/$COLLECTION/schema" 2>/dev/null || true

# Add fields
curl -X POST -H 'Content-type:application/json' --data-binary '{
  "add-field": [
    {"name":"user_id", "type":"plong", "stored":true, "indexed":true},
    {"name":"type", "type":"string", "stored":true, "indexed":true},
    {"name":"status", "type":"string", "stored":true, "indexed":true},
    {"name":"order_kind", "type":"string", "stored":true, "indexed":true},
    {"name":"crypto_symbol", "type":"string", "stored":true, "indexed":true},
    {"name":"crypto_name", "type":"text_general", "stored":true, "indexed":true},
    {"name":"quantity", "type":"pdouble", "stored":true, "indexed":true},
    {"name":"price", "type":"pdouble", "stored":true, "indexed":true},
    {"name":"total_amount", "type":"pdouble", "stored":true, "indexed":true},
    {"name":"fee", "type":"pdouble", "stored":true, "indexed":true},
    {"name":"created_at", "type":"pdate", "stored":true, "indexed":true},
    {"name":"executed_at", "type":"pdate", "stored":true, "indexed":true},
    {"name":"updated_at", "type":"pdate", "stored":true, "indexed":true},
    {"name":"cancelled_at", "type":"pdate", "stored":true, "indexed":true},
    {"name":"error_message", "type":"text_general", "stored":true, "indexed":true},
    {"name":"search_text", "type":"text_general", "stored":true, "indexed":true}
  ]
}' "$SOLR_URL/$COLLECTION/schema"

echo "Schema initialization completed for $COLLECTION"