package selfmanagerpki

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"github.com/Orange-OpenSource/nifikop/api/v1alpha1"
	"github.com/Orange-OpenSource/nifikop/pkg/util/pki"
	"log"
	"math/big"
	"net"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

var subject = pkix.Name{
	Organization:  []string{"Orange"},
	Country:       []string{"FR"},
	Province:      []string{""},
	Locality:      []string{"Paris"},
	StreetAddress: []string{"78 Rue Olivier de Serres"},
	PostalCode:    []string{"75015"},
}

type SelfManager interface {
	pki.Manager
}

type selfManager struct {
	client  client.Client
	cluster *v1alpha1.NifiCluster

	// TODO PEMs or objects ?
	caCert *x509.Certificate
	caKey  *rsa.PrivateKey
}

// Return a new fully instantiated SelfManager struct
func New(client client.Client, cluster *v1alpha1.NifiCluster) SelfManager {
	selfmanager := selfManager{
		client:  client,
		cluster: cluster,
	}

	//get our ca and server certificate
	err := selfmanager.setupCA()
	if err != nil {
		log.Panic(err)
	}

	return selfmanager
}

// Sets up the caCert & caKey variables by setting up a new self signed CA
func (s *selfManager) setupCA() (err error) {
	// set up our CA certificate
	s.caCert = &x509.Certificate{
		SerialNumber:          big.NewInt(2019),
		Subject:               subject,
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		IsCA:                  true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}

	// create our private and public key
	s.caKey, err = rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return
	}

	// create the CA
	caBytes, err := x509.CreateCertificate(rand.Reader, s.caCert, s.caCert, &s.caKey.PublicKey, s.caKey)
	if err != nil {
		return
	}

	// pem encode
	caPEM := new(bytes.Buffer)
	err = pem.Encode(caPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: caBytes,
	})
	if err != nil {
		return
	}

	caPrivKeyPEM := new(bytes.Buffer)
	err = pem.Encode(caPrivKeyPEM, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(s.caKey),
	})
	if err != nil {
		return
	}

	return nil
}

// TODO PEM or Bytes + params ?
// Generate one cert from selfmanager's CA
func (s *selfManager) generateCert() (certPEM *bytes.Buffer, certPrivKeyPEM *bytes.Buffer, err error) {
	// set up our server certificate
	cert := &x509.Certificate{
		SerialNumber: big.NewInt(2019),
		Subject:      subject,
		IPAddresses:  []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().AddDate(10, 0, 0),
		SubjectKeyId: []byte{1, 2, 3, 4, 6},
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}

	certPrivKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, nil, err
	}

	certBytes, err := x509.CreateCertificate(rand.Reader, cert, s.caCert, &certPrivKey.PublicKey, s.caKey)
	if err != nil {
		return nil, nil, err
	}

	certPEM = new(bytes.Buffer)
	err = pem.Encode(certPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certBytes,
	})
	if err != nil {
		return nil, nil, err
	}

	certPrivKeyPEM = new(bytes.Buffer)
	err = pem.Encode(certPrivKeyPEM, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(certPrivKey),
	})
	if err != nil {
		return nil, nil, err
	}

	return
}
