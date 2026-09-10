#!/bin/bash

# Ensure script stops on first error
set -e

# Asegurar que los binarios de Go estén disponibles en el PATH del script
export PATH=$PATH:$(go env GOPATH)/bin

echo "🚀 Iniciando proceso de release..."

# 1. Asegurar dependencias (svu y git-chglog)
echo "📦 Verificando dependencias..."
command -v svu >/dev/null 2>&1 || go install github.com/caarlos0/svu@latest
command -v git-chglog >/dev/null 2>&1 || go install github.com/git-chglog/git-chglog/cmd/git-chglog@latest

# 2. Calcular siguiente versión
echo "🔍 Calculando próxima versión semántica..."
NEXT_VERSION=$(svu next --force-patch-increment)
CURRENT_VERSION=$(svu current)

if [ "$CURRENT_VERSION" = "$NEXT_VERSION" ]; then
    echo "⚠️ No hay cambios suficientes (feat/fix/BREAKING) para una nueva versión."
    echo "Actual: $CURRENT_VERSION | Próxima: $NEXT_VERSION"
    exit 1
fi

echo "✨ Siguiente versión será: $NEXT_VERSION"

# 3. Generar Changelog
echo "📝 Generando CHANGELOG.md..."
git-chglog --next-tag "$NEXT_VERSION" -o CHANGELOG.md

# 4. Git Commit y Tag
echo "💾 Creando commit y tag de release..."
git add CHANGELOG.md
git commit -m "chore(release): $NEXT_VERSION"
git tag "$NEXT_VERSION"

# 5. Push al repositorio remoto
echo "☁️ Subiendo cambios y tags a GitHub..."
git push
git push --tags

echo "✅ ¡Release $NEXT_VERSION publicado exitosamente en producción!"
