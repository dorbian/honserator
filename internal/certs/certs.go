package certs

import (
    "crypto/rand"
    "crypto/rsa"
    "crypto/x509"
    "crypto/x509/pkix"
    "encoding/pem"
    "math/big"
    "time"
)

// GenerateSelfSignedCert returns a self-signed certificate and key in PEM format.
// dnsNames will be added as SANs; the first one is used as CommonName if cn is empty.
func GenerateSelfSignedCert(cn string, dnsNames []string) (certPEM []byte, keyPEM []byte, err error) {
    if cn == "" && len(dnsNames) > 0 {
        cn = dnsNames[0]
    }

    priv, err := rsa.GenerateKey(rand.Reader, 2048)
    if err != nil {
        return nil, nil, err
    }

    serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
    serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
    if err != nil {
        return nil, nil, err
    }

    tmpl := x509.Certificate{
        SerialNumber: serialNumber,
        Subject: pkix.Name{
            CommonName: cn,
        },
        NotBefore:             time.Now().Add(-5 * time.Minute),
        NotAfter:              time.Now().AddDate(1, 0, 0), // 1 year
        KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
        ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
        BasicConstraintsValid: true,
        DNSNames:              dnsNames,
    }

    derBytes, err := x509.CreateCertificate(rand.Reader, &tmpl, &tmpl, &priv.PublicKey, priv)
    if err != nil {
        return nil, nil, err
    }

    certPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
    keyPEM = pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})
    return certPEM, keyPEM, nil
}
