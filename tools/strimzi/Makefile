SHELL := /bin/bash
.DEFAULT_GOAL := help
# ANSI Colors
RED := \033[0;31m
GREEN := \033[0;32m
YELLOW := \033[1;33m
NC := \033[0m

# ANSI Colors
RED := \033[0;31m
GREEN := \033[0;32m
YELLOW := \033[1;33m
NC := \033[0m

help:
	@echo "═══════════════════════════════════════════════"
	@echo "🚀 Available Commands for Kafka Showcase Setup"
	@echo "═══════════════════════════════════════════════"
	@echo ""
	@echo "🔧 Required Setup:"
	@echo "  strimzi_operator_install        ▶️  Install Strimzi Operator"
	@echo ""
	@echo "🌐 UI Tools:"
	@echo "  kafka_ui_port_forward_ui      ▶️  Port-forward Kafka UI"
	@echo ""
	@echo "📘 Run 'make <command>' to execute a specific task."
	@echo "═══════════════════════════════════════════════"


# ─────────────────────────────────────────────────────────────
# 🧰 Installation & Deinstallation – Strimzi
# ─────────────────────────────────────────────────────────────

strimzi_operator_install:
	# kubectl create -f 'https://strimzi.io/install/latest?namespace=kafka' -n kafka
	@echo "🚀 Installing Strimzi Operator via ArgoCD Application..."
	@kubectl create namespace kafka
	@kubectl create -f 'https://strimzi.io/install/latest?namespace=kafka' -n kafka
	#@kubectl apply -f ./app/
	@echo -e "$(GREEN)✅ Strimzi Operator Application has been applied via ArgoCD.$(NC)"

# ─────────────────────────────────────────────────────────────
# 🌐 UI Zugriff – Kafka UI
# ─────────────────────────────────────────────────────────────

kafka_ui_port_forward_ui:
	@kubectl -n kafka-ui port-forward services/kafka-ui-service 8089:8080 > /dev/null 2>&1 &
	@sleep 3
	@echo "🌐 Kafka UI is available at http://localhost:8089/"
