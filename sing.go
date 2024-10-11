package carry

import (
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/zeiss/pkg/b64"
)

// SignProvid provides a Signer for requests.
type SignerProvider interface {
	Sign(req *http.Request) error
}

type noopSigner struct{}

func (s noopSigner) Sign(req *http.Request) error {
	return nil
}

// NewHMacSigner returns a new HMACSigner.
func NewHMacSigner(key string) *HMACSigner {
	return &HMACSigner{
		Key: key,
	}
}

// HMACSigner signs requests with an HMAC signature.
type HMACSigner struct {
	Key string
}

// Sign signs the request with an HMAC signature.
func (s HMACSigner) Sign(req *http.Request) error {
	r, err := req.GetBody()
	if err != nil {
		return err
	}

	b, err := io.ReadAll(r)
	if err != nil {
		return err
	}

	reqURL, err := url.Parse(req.URL.String())
	if err != nil {
		return err
	}

	host := reqURL.Host

	reqURL.Host = ""
	reqURL.Scheme = ""

	date := time.Now().UTC().Format(http.TimeFormat)

	hash, authHeader, err := createAuthHeader(
		req.Method,
		host,
		reqURL.String(),
		date,
		s.Key,
		b,
	)
	if err != nil {
		return err
	}

	req.Header.Set("x-ms-date", date)
	req.Header.Set("x-ms-content-sha256", hash)
	req.Header.Set("Authorization", authHeader)
	req.Header.Set("Content-Type", "application/json")

	return nil
}

func createAuthHeader(method string, host string, path string, date string, secret string, body []byte) (string, string, error) {
	hash, err := b64.ContentHash(body)
	if err != nil {
		return "", "", err
	}

	msg := stringBuilder(
		method,
		"\n",
		path,
		"\n",
		date,
		";",
		host,
		";",
		hash,
	)

	sig, err := b64.Hmac256(msg, secret)
	if err != nil {
		return "", "", err
	}

	authorizationHeader := signedHeaderPrefix + sig

	return hash, authorizationHeader, nil
}

func stringBuilder(strs ...string) string {
	buff := strings.Builder{}
	for _, str := range strs {
		buff.WriteString(str)
	}
	return buff.String()
}
