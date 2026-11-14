package afip

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"os"
	"time"

	"go.mozilla.org/pkcs7"
)

// GenerateTRA creates the XML content for the Ticket de Requerimiento de Acceso (TRA).
func GenerateTRA(service string) string {
	now := time.Now()
	// ARCA requires ISO-8601 timestamps. We add a 10-minute generation margin to avoid clock-skew issues.
	genTime := now.Add(-10 * time.Minute).Format(time.RFC3339)
	expTime := now.Add(10 * time.Minute).Format(time.RFC3339)
	uniqueID := now.Unix()

	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<loginTicketRequest version="1.0">
  <header>
    <uniqueId>%d</uniqueId>
    <generationTime>%s</generationTime>
    <expirationTime>%s</expirationTime>
  </header>
  <service>%s</service>
</loginTicketRequest>`, uniqueID, genTime, expTime, service)
}

// SignTRA signs the XML TRA using PKCS#7 / CMS format with the provided RSA private key and certificate.
// Returns the DER-encoded signature formatted as a Base64 string required by WSAA.
func SignTRA(traXML string, certPath string, keyPath string) (string, error) {
	// 1. Read Certificate
	certPEM, err := os.ReadFile(certPath)
	if err != nil {
		return "", fmt.Errorf("failed to read cert file at %s: %w", certPath, err)
	}

	certBlock, _ := pem.Decode(certPEM)
	if certBlock == nil {
		return "", fmt.Errorf("failed to parse PEM block from cert file at %s", certPath)
	}

	cert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return "", fmt.Errorf("failed to parse X509 certificate: %w", err)
	}

	// 2. Read Private Key
	keyPEM, err := os.ReadFile(keyPath)
	if err != nil {
		return "", fmt.Errorf("failed to read key file at %s: %w", keyPath, err)
	}

	keyBlock, _ := pem.Decode(keyPEM)
	if keyBlock == nil {
		return "", fmt.Errorf("failed to parse PEM block from key file at %s", keyPath)
	}

	var privateKey *rsa.PrivateKey
	if parsedKey, err := x509.ParsePKCS1PrivateKey(keyBlock.Bytes); err == nil {
		privateKey = parsedKey
	} else if parsedKey, err := x509.ParsePKCS8PrivateKey(keyBlock.Bytes); err == nil {
		var ok bool
		privateKey, ok = parsedKey.(*rsa.PrivateKey)
		if !ok {
			return "", fmt.Errorf("parsed PKCS#8 private key is not RSA")
		}
	} else {
		return "", fmt.Errorf("unsupported RSA private key format: %w", err)
	}

	// 3. Initialize SignedData structure
	toBeSigned, err := pkcs7.NewSignedData([]byte(traXML))
	if err != nil {
		return "", fmt.Errorf("failed to initialize pkcs7 SignedData: %w", err)
	}

	// 4. Attach signer (Certificate + Private Key)
	if err := toBeSigned.AddSigner(cert, privateKey, pkcs7.SignerInfoConfig{}); err != nil {
		return "", fmt.Errorf("failed to add signer to pkcs7: %w", err)
	}

	// 5. Finish signing and get DER bytes
	derBytes, err := toBeSigned.Finish()
	if err != nil {
		return "", fmt.Errorf("failed to finish pkcs7 signature: %w", err)
	}

	// 6. Return Base64 encoded string
	return base64.StdEncoding.EncodeToString(derBytes), nil
}
