package afip

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/mirazopablo/viking-app-go/config"
	"github.com/mirazopablo/viking-app-go/models"
)

// WSAAService manages authentication tickets from ARCA.
type WSAAService struct {
	mu         sync.RWMutex
	tokenCache map[string]*models.AFIPAuthToken // Key: service name (e.g., "wsfe")
	client     *http.Client
}

// Global WSAA instance
var defaultWSAA *WSAAService
var onceWSAA sync.Once

// GetWSAAService returns the singleton instance of WSAAService.
func GetWSAAService() *WSAAService {
	onceWSAA.Do(func() {
		defaultWSAA = &WSAAService{
			tokenCache: make(map[string]*models.AFIPAuthToken),
			client: &http.Client{
				Timeout: 15 * time.Second,
			},
		}
	})
	return defaultWSAA
}

// WSAAFault represents a SOAP fault response from WSAA.
type WSAAFault struct {
	XMLName xml.Name `xml:"Fault"`
	Code    string   `xml:"faultcode"`
	String  string   `xml:"faultstring"`
}

// LoginTicketResponse represents the XML structure returned inside the WSAA response.
type LoginTicketResponse struct {
	XMLName xml.Name `xml:"loginTicketResponse"`
	Header  struct {
		ExpirationTime string `xml:"expirationTime"`
	} `xml:"header"`
	Credentials struct {
		Token string `xml:"token"`
		Sign  string `xml:"sign"`
	} `xml:"credentials"`
}

// GetAuthToken retrieves a valid Token and Sign for the specified service (e.g. "wsfe").
// Uses a 2-tier cache: in-memory sync.RWMutex first, then PostgreSQL DB.
// Survives server restarts and avoids "coe.alreadyAuthenticated" faults from ARCA.
func (s *WSAAService) GetAuthToken(service string) (*models.AFIPAuthToken, error) {
	env := config.AppConfig.AfipEnv

	// 1. Check in-memory cache
	s.mu.RLock()
	cached, exists := s.tokenCache[service]
	if exists && cached.IsValid() {
		s.mu.RUnlock()
		return cached, nil
	}
	s.mu.RUnlock()

	// Acquire write lock to update cache/DB
	s.mu.Lock()
	defer s.mu.Unlock()

	// Double check memory cache after acquiring lock
	if cached, exists := s.tokenCache[service]; exists && cached.IsValid() {
		return cached, nil
	}

	// 2. Check Database persistence
	var dbToken models.AFIPAuthToken
	if err := config.DB.Where("service = ? AND environment = ?", service, env).First(&dbToken).Error; err == nil {
		if dbToken.IsValid() {
			s.tokenCache[service] = &dbToken
			return &dbToken, nil
		}
	}

	// 3. Generate TRA and request new ticket from WSAA
	traXML := GenerateTRA(service)

	// 4. Sign TRA
	cmsBase64, err := SignTRA(traXML, config.AppConfig.AfipCertPath, config.AppConfig.AfipKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to sign TRA for service %s: %w", service, err)
	}

	// 3. Build SOAP envelope for LoginCms
	soapEnvelope := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<soapenv:Envelope xmlns:soapenv="http://schemas.xmlsoap.org/soap/envelope/" xmlns:wsaa="http://wsaa.view.sua.dvadac.desein.afip.gov">
   <soapenv:Header/>
   <soapenv:Body>
      <wsaa:loginCms>
         <wsaa:in0>%s</wsaa:in0>
      </wsaa:loginCms>
   </soapenv:Body>
</soapenv:Envelope>`, cmsBase64)

	req, err := http.NewRequest("POST", config.AppConfig.AfipWSAAURL, bytes.NewBufferString(soapEnvelope))
	if err != nil {
		return nil, fmt.Errorf("failed to create SOAP request for WSAA: %w", err)
	}

	req.Header.Set("Content-Type", "text/xml; charset=utf-8")
	req.Header.Set("SOAPAction", "")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute WSAA request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read WSAA response: %w", err)
	}

	// 4. Parse SOAP response
	var envelope struct {
		XMLName xml.Name `xml:"Envelope"`
		Body    struct {
			Fault            *WSAAFault `xml:"Fault"`
			LoginCmsResponse struct {
				LoginCmsReturn string `xml:"loginCmsReturn"`
			} `xml:"loginCmsResponse"`
		} `xml:"Body"`
	}

	if err := xml.Unmarshal(respBody, &envelope); err != nil {
		return nil, fmt.Errorf("failed to unmarshal WSAA SOAP response: %w (raw response: %s)", err, string(respBody))
	}

	if envelope.Body.Fault != nil {
		return nil, fmt.Errorf("WSAA Fault: code=%s, detail=%s", envelope.Body.Fault.Code, envelope.Body.Fault.String)
	}

	loginCmsReturn := envelope.Body.LoginCmsResponse.LoginCmsReturn
	if loginCmsReturn == "" {
		return nil, fmt.Errorf("empty loginCmsReturn from WSAA (HTTP %d): %s", resp.StatusCode, string(respBody))
	}

	// 5. Parse LoginTicketResponse inside the returned string
	var ticket LoginTicketResponse
	if err := xml.Unmarshal([]byte(loginCmsReturn), &ticket); err != nil {
		return nil, fmt.Errorf("failed to parse loginTicketResponse XML: %w", err)
	}

	expTime, err := time.Parse(time.RFC3339, ticket.Header.ExpirationTime)
	if err != nil {
		// Fallback parse if format differs
		expTime = time.Now().Add(12 * time.Hour)
	}

	newToken := &models.AFIPAuthToken{
		Service:     service,
		Environment: env,
		Token:       ticket.Credentials.Token,
		Sign:        ticket.Credentials.Sign,
		Expiration:  expTime,
	}

	// Persist or update in database
	var existing models.AFIPAuthToken
	if err := config.DB.Where("service = ? AND environment = ?", service, env).First(&existing).Error; err == nil {
		config.DB.Model(&existing).Updates(map[string]interface{}{
			"token":      newToken.Token,
			"sign":       newToken.Sign,
			"expiration": newToken.Expiration,
		})
	} else {
		config.DB.Create(newToken)
	}

	s.tokenCache[service] = newToken
	return newToken, nil
}
