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


# Global Variables
MINIKUBE_IP := localhost

help:
	@echo "═══════════════════════════════════════════════"
	@echo "🚀 Available Commands"
	@echo "═══════════════════════════════════════════════"
	@echo ""
	@echo "💻 Minikube:"
	@echo "  minikube_start                  ▶️  Start Minikube"
	@echo "  minikube_stop                   ▶️  Stop Minikube"
	@echo "  minikube_destroy                ▶️  Delete Minikube"
	@echo ""
	@echo "📘 Run 'make <command>' to execute a specific task."
	@echo "═══════════════════════════════════════════════"

# ─────────────────────────────────────────────────────────────
# 🐳 Minikube – Start, Stop, Destroy
minikube_start:
	@echo "🚀 Starting Minikube..."
	@minikube start --driver=docker --memory=4000 --cpus=3 --force

minikube_stop:
	@echo "🛑 Stopping Minikube..."
	@minikube stop

minikube_destroy:
	@echo "🗑️  Destroying Minikube..."
	@minikube delete
	@echo -e "$(GREEN)✅ Minikube has been destroyed.$(NC)"