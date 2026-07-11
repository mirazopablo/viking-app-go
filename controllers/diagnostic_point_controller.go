package controllers

import (
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/png"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/mirazopablo/viking-app-go/config"
	"github.com/mirazopablo/viking-app-go/services"
)

// DiagnosticPointController handles multimedia evidence and diagnostic log endpoints.
type DiagnosticPointController struct {
	service services.DiagnosticPointService
}

// NewDiagnosticPointController instantiates a new DiagnosticPointController.
func NewDiagnosticPointController(service services.DiagnosticPointService) *DiagnosticPointController {
	return &DiagnosticPointController{service: service}
}

// AddDiagnosticPoint godoc
// @Summary Agregar punto de diagnóstico
// @Description Agrega un nuevo punto de diagnóstico a una orden de trabajo
// @Tags DiagnosticPoint
// @ID addDiagnosticPoint
// @Accept multipart/form-data
// @Produce json
// @Param workOrderId formData string false "Work Order UUID"
// @Param clientId formData string false "Client UUID"
// @Param diagnosticPoint formData string true "Diagnostic description or JSON string"
// @Param file formData file true "Attached image file"
// @Success 200 {object} models.DiagnosticPointResponseDto "OK"
// @Security bearer-jwt
// @Router /api/diagnostic-points/add [post]
func (dpc *DiagnosticPointController) AddDiagnosticPoint(c *gin.Context) {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File upload is required: " + err.Error()})
		return
	}

	workOrderID := c.PostForm("workOrderId")
	clientID := c.PostForm("clientId")
	description := c.PostForm("description")
	if description == "" {
		description = c.PostForm("diagnosticPoint")
	}

	// Support JSON string sent in diagnosticPoint field (common in Axios/Spring Boot multipart requests)
	if description != "" && strings.HasPrefix(strings.TrimSpace(description), "{") {
		var payload struct {
			WorkOrderID string `json:"workOrderId"`
			ClientID    string `json:"clientId"`
			Description string `json:"description"`
			Diagnostic  string `json:"diagnosticPoint"`
		}
		if err = json.Unmarshal([]byte(description), &payload); err == nil {
			if workOrderID == "" {
				workOrderID = payload.WorkOrderID
			}
			if clientID == "" {
				clientID = payload.ClientID
			}
			if payload.Description != "" {
				description = payload.Description
			} else if payload.Diagnostic != "" {
				description = payload.Diagnostic
			}
		}
	}

	if strings.TrimSpace(workOrderID) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "workOrderId is required"})
		return
	}

	workOrderDir := filepath.Join(config.AppConfig.UploadDir, workOrderID)
	if err = os.MkdirAll(workOrderDir, os.ModePerm); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create work order upload directory"})
		return
	}

	dpID := uuid.New().String()
	timestamp := time.Now().Format("20060102_150405")
	fileExt := filepath.Ext(fileHeader.Filename)
	newFileName := fmt.Sprintf("%s_%s%s", timestamp, dpID, fileExt)
	savePath := filepath.Join(workOrderDir, newFileName)

	if err = c.SaveUploadedFile(fileHeader, savePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file: " + err.Error()})
		return
	}

	// Optimize image on disk (resize max dimension to 1024px and compress at quality 45)
	// Keeps storage usage minimal and allows instant downloads over remote ~1 Mbps links.
	optimizeImage(savePath, 1024, 45)

	imageURL := "/auth/uploads/" + workOrderID + "/" + newFileName

	saved, err := dpc.service.AddDiagnosticPoint(dpID, workOrderID, clientID, description, imageURL)
	if err != nil {
		if errors.Is(err, services.ErrWorkOrderNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Associated work order not found"})
			return
		}
		if errors.Is(err, services.ErrUserNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Associated client not found"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, saved)
}

// GetDiagnosticPointsByWorkOrderAndClient godoc
// @Summary Obtener puntos de diagnóstico por ID de orden de trabajo
// @Description Obtiene una lista de puntos de diagnóstico por ID de orden de trabajo y Cliente Asociado a la orden de trabajo
// @Tags DiagnosticPoint
// @ID getDiagnosticPointsByWorkOrderAndClient
// @Produce json
// @Param workOrderId path string true "Work Order UUID" format(uuid)
// @Param clientId path string true "Client UUID" format(uuid)
// @Success 200 {array} models.DiagnosticPointResponseDto "OK"
// @Security bearer-jwt
// @Router /api/diagnostic-points/by-work-order/{workOrderId}/client/{clientId} [get]
func (dpc *DiagnosticPointController) GetDiagnosticPointsByWorkOrderAndClient(c *gin.Context) {
	workOrderID := c.Param("workOrderId")
	clientID := c.Param("clientId")

	points, err := dpc.service.GetByWorkOrderAndClient(workOrderID, clientID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, points)
}

// GetDiagnosticPointsByWorkOrder godoc
// @Summary Obtener puntos de diagnóstico por ID de orden de trabajo
// @Description Obtiene una lista de puntos de diagnóstico por ID de orden de trabajo sin acoplamiento a clientId
// @Tags DiagnosticPoint
// @ID getDiagnosticPointsByWorkOrder
// @Produce json
// @Param workOrderId path string true "Work Order UUID" format(uuid)
// @Success 200 {array} models.DiagnosticPointResponseDto "OK"
// @Security bearer-jwt
// @Router /api/diagnostic-points/by-work-order/{workOrderId} [get]
func (dpc *DiagnosticPointController) GetDiagnosticPointsByWorkOrder(c *gin.Context) {
	workOrderID := c.Param("workOrderId")

	points, err := dpc.service.GetByWorkOrderAndClient(workOrderID, "")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, points)
}

