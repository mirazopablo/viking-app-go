package controllers

import (
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/mirazopablo/viking-app-go/config"
)

// FileController handles static file serving and uploads.
type FileController struct{}

// NewFileController instantiates a new FileController.
func NewFileController() *FileController {
	return &FileController{}
}

// ServeFile godoc
// @Summary Cargar archivo
// @Description Carga un archivo o imagen desde el servidor (incluso en subcarpetas)
// @Tags File Controller
// @ID serveFile
// @Produce octet-stream
// @Param filepath path string true "Ruta relativa del archivo"
// @Success 200 {file} file "OK"
// @Router /auth/uploads/{filepath} [get]
func (fc *FileController) ServeFile(c *gin.Context) {
	relPath := c.Param("filepath")
	relPath = strings.TrimPrefix(relPath, "/")

	// Prevent directory traversal attacks while allowing safe subdirectories (like workOrderId/image.jpg)
	if strings.Contains(relPath, "..") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid filepath"})
		return
	}

	filePath := filepath.Join(config.AppConfig.UploadDir, relPath)

	// Instruct clients (mobile apps, browsers, proxies) to cache the image permanently (1 year, immutable)
	// This eliminates network downloads on subsequent renders, loading in 0 milliseconds.
	c.Header("Cache-Control", "public, max-age=31536000, immutable")

	// c.File uses Linux zero-copy sendfile syscall under the hood and handles 404 automatically
	c.File(filePath)
}
