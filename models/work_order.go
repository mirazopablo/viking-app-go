package models

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Valid repair status values according to system rules and openapi.yaml
const (
	StatusReceived   = "RECEIVED"
	StatusInQueue    = "IN_QUEUE"
	StatusInProgress = "IN_PROGRESS"
	StatusDone       = "DONE"
	StatusWithdrawn  = "WITHDRAWN"
)

// WorkOrder represents a repair job or technical intervention in the workshop.
type WorkOrder struct {
	ID                   string         `gorm:"type:uuid;primary_key;" json:"id"`
	ClientID             *string        `gorm:"type:uuid;index" json:"clientId,omitempty"`
	Client               User           `gorm:"foreignKey:ClientID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"-"`
	DeviceID             *string        `gorm:"type:uuid;index" json:"deviceId,omitempty"`
	Device               Device         `gorm:"foreignKey:DeviceID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"-"`
	SecurityCodeHash     string         `gorm:"type:varchar(255);column:security_code_hash;" json:"-"`
	ClientNameSnapshot   string         `gorm:"type:varchar(150);" json:"clientName"`
	ClientDniSnapshot    int32          `json:"clientDni"`
	ClientPhoneSnapshot  string         `gorm:"type:varchar(50);" json:"clientPhone"`
	DeviceBrandSnapshot  string         `gorm:"type:varchar(100);" json:"deviceBrand"`
	DeviceModelSnapshot  string         `gorm:"type:varchar(150);" json:"deviceModel"`
	DeviceSerialSnapshot string         `gorm:"type:varchar(150);" json:"deviceSerialNumber"`
	StaffID              *string        `gorm:"type:uuid;index" json:"staffId,omitempty"`
	Staff                *User          `gorm:"foreignKey:StaffID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"-"`
	IssueDescription     string         `gorm:"type:text;not null" json:"issueDescription"`
	Notes                string         `gorm:"type:text;" json:"notes,omitempty"`
	RepairStatus         string         `gorm:"type:varchar(50);not null;index" json:"repairStatus"`
	CreatedAt            time.Time `json:"createdAt"`
	UpdatedAt            time.Time `json:"updatedAt"`
}

// BeforeCreate hooks into GORM to generate a UUID prior to database insertion.
func (wo *WorkOrder) BeforeCreate(tx *gorm.DB) (err error) {
	if wo.ID == "" {
		wo.ID = uuid.New().String()
	}
	return
}

// WorkOrderCreateRequest defines the JSON contract for opening a new repair job.
type WorkOrderCreateRequest struct {
	ClientID         string `json:"clientId" binding:"required"`
	DeviceID         string `json:"deviceId" binding:"required"`
	IssueDescription string `json:"issueDescription" binding:"required,min=2,max=200"`
	Notes            string `json:"notes"`
	RepairStatus     string `json:"repairStatus" binding:"required"`
}

// WorkOrderStatusUpdateRequestDto defines the JSON contract for updating repair status.
type WorkOrderStatusUpdateRequestDto struct {
	RepairStatus string `json:"repairStatus" binding:"required"`
}

// WorkOrderResponseDto defines a flattened, rich representation of a work order for consumers.
type WorkOrderResponseDto struct {
	ID                 string `json:"id"`
	SecurityCode       string `json:"securityCode,omitempty"`
	ClientID           string `json:"clientId"`
	ClientName         string `json:"clientName"`
	ClientDni          int32  `json:"clientDni"`
	ClientPhone        string `json:"clientPhone"`
	DeviceID           string `json:"deviceId"`
	DeviceBrand        string `json:"deviceBrand"`
	DeviceModel        string `json:"deviceModel"`
	DeviceSerialNumber string `json:"deviceSerialNumber"`
	StaffID            string `json:"staffId,omitempty"`
	StaffName          string `json:"staffName,omitempty"`
	IssueDescription   string `json:"issueDescription"`
	Notes              string `json:"notes,omitempty"`
	RepairStatus       string `json:"repairStatus"`
	CreatedAt          string `json:"createdAt"`
	UpdatedAt          string `json:"updatedAt"`
}

// WorkOrderPublicQueryRequestDto defines the input payload for client public inquiries.
type WorkOrderPublicQueryRequestDto struct {
	ID           string `json:"id" binding:"required"`
	SecurityCode string `json:"securityCode" binding:"required"`
}

// WorkOrderPublicDniQueryRequestDto defines the input payload for client public inquiries by DNI.
type WorkOrderPublicDniQueryRequestDto struct {
	ClientDni    int32  `json:"clientDni" binding:"required"`
	SecurityCode string `json:"securityCode" binding:"required"`
}

// WorkOrderPublicStatusResponseDto aggregates the work order details and its diagnostic timeline.
type WorkOrderPublicStatusResponseDto struct {
	WorkOrder        WorkOrderResponseDto         `json:"workOrder"`
	DiagnosticPoints []DiagnosticPointResponseDto `json:"diagnosticPoints"`
}

// ValidateRepairStatus checks if the provided string is a valid enum value.
func ValidateRepairStatus(status string) error {
	s := strings.ToUpper(strings.TrimSpace(status))
	switch s {
	case StatusReceived, StatusInQueue, StatusInProgress, StatusDone, StatusWithdrawn:
		return nil
	default:
		return errors.New("invalid repairStatus: must be one of RECEIVED, IN_QUEUE, IN_PROGRESS, DONE, WITHDRAWN")
	}
}
