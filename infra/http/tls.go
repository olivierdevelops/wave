package http

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"log"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// TLSConfig holds the parameters for self-signed certificate generation.
// This is a pure-infra type; callers map their own config struct into it.
type TLSConfig struct {
	SSLCertfile  string
	SSLKeyfile   string
	Organization []string
	DNSNames     []string
	CommonName   string
}

// GenerateSelfSigned creates a self-signed TLS certificate + key at the
// paths specified in cfg. It is idempotent only if the caller checks for
// file existence before calling.
func GenerateSelfSigned(cfg TLSConfig) error {
	log.Println("Generating HTTPS config")

	if cfg.SSLCertfile == "" {
		return fmt.Errorf("missing 'ssl_certfile' path")
	}
	if cfg.SSLKeyfile == "" {
		return fmt.Errorf("missing 'ssl_keyfile' path")
	}

	certDir := filepath.Dir(cfg.SSLCertfile)
	keyDir := filepath.Dir(cfg.SSLKeyfile)
	if err := os.MkdirAll(certDir, 0755); err != nil {
		return err
	}
	if err := os.MkdirAll(keyDir, 0755); err != nil {
		return err
	}

	ips, err := localIPs()
	if err != nil {
		return err
	}

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return err
	}

	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return err
	}

	org := cfg.Organization
	if len(org) == 0 {
		org = []string{"Local Network"}
	}

	dnsNames := cfg.DNSNames
	if len(dnsNames) == 0 {
		dnsNames = []string{"localhost", "*.local"}
	}

	cn := strings.TrimSpace(cfg.CommonName)
	if cn == "" {
		cn = "localhost"
	}

	tmpl := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: org,
			CommonName:   cn,
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(1, 0, 0),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IPAddresses:           ips,
		DNSNames:              dnsNames,
	}

	certBytes, err := x509.CreateCertificate(rand.Reader, &tmpl, &tmpl, &privateKey.PublicKey, privateKey)
	if err != nil {
		return err
	}

	certFile, err := os.Create(cfg.SSLCertfile)
	if err != nil {
		return err
	}
	defer certFile.Close()
	if err := pem.Encode(certFile, &pem.Block{Type: "CERTIFICATE", Bytes: certBytes}); err != nil {
		return err
	}

	keyFile, err := os.Create(cfg.SSLKeyfile)
	if err != nil {
		return err
	}
	defer keyFile.Close()
	if err := pem.Encode(keyFile, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	}); err != nil {
		return err
	}

	log.Println("HTTPS certificate and key generation completed successfully")
	return nil
}

func localIPs() ([]net.IP, error) {
	ips := []net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("::1")}
	interfaces, err := net.Interfaces()
	if err != nil {
		return ips, err
	}
	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip != nil && !ip.IsLoopback() {
				ips = append(ips, ip)
			}
		}
	}
	return ips, nil
}
