package geturl

import (
	"crypto/tls"
	"crypto/x509"
	"io"
	"net/http"
	"net/url"
	"os"
)

func GetURL(dstPathname, srcURL string, ca, cert, key []byte) (int64, error) {
	client := http.Client{}
	parsedURL, err := url.Parse(srcURL)
	if err != nil {
		return 0, Fatal(err)
	}
	if parsedURL.Scheme == "https" {
		var caCertPool *x509.CertPool
		if len(ca) > 0 {
			caCertPool = x509.NewCertPool()
			ok := caCertPool.AppendCertsFromPEM(ca)
			if !ok {
				return 0, Fatalf("failed appending ca to cert pool")
			}
		} else {
			caCertPool, err = x509.SystemCertPool()
			if err != nil {
				return 0, Fatalf("failed reading SystemCertPool: %v", err)
			}
		}

		transport := http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs: caCertPool,
			},
		}

		if len(cert) > 0 && len(key) > 0 {
			clientCert, err := tls.X509KeyPair(cert, key)
			if err != nil {
				return 0, Fatal(err)
			}
			transport.TLSClientConfig.Certificates = []tls.Certificate{clientCert}

		}

		client.Transport = &transport
	}

	response, err := client.Get(parsedURL.String())
	if err != nil {
		return 0, Fatal(err)
	}
	defer response.Body.Close()
	ofp, err := os.Create(dstPathname)
	if err != nil {
		return 0, Fatal(err)
	}
	defer ofp.Close()
	count, err := io.Copy(ofp, response.Body)
	if err != nil {
		return 0, Fatal(err)
	}
	return count, nil
}
