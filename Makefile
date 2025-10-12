# ============================================================================
# CryptoSim - Makefile
# Comandos útiles para gestionar el proyecto
# ============================================================================

.PHONY: help build up down restart logs clean test ps status

# Colores para output
BLUE := \033[0;34m
GREEN := \033[0;32m
YELLOW := \033[1;33m
RED := \033[0;31m
NC := \033[0m # No Color

# ----------------------------------------------------------------------------
# HELP - Muestra todos los comandos disponibles
# ----------------------------------------------------------------------------
help:
	@echo "$(BLUE)═══════════════════════════════════════════════════════════════$(NC)"
	@echo "$(GREEN)  CryptoSim - Docker Compose Management$(NC)"
	@echo "$(BLUE)═══════════════════════════════════════════════════════════════$(NC)"
	@echo ""
	@echo "$(YELLOW)Comandos principales:$(NC)"
	@echo "  $(GREEN)make up$(NC)              - Levantar todos los servicios"
	@echo "  $(GREEN)make down$(NC)            - Detener y eliminar todos los contenedores"
	@echo "  $(GREEN)make restart$(NC)         - Reiniciar todos los servicios"
	@echo "  $(GREEN)make build$(NC)           - Construir todas las imágenes"
	@echo "  $(GREEN)make rebuild$(NC)         - Reconstruir y levantar (sin cache)"
	@echo ""
	@echo "$(YELLOW)Comandos de monitoreo:$(NC)"
	@echo "  $(GREEN)make logs$(NC)            - Ver logs de todos los servicios"
	@echo "  $(GREEN)make logs-users$(NC)      - Ver logs del Users API"
	@echo "  $(GREEN)make logs-orders$(NC)     - Ver logs del Orders API"
	@echo "  $(GREEN)make logs-search$(NC)     - Ver logs del Search API"
	@echo "  $(GREEN)make logs-market$(NC)     - Ver logs del Market Data API"
	@echo "  $(GREEN)make logs-portfolio$(NC)  - Ver logs del Portfolio API"
	@echo "  $(GREEN)make logs-wallet$(NC)     - Ver logs del Wallet API"
	@echo "  $(GREEN)make ps$(NC)              - Ver estado de los contenedores"
	@echo "  $(GREEN)make status$(NC)          - Ver estado detallado"
	@echo ""
	@echo "$(YELLOW)Comandos de limpieza:$(NC)"
	@echo "  $(GREEN)make clean$(NC)           - Limpiar contenedores y volúmenes"
	@echo "  $(GREEN)make clean-all$(NC)       - Limpieza completa (incluye imágenes)"
	@echo "  $(GREEN)make prune$(NC)           - Limpiar recursos no utilizados de Docker"
	@echo ""
	@echo "$(YELLOW)Comandos por servicio:$(NC)"
	@echo "  $(GREEN)make up-users$(NC)        - Levantar solo Users API + deps"
	@echo "  $(GREEN)make up-orders$(NC)       - Levantar solo Orders API + deps"
	@echo "  $(GREEN)make up-infra$(NC)        - Levantar solo infraestructura"
	@echo ""
	@echo "$(YELLOW)Monitoring:$(NC)"
	@echo "  $(GREEN)make monitoring-up$(NC)   - Levantar Prometheus + Grafana"
	@echo "  $(GREEN)make monitoring-down$(NC) - Detener Prometheus + Grafana"
	@echo ""
	@echo "$(YELLOW)Utilidades:$(NC)"
	@echo "  $(GREEN)make test$(NC)            - Ejecutar tests de todos los servicios"
	@echo "  $(GREEN)make health$(NC)          - Verificar salud de todos los servicios"
	@echo "  $(GREEN)make env$(NC)             - Copiar .env.example a .env"
	@echo ""

# ----------------------------------------------------------------------------
# MAIN COMMANDS
# ----------------------------------------------------------------------------

# Levantar todos los servicios
up:
	@echo "$(BLUE)▶ Levantando todos los servicios...$(NC)"
	docker-compose up -d
	@echo "$(GREEN)✓ Servicios levantados$(NC)"
	@make status

# Detener todos los servicios
down:
	@echo "$(BLUE)▶ Deteniendo todos los servicios...$(NC)"
	docker-compose down
	@echo "$(GREEN)✓ Servicios detenidos$(NC)"

# Reiniciar todos los servicios
restart:
	@echo "$(BLUE)▶ Reiniciando servicios...$(NC)"
	docker-compose restart
	@echo "$(GREEN)✓ Servicios reiniciados$(NC)"

# Construir todas las imágenes
build:
	@echo "$(BLUE)▶ Construyendo imágenes...$(NC)"
	docker-compose build
	@echo "$(GREEN)✓ Imágenes construidas$(NC)"

# Reconstruir sin cache
rebuild:
	@echo "$(BLUE)▶ Reconstruyendo sin cache...$(NC)"
	docker-compose build --no-cache
	docker-compose up -d
	@echo "$(GREEN)✓ Servicios reconstruidos y levantados$(NC)"

# ----------------------------------------------------------------------------
# LOGS
# ----------------------------------------------------------------------------

logs:
	docker-compose logs -f

logs-users:
	docker-compose logs -f users-api

logs-orders:
	docker-compose logs -f orders-api

logs-search:
	docker-compose logs -f search-api

logs-market:
	docker-compose logs -f market-data-api

logs-portfolio:
	docker-compose logs -f portfolio-api

logs-wallet:
	docker-compose logs -f wallet-api

logs-mysql:
	docker-compose logs -f users-mysql

logs-mongo:
	docker-compose logs -f orders-mongo portfolio-mongo wallet-mongo

logs-redis:
	docker-compose logs -f shared-redis

logs-rabbitmq:
	docker-compose logs -f shared-rabbitmq

# ----------------------------------------------------------------------------
# STATUS & MONITORING
# ----------------------------------------------------------------------------

ps:
	docker-compose ps

status:
	@echo "$(BLUE)═══════════════════════════════════════════════════════════════$(NC)"
	@echo "$(GREEN)  Estado de los servicios$(NC)"
	@echo "$(BLUE)═══════════════════════════════════════════════════════════════$(NC)"
	@docker-compose ps --format "table {{.Name}}\t{{.Status}}\t{{.Ports}}"

health:
	@echo "$(BLUE)▶ Verificando salud de los servicios...$(NC)"
	@echo ""
	@echo "$(YELLOW)Users API:$(NC)"
	@curl -s http://localhost:8001/health || echo "$(RED)✗ No disponible$(NC)"
	@echo ""
	@echo "$(YELLOW)Orders API:$(NC)"
	@curl -s http://localhost:8002/health || echo "$(RED)✗ No disponible$(NC)"
	@echo ""
	@echo "$(YELLOW)Search API:$(NC)"
	@curl -s http://localhost:8003/api/v1/health || echo "$(RED)✗ No disponible$(NC)"
	@echo ""
	@echo "$(YELLOW)Market Data API:$(NC)"
	@curl -s http://localhost:8004/health || echo "$(RED)✗ No disponible$(NC)"
	@echo ""
	@echo "$(YELLOW)Portfolio API:$(NC)"
	@curl -s http://localhost:8005/health || echo "$(RED)✗ No disponible$(NC)"
	@echo ""
	@echo "$(YELLOW)Wallet API:$(NC)"
	@curl -s http://localhost:8006/health || echo "$(RED)✗ No disponible$(NC)"
	@echo ""

# ----------------------------------------------------------------------------
# CLEANUP
# ----------------------------------------------------------------------------

clean:
	@echo "$(YELLOW)⚠ Deteniendo y eliminando contenedores y volúmenes...$(NC)"
	docker-compose down -v
	@echo "$(GREEN)✓ Limpieza completada$(NC)"

clean-all:
	@echo "$(RED)⚠ CUIDADO: Esto eliminará contenedores, volúmenes e imágenes$(NC)"
	@read -p "¿Estás seguro? [y/N]: " confirm && [ "$$confirm" = "y" ]
	docker-compose down -v --rmi all
	@echo "$(GREEN)✓ Limpieza completa realizada$(NC)"

prune:
	@echo "$(BLUE)▶ Limpiando recursos no utilizados de Docker...$(NC)"
	docker system prune -f
	docker volume prune -f
	@echo "$(GREEN)✓ Docker limpio$(NC)"

# ----------------------------------------------------------------------------
# SERVICIOS INDIVIDUALES
# ----------------------------------------------------------------------------

up-users:
	docker-compose up -d users-api users-mysql shared-redis

up-orders:
	docker-compose up -d orders-api orders-mongo shared-rabbitmq

up-search:
	docker-compose up -d search-api solr memcached

up-market:
	docker-compose up -d market-data-api shared-redis

up-portfolio:
	docker-compose up -d portfolio-api portfolio-mongo shared-redis shared-rabbitmq

up-wallet:
	docker-compose up -d wallet-api wallet-mongo shared-redis shared-rabbitmq

up-infra:
	docker-compose up -d shared-redis shared-rabbitmq solr memcached \
		users-mysql orders-mongo portfolio-mongo wallet-mongo

# ----------------------------------------------------------------------------
# MONITORING
# ----------------------------------------------------------------------------

monitoring-up:
	@echo "$(BLUE)▶ Levantando servicios de monitoring...$(NC)"
	docker-compose --profile monitoring up -d
	@echo "$(GREEN)✓ Prometheus: http://localhost:9090$(NC)"
	@echo "$(GREEN)✓ Grafana: http://localhost:3000$(NC)"

monitoring-down:
	docker-compose --profile monitoring down

# ----------------------------------------------------------------------------
# TESTING
# ----------------------------------------------------------------------------

test:
	@echo "$(BLUE)▶ Ejecutando tests...$(NC)"
	@echo "$(YELLOW)Users API:$(NC)"
	cd users-api && go test ./... -v || true
	@echo ""
	@echo "$(YELLOW)Orders API:$(NC)"
	cd orders-api && go test ./... -v || true
	@echo ""
	@echo "$(YELLOW)Portfolio API:$(NC)"
	cd portfolio-api && go test ./... -v || true
	@echo ""
	@echo "$(YELLOW)Wallet API:$(NC)"
	cd wallet-api && go test ./... -v || true
	@echo ""
	@echo "$(YELLOW)Market Data API:$(NC)"
	cd market-data-api && go test ./... -v || true
	@echo ""
	@echo "$(YELLOW)Search API:$(NC)"
	cd search-api && go test ./... -v || true

# ----------------------------------------------------------------------------
# UTILITIES
# ----------------------------------------------------------------------------

env:
	@if [ -f .env ]; then \
		echo "$(YELLOW)⚠ El archivo .env ya existe$(NC)"; \
	else \
		cp .env.example .env; \
		echo "$(GREEN)✓ Archivo .env creado desde .env.example$(NC)"; \
		echo "$(YELLOW)⚠ Recuerda modificar los valores según tu entorno$(NC)"; \
	fi

# Entrar a un contenedor
shell-users:
	docker-compose exec users-api sh

shell-orders:
	docker-compose exec orders-api sh

shell-portfolio:
	docker-compose exec portfolio-api sh

shell-wallet:
	docker-compose exec wallet-api sh

shell-mysql:
	docker-compose exec users-mysql mysql -u root -p

shell-mongo:
	docker-compose exec orders-mongo mongosh

shell-redis:
	docker-compose exec shared-redis redis-cli

# ----------------------------------------------------------------------------
# DATABASE OPERATIONS
# ----------------------------------------------------------------------------

db-backup-mysql:
	@echo "$(BLUE)▶ Creando backup de MySQL...$(NC)"
	docker-compose exec users-mysql mysqldump -u root -p users_db > backup-mysql-$(shell date +%Y%m%d-%H%M%S).sql
	@echo "$(GREEN)✓ Backup creado$(NC)"

db-backup-mongo:
	@echo "$(BLUE)▶ Creando backup de MongoDB...$(NC)"
	docker-compose exec orders-mongo mongodump --out=/tmp/backup
	docker cp cryptosim-orders-mongo:/tmp/backup ./backup-mongo-$(shell date +%Y%m%d-%H%M%S)
	@echo "$(GREEN)✓ Backup creado$(NC)"

# ----------------------------------------------------------------------------
# DEVELOPMENT
# ----------------------------------------------------------------------------

dev-up:
	@echo "$(BLUE)▶ Levantando entorno de desarrollo completo...$(NC)"
	docker-compose --profile monitoring up -d
	@make status
	@echo ""
	@echo "$(GREEN)═══════════════════════════════════════════════════════════════$(NC)"
	@echo "$(GREEN)  Servicios disponibles:$(NC)"
	@echo "$(GREEN)═══════════════════════════════════════════════════════════════$(NC)"
	@echo "  Users API:          http://localhost:8001"
	@echo "  Orders API:         http://localhost:8002"
	@echo "  Search API:         http://localhost:8003"
	@echo "  Market Data API:    http://localhost:8004"
	@echo "  Portfolio API:      http://localhost:8005"
	@echo "  Wallet API:         http://localhost:8006"
	@echo "  RabbitMQ Management: http://localhost:15672 (guest/guest)"
	@echo "  Prometheus:         http://localhost:9090"
	@echo "  Grafana:            http://localhost:3000 (admin/admin)"
	@echo "$(GREEN)═══════════════════════════════════════════════════════════════$(NC)"

# Default target
.DEFAULT_GOAL := help
