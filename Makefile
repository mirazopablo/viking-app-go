.PHONY: release setup

setup:
	@echo "Verificando dependencias..."
	@command -v svu >/dev/null 2>&1 || go install github.com/caarlos0/svu@latest
	@command -v git-chglog >/dev/null 2>&1 || go install github.com/git-chglog/git-chglog/cmd/git-chglog@latest

release: setup
	@echo "Calculando próxima versión semántica..."
	$(eval NEXT_VERSION := $(shell svu next))
	@if [ "$(shell svu current)" = "$(NEXT_VERSION)" ]; then \
		echo "No hay cambios suficientes (feat/fix/BREAKING) para una nueva versión."; \
		exit 1; \
	fi
	@echo "Siguiente versión será: $(NEXT_VERSION)"
	@git-chglog --next-tag $(NEXT_VERSION) -o CHANGELOG.md
	@git add CHANGELOG.md
	@git commit -m "chore(release): $(NEXT_VERSION)"
	@git tag $(NEXT_VERSION)
	@echo "✅ Release $(NEXT_VERSION) creado localmente."
	@echo "🚀 Recuerda ejecutar 'git push && git push --tags' para enviarlo a GitHub."
