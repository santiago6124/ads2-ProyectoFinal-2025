#!/bin/bash
# Script para inicializar el índice de SolR para Orders

SOLR_URL="${SOLR_URL:-http://localhost:8983/solr}"
COLLECTION="${COLLECTION:-orders_search}"

echo "Inicializando colección SolR para Orders: $COLLECTION"
echo "Se utilizará la configuración dinámica por defecto. No se requieren pasos adicionales."

