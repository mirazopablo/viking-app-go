#!/bin/bash

# Ensure script stops on first error
set -e

# Asegurar que los binarios de Go estén disponibles en el PATH del script
export PATH=$PATH:$(go env GOPATH)/bin

echo "🚀 Iniciando proceso de release..."

# 0. Validar estado del repositorio y rama base
CURRENT_BRANCH=$(git rev-parse --abbrev-ref HEAD)
if [ "$CURRENT_BRANCH" != "main" ]; then
    echo "⛔ Debes ejecutar ./release.sh desde la rama 'main' (con tus PRs ya mergeados)."
    echo "Rama actual: $CURRENT_BRANCH"
    exit 1
fi

if [ -n "$(git status --porcelain)" ]; then
    echo "⛔ Hay cambios sin commitear en el directorio de trabajo. Limpia o guarda tus cambios antes de continuar."
    exit 1
fi

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

# 3. Crear rama de release (para cumplir con el hook global que prohíbe push directo a main)
RELEASE_BRANCH="chore/release-$NEXT_VERSION"
echo "🌿 Creando rama de release: $RELEASE_BRANCH..."
git checkout -b "$RELEASE_BRANCH"

# 4. Generar Changelog
echo "📝 Generando CHANGELOG.md..."
git-chglog --next-tag "$NEXT_VERSION" -o CHANGELOG.md

# 5. Git Commit en la rama de release
echo "💾 Creando commit de release en $RELEASE_BRANCH..."
git add CHANGELOG.md
git commit -m "chore(release): $NEXT_VERSION"

# 6. Push de la rama de release a GitHub
echo "☁️ Subiendo rama $RELEASE_BRANCH a GitHub..."
git push -u origin "$RELEASE_BRANCH"

echo ""
echo "✅ Rama $RELEASE_BRANCH subida exitosamente."
echo "👉 Siguientes pasos:"
echo "   1. Mergeá el PR '$RELEASE_BRANCH -> main' en GitHub (Create a merge commit)."
echo "   2. Creá y subí el tag apuntando a main:"
echo "      git fetch origin main && git tag $NEXT_VERSION origin/main && git push origin $NEXT_VERSION"
echo "   3. Volvé a main y limpiá la rama temporal:"
echo "      git checkout main && git pull origin main && git branch -d $RELEASE_BRANCH"
