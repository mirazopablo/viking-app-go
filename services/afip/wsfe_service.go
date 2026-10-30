package afip

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/mirazopablo/viking-app-go/config"
	"github.com/mirazopablo/viking-app-go/models"
)

// WSFEService manages electronic invoice operations against ARCA WSFEV1.
type WSFEService struct {
	wsaaService *WSAAService
	client      *http.Client
}

// Global instance
var defaultWSFE *WSFEService

// GetWSFEService returns the singleton instance of WSFEService.
func GetWSFEService() *WSFEService {
	if defaultWSFE == nil {
		defaultWSFE = &WSFEService{
			wsaaService: GetWSAAService(),
			client: &http.Client{
				Timeout: 20 * time.Second,
			},
		}
	}
	return defaultWSFE
}

// SOAP Fault for WSFE
type WSFEFault struct {
	XMLName xml.Name `xml:"Fault"`
	Code    string   `xml:"faultcode"`
	String  string   `xml:"faultstring"`
}

// GetLastVoucherNumber queries WSFE for the last authorized voucher number for a point of sale and voucher type.
func (s *WSFEService) GetLastVoucherNumber(puntoVenta int, tipoCmp int) (int64, error) {
	auth, err := s.wsaaService.GetAuthToken("wsfe")
	if err != nil {
		return 0, fmt.Errorf("failed to get WSAA auth token: %w", err)
	}

	cuitInt, err := strconv.ParseInt(config.AppConfig.AfipCUIT, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid AFIP_CUIT configured: %w", err)
	}

	soapEnvelope := fmt.Sprintf(`<?xml version="1.0" encoding="utf-8"?>
<soap:Envelope xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns:xsd="http://www.w3.org/2001/XMLSchema" xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
    <FECompUltimoAutorizado xmlns="http://ar.gov.afip.dif.FEV1/">
      <Auth>
        <Token>%s</Token>
        <Sign>%s</Sign>
        <Cuit>%d</Cuit>
      </Auth>
      <PtoVta>%d</PtoVta>
      <CbteTipo>%d</CbteTipo>
    </FECompUltimoAutorizado>
  </soap:Body>
</soap:Envelope>`, auth.Token, auth.Sign, cuitInt, puntoVenta, tipoCmp)

	req, err := http.NewRequest("POST", config.AppConfig.AfipWSFEURL, bytes.NewBufferString(soapEnvelope))
	if err != nil {
		return 0, fmt.Errorf("failed to create SOAP request for FECompUltimoAutorizado: %w", err)
	}

	req.Header.Set("Content-Type", "text/xml; charset=utf-8")
	req.Header.Set("SOAPAction", "http://ar.gov.afip.dif.FEV1/FECompUltimoAutorizado")

	resp, err := s.client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to call FECompUltimoAutorizado: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("failed to read FECompUltimoAutorizado response: %w", err)
	}

	var envelope struct {
		XMLName xml.Name `xml:"Envelope"`
		Body    struct {
			Fault                          *WSFEFault `xml:"Fault"`
			FECompUltimoAutorizadoResponse struct {
				FECompUltimoAutorizadoResult struct {
					CbteNro int64 `xml:"CbteNro"`
					Errors  struct {
						Err []struct {
							Code int    `xml:"Code"`
							Msg  string `xml:"Msg"`
						} `xml:"Err"`
					} `xml:"Errors"`
				} `xml:"FECompUltimoAutorizadoResult"`
			} `xml:"FECompUltimoAutorizadoResponse"`
		} `xml:"Body"`
	}

	if err := xml.Unmarshal(respBytes, &envelope); err != nil {
		return 0, fmt.Errorf("failed to unmarshal FECompUltimoAutorizado response: %w (raw: %s)", err, string(respBytes))
	}

	if envelope.Body.Fault != nil {
		return 0, fmt.Errorf("WSFE Fault: %s", envelope.Body.Fault.String)
	}

	res := envelope.Body.FECompUltimoAutorizadoResponse.FECompUltimoAutorizadoResult
	if len(res.Errors.Err) > 0 {
		return 0, fmt.Errorf("ARCA error: [%d] %s", res.Errors.Err[0].Code, res.Errors.Err[0].Msg)
	}

	return res.CbteNro, nil
}