// DeleteDiagnosticPoint godoc
// @Summary Eliminar Punto de Diagnóstico
// @Description Borra una evidencia multimedia por UUID
// @Tags DiagnosticPoint
// @ID deleteDiagnosticPoint
// @Param id path string true "Diagnostic Point UUID" format(uuid)
// @Success 200 "OK"
// @Security bearer-jwt
// @Router /api/diagnostic-points/delete/{id} [delete]
func (dpc *DiagnosticPointController) DeleteDiagnosticPoint(c *gin.Context) {
	id := c.Param("id")
	if err := dpc.service.DeleteDiagnosticPoint(id); err != nil {
		if errors.Is(err, services.ErrDiagnosticPointNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Diagnostic point not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusOK)
}

// optimizeImage decodes, downsamples (if > maxDim), and re-encodes an image as JPEG at the given quality.
// Uses pure Go standard library (image, image/jpeg, image/png) without external dependencies.
func optimizeImage(filePath string, maxDim int, quality int) {
	file, err := os.Open(filePath)
	if err != nil {
		return
	}
	img, _, err := image.Decode(file)
	file.Close()
	if err != nil {
		// Not an image or unsupported format, leave original untouched
		return
	}

	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	if w > maxDim || h > maxDim {
		var newW, newH int
		if w > h {
			newW = maxDim
			newH = (h * maxDim) / w
		} else {
			newH = maxDim
			newW = (w * maxDim) / h
		}
		if newW <= 0 {
			newW = 1
		}
		if newH <= 0 {
			newH = 1
		}

		dst := image.NewRGBA(image.Rect(0, 0, newW, newH))
		for y := 0; y < newH; y++ {
			for x := 0; x < newW; x++ {
				srcX := bounds.Min.X + (x * w / newW)
				srcY := bounds.Min.Y + (y * h / newH)
				dst.Set(x, y, img.At(srcX, srcY))
			}
		}
		img = dst
	}

	out, err := os.Create(filePath)
	if err != nil {
		return
	}
	defer out.Close()

	_ = jpeg.Encode(out, img, &jpeg.Options{Quality: quality})
}
