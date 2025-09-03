package server

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"github.com/rstms/winexec/message"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
)

func (s *WinexecServer) handleFileGet(w http.ResponseWriter, r *http.Request) {
	if Verbose {
		log.Printf("%s -> winexec %s %s\n", r.RemoteAddr, r.Method, r.URL.Path)
	}
	var request message.FileGetRequest
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		Warning("%v", Fatal(err))
		fail(w, r, "failed decoding request", http.StatusBadRequest)
		return
	}
	if Verbose {
		log.Printf("%+v\n", request)
	}

	count, err := getFile(request.Pathname, request.URL, request.CA, request.Cert, request.Key)
	if err != nil {
		Warning("%v", Fatal(err))
		fail(w, r, "get request failed", http.StatusBadRequest)
		return
	}

	s.setAutoDelete(request.Pathname, request.AutoDeleteSeconds)

	response := message.FileGetResponse{
		Success:  true,
		Message:  "downloaded",
		Pathname: request.Pathname,
		Bytes:    count,
	}
	succeed(w, r, &response)
}

func getFile(dstPathname, srcURL string, ca, cert, key []byte) (int64, error) {
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