// RequestCAE authorizes a Factura C (Tipo 11) using FECAESolicitar.
func (s *WSFEService) RequestCAE(reqData models.CreateInvoiceRequest) (*models.Invoice, error) {
	auth, err := s.wsaaService.GetAuthToken("wsfe")
	if err != nil {
		return nil, fmt.Errorf("failed to get WSAA auth token: %w", err)
	}

	cuitInt, err := strconv.ParseInt(config.AppConfig.AfipCUIT, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid AFIP_CUIT configured: %w", err)
	}

	puntoVenta := config.AppConfig.AfipPuntoVenta
	tipoCmp := 11 // Factura C

	// 1. Get next voucher number
	lastNro, err := s.GetLastVoucherNumber(puntoVenta, tipoCmp)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve last voucher number: %w", err)
	}
	nextNro := lastNro + 1

	today := time.Now().Format("20060102")

	// Service dates handling
	serviceXML := ""
	if reqData.Concepto == 2 || reqData.Concepto == 3 {
		desde := today
		hasta := today
		vto := today

		if reqData.FchServDesde != nil && *reqData.FchServDesde != "" {
			desde = *reqData.FchServDesde
		}
		if reqData.FchServHasta != nil && *reqData.FchServHasta != "" {
			hasta = *reqData.FchServHasta
		}
		if reqData.FchVtoPago != nil && *reqData.FchVtoPago != "" {
			vto = *reqData.FchVtoPago
		}

		serviceXML = fmt.Sprintf(`
        <FchServDesde>%s</FchServDesde>
        <FchServHasta>%s</FchServHasta>
        <FchVtoPago>%s</FchVtoPago>`, desde, hasta, vto)
	}

	// Condicion Frente al IVA del Receptor (RG 5616)
	// Default: 5 (Consumidor Final)
	condicionIvaReceptorId := 5
	if reqData.CondicionIvaReceptorId != nil && *reqData.CondicionIvaReceptorId > 0 {
		condicionIvaReceptorId = *reqData.CondicionIvaReceptorId
	} else if reqData.DocTipo == 80 {
		// Si es CUIT y no se especifico, asumir Responsable Inscripto (1)
		condicionIvaReceptorId = 1
	}

	// 2. Build FECAESolicitar XML
	soapEnvelope := fmt.Sprintf(`<?xml version="1.0" encoding="utf-8"?>
<soap:Envelope xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns:xsd="http://www.w3.org/2001/XMLSchema" xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
    <FECAESolicitar xmlns="http://ar.gov.afip.dif.FEV1/">
      <Auth>
        <Token>%s</Token>
        <Sign>%s</Sign>
        <Cuit>%d</Cuit>
      </Auth>
      <FeCAEReq>
        <FeCabReq>
          <CantReg>1</CantReg>
          <PtoVta>%d</PtoVta>
          <CbteTipo>%d</CbteTipo>
        </FeCabReq>
        <FeDetReq>
          <FECAEDetRequest>
            <Concepto>%d</Concepto>
            <DocTipo>%d</DocTipo>
            <DocNro>%d</DocNro>
            <CbteDesde>%d</CbteDesde>
            <CbteHasta>%d</CbteHasta>
            <CbteFch>%s</CbteFch>
            <ImpTotal>%.2f</ImpTotal>
            <ImpTotConc>0</ImpTotConc>
            <ImpNeto>%.2f</ImpNeto>
            <ImpOpEx>0</ImpOpEx>
            <ImpTrib>0</ImpTrib>
            <ImpIVA>0</ImpIVA>%s
            <MonId>PES</MonId>
            <MonCotiz>1</MonCotiz>
            <CondicionIVAReceptorId>%d</CondicionIVAReceptorId>
          </FECAEDetRequest>
        </FeDetReq>
      </FeCAEReq>
    </FECAESolicitar>
  </soap:Body>
</soap:Envelope>`,
		auth.Token, auth.Sign, cuitInt,
		puntoVenta, tipoCmp,
		reqData.Concepto, reqData.DocTipo, reqData.DocNro,
		nextNro, nextNro, today,
		reqData.ImporteTotal, reqData.ImporteTotal,
		serviceXML,
		condicionIvaReceptorId,
	)

	req, err := http.NewRequest("POST", config.AppConfig.AfipWSFEURL, bytes.NewBufferString(soapEnvelope))
	if err != nil {
		return nil, fmt.Errorf("failed to create SOAP request for FECAESolicitar: %w", err)
	}

	req.Header.Set("Content-Type", "text/xml; charset=utf-8")
	req.Header.Set("SOAPAction", "http://ar.gov.afip.dif.FEV1/FECAESolicitar")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call FECAESolicitar: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read FECAESolicitar response: %w", err)
	}

	// 3. Parse FECAESolicitar Response
	var envelope struct {
		XMLName xml.Name `xml:"Envelope"`
		Body    struct {
			Fault                *WSFEFault `xml:"Fault"`
			FECAESolicitarResponse struct {
				FECAESolicitarResult struct {
					FeCabResp struct {
						Resultado string `xml:"Resultado"`
					} `xml:"FeCabResp"`
					FeDetResp struct {
						FECAEDetResponse []struct {
							Concepto  int    `xml:"Concepto"`
							DocTipo   int    `xml:"DocTipo"`
							DocNro    int64  `xml:"DocNro"`
							CbteDesde int64  `xml:"CbteDesde"`
							CbteHasta int64  `xml:"CbteHasta"`
							CbteFch   string `xml:"CbteFch"`
							Resultado string `xml:"Resultado"`
							CAE       string `xml:"CAE"`
							CAEFchVto string `xml:"CAEFchVto"`
							Observaciones struct {
								Obs []struct {
									Code int    `xml:"Code"`
									Msg  string `xml:"Msg"`
								} `xml:"Obs"`
							} `xml:"Observaciones"`
						} `xml:"FECAEDetResponse"`
					} `xml:"FeDetResp"`
					Errors struct {
						Err []struct {
							Code int    `xml:"Code"`
							Msg  string `xml:"Msg"`
						} `xml:"Err"`
					} `xml:"Errors"`
					Events struct {
						Evt []struct {
							Code int    `xml:"Code"`
							Msg  string `xml:"Msg"`
						} `xml:"Evt"`
					} `xml:"Events"`
				} `xml:"FECAESolicitarResult"`
			} `xml:"FECAESolicitarResponse"`
		} `xml:"Body"`
	}

	if err := xml.Unmarshal(respBytes, &envelope); err != nil {
		return nil, fmt.Errorf("failed to unmarshal FECAESolicitar response: %w (raw: %s)", err, string(respBytes))
	}

	if envelope.Body.Fault != nil {
		return nil, fmt.Errorf("WSFE Fault: %s", envelope.Body.Fault.String)
	}

	res := envelope.Body.FECAESolicitarResponse.FECAESolicitarResult
	if len(res.Errors.Err) > 0 {
		return nil, fmt.Errorf("ARCA error: [%d] %s", res.Errors.Err[0].Code, res.Errors.Err[0].Msg)
	}

	if len(res.FeDetResp.FECAEDetResponse) == 0 {
		return nil, fmt.Errorf("no voucher detail response returned by ARCA")
	}

	det := res.FeDetResp.FECAEDetResponse[0]

	obsMsg := ""
	for _, evt := range res.Events.Evt {
		obsMsg += fmt.Sprintf("[Evt %d: %s] ", evt.Code, evt.Msg)
	}
	for _, o := range det.Observaciones.Obs {
		obsMsg += fmt.Sprintf("[Obs %d: %s] ", o.Code, o.Msg)
	}

	invoice := &models.Invoice{
		PuntoVenta:             puntoVenta,
		TipoCmp:                tipoCmp,
		NumeroCmp:              nextNro,
		FechaEmision:           today,
		Concepto:               reqData.Concepto,
		DocTipo:                reqData.DocTipo,
		DocNro:                 reqData.DocNro,
		CondicionIvaReceptorId: condicionIvaReceptorId,
		ImporteTotal:           reqData.ImporteTotal,
		CAE:                    det.CAE,
		CAEVto:                 det.CAEFchVto,
		Resultado:              det.Resultado,
		Observaciones:          obsMsg,
	}

	if det.Resultado == "A" {
		invoice.Status = models.InvoiceStatusApproved
	} else {
		invoice.Status = models.InvoiceStatusRejected
	}

	return invoice, nil
}
