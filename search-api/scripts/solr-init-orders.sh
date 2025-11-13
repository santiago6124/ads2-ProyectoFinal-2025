#!/bin/bash
# Script para inicializar el Ã­ndice de SolR para Orders

SOLR_URL="${SOLR_URL:-http://localhost:8983/solr}"
COLLECTION="${COLLECTION:-orders_search}"

echo "Inicializando colecciÃ³n SolR para Orders: $COLLECTION"
echo "Se utilizarÃ¡ la configuraciÃ³n dinÃ¡mica por defecto. No se requieren pasos adicionales."

