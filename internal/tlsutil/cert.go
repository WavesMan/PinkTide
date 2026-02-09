package tlsutil

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"log/slog"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Result 返回证书路径与是否新生成。
type Result struct {
	CertFile  string
	KeyFile   string
	Generated bool
}

// EnsureCertificate 优先使用本地证书，缺失时自动生成自签证书。
func EnsureCertificate(certFile, keyFile, certDir, listenAddr string, logger *slog.Logger) (Result, error) {
	certFile = strings.TrimSpace(certFile)
	keyFile = strings.TrimSpace(keyFile)
	certDir = strings.TrimSpace(certDir)

	if certFile == "" && keyFile == "" {
		if certDir == "" {
			certDir = "certs"
		}
		certFile = filepath.Join(certDir, "cert.pem")
		keyFile = filepath.Join(certDir, "key.pem")
	}

	certExists := fileExists(certFile)
	keyExists := fileExists(keyFile)
	if certExists && keyExists {
		if logger != nil {
			logger.Info("tls cert loaded", "cert_file", certFile, "key_file", keyFile)
		}
		return Result{CertFile: certFile, KeyFile: keyFile, Generated: false}, nil
	}

	if err := os.MkdirAll(filepath.Dir(certFile), 0o755); err != nil {
		return Result{}, fmt.Errorf("create cert dir failed: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(keyFile), 0o755); err != nil {
		return Result{}, fmt.Errorf("create key dir failed: %w", err)
	}

	certPEM, keyPEM, err := generateSelfSigned(listenAddr)
	if err != nil {
		return Result{}, err
	}

	if err := os.WriteFile(certFile, certPEM, 0o644); err != nil {
		return Result{}, fmt.Errorf("write cert failed: %w", err)
	}
	if err := os.WriteFile(keyFile, keyPEM, 0o600); err != nil {
		return Result{}, fmt.Errorf("write key failed: %w", err)
	}

	if logger != nil {
		logger.Info("tls cert generated", "cert_file", certFile, "key_file", keyFile)
	}
	return Result{CertFile: certFile, KeyFile: keyFile, Generated: true}, nil
}

// fileExists 判断文件是否存在。
func fileExists(path string) bool {
	if path == "" {
		return false
	}
	_, err := os.Stat(path)
	return err == nil
}

// generateSelfSigned 生成自签证书与私钥。
func generateSelfSigned(listenAddr string) ([]byte, []byte, error) {
	now := time.Now()
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, nil, fmt.Errorf("create serial failed: %w", err)
	}

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, fmt.Errorf("generate key failed: %w", err)
	}

	hosts := hostsFromAddr(listenAddr)
	if len(hosts) == 0 {
		hosts = []string{"localhost"}
	}

	tmpl := x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName: hosts[0],
		},
		NotBefore:             now.Add(-1 * time.Hour),
		NotAfter:              now.Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	for _, h := range hosts {
		if ip := net.ParseIP(h); ip != nil {
			tmpl.IPAddresses = append(tmpl.IPAddresses, ip)
			continue
		}
		tmpl.DNSNames = append(tmpl.DNSNames, h)
	}

	der, err := x509.CreateCertificate(rand.Reader, &tmpl, &tmpl, &key.PublicKey, key)
	if err != nil {
		return nil, nil, fmt.Errorf("create cert failed: %w", err)
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	return certPEM, keyPEM, nil
}

// hostsFromAddr 将监听地址转换为证书可用的主机列表。
func hostsFromAddr(addr string) []string {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return nil
	}

	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		host = addr
	}
	host = strings.TrimSpace(host)
	if host == "" || host == "0.0.0.0" || host == "::" {
		return []string{"localhost", "127.0.0.1", "::1"}
	}
	return []string{host}
}
