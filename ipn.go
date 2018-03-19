package amazonpay

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

// IPN ipn message
type IPN struct {
	Type             string
	MessageID        string `json:"MessageId"`
	TopicArn         string
	Timestamp        string
	SignatureVersion string
	Signature        string
	SigningCertURL   string
	UnsubscribeURL   string
	Message          struct {
		NotificationReferenceID string `json:"NotificationReferenceId"`
		NotificationType        string
		SellerID                string `json:"SellerId"`
		ReleaseEnvironment      string
		Version                 string
		NotificationData        string
		Timestamp               string
	}
}

// VerifyIPNRequest verify IPN request message
func VerifyIPNRequest(req *http.Request) (*IPN, bool) {
	if req.Header.Get("x-amz-sns-message-type") != "Notification" {
		return nil, false
	}

	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return nil, false
	}

	var ipn IPN
	if err := json.Unmarshal(body, &ipn); err != nil {
		return nil, false
	}

	if cert := getCert(&ipn); verifyCertSubject(&ipn, cert) && verifySignedString(&ipn, cert) {
		return &ipn, true
	}

	return &ipn, false
}

func getCert(ipn *IPN) *x509.Certificate {
	if resp, err := http.Get(ipn.SigningCertURL); err == nil {
		if body, err := ioutil.ReadAll(resp.Body); err == nil {
			if cert, err := x509.ParseCertificate(body); err == nil {
				return cert
			}
		}
	}
	return nil
}

func verifyCertSubject(ipn *IPN, cert *x509.Certificate) bool {
	return cert.Subject.CommonName == "sns.amazonaws.com"
}

func verifySignedString(ipn *IPN, cert *x509.Certificate) bool {
	canonicalString := fmt.Sprintf("")

	result, _ := json.Marshal(ipn.Message)
	canonicalString += "Message\n" + string(result) + "\n"

	if ipn.MessageID != "" {
		canonicalString += "MessageId\n" + ipn.MessageID + "\n"
	}
	if ipn.Timestamp != "" {
		canonicalString += "Timestamp\n" + ipn.Timestamp + "\n"
	}
	if ipn.TopicArn != "" {
		canonicalString += "TopicArn\n" + ipn.TopicArn + "\n"
	}
	if ipn.Type != "" {
		canonicalString += "Type\n" + ipn.Type + "\n"
	}

	ds, _ := base64.StdEncoding.DecodeString(ipn.Signature)
	h := sha1.New()
	h.Write([]byte(canonicalString))
	digest := h.Sum(nil)
	return rsa.VerifyPKCS1v15(cert.PublicKey.(*rsa.PublicKey), crypto.SHA1, digest, ds) == nil
}
