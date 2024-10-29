package carry

import (
	"context"
	"encoding/base64"
	"io"
	"net/http"
	"net/url"

	goquery "github.com/google/go-querystring/query"
	"github.com/zeiss/pkg/utilx"
)

const (
	contentType        = "Content-Type"
	jsonContentType    = "application/json"
	formContentType    = "application/x-www-form-urlencoded"
	signedHeaderPrefix = "HMAC-SHA256 SignedHeaders=x-ms-date;host;x-ms-content-sha256&Signature="
)

// Doer executes http requests.  It is implemented by *http.Client.  You can
// wrap *http.Client with layers of Doers to form a stack of client-side
// middleware.
type Doer interface {
	Do(req *http.Request) (*http.Response, error)
}

// Client is an HTTP Request builder and sender.
type Client struct {
	httpClient      Doer
	method          string
	rawURL          string
	header          http.Header
	queryStructs    []interface{}
	bodyProvider    BodyProvider
	responseDecoder ResponseDecoder
	signer          SignerProvider
}

// New returns a new Client with an http DefaultClient.
func New() *Client {
	return &Client{
		httpClient:      http.DefaultClient,
		method:          "GET",
		header:          make(http.Header),
		queryStructs:    make([]interface{}, 0),
		responseDecoder: jsonDecoder{},
		signer:          noopSigner{},
	}
}

// New returns a copy of a Client for creating a new Client with properties
// from a parent Client.
func (s *Client) New() *Client {
	// copy Headers pairs into new Header map
	headerCopy := make(http.Header)
	for k, v := range s.header {
		headerCopy[k] = v
	}

	return &Client{
		httpClient:      s.httpClient,
		method:          s.method,
		rawURL:          s.rawURL,
		header:          headerCopy,
		queryStructs:    append([]interface{}{}, s.queryStructs...),
		bodyProvider:    s.bodyProvider,
		responseDecoder: s.responseDecoder,
		signer:          s.signer,
	}
}

// Client sets the http Client used to do requests. If a nil client is given,
// the http.DefaultClient will be used.
func (s *Client) Client(httpClient *http.Client) *Client {
	if utilx.Empty(httpClient) {
		return s.Doer(http.DefaultClient)
	}

	return s.Doer(httpClient)
}

// Doer sets the custom Doer implementation used to do requests.
// If a nil client is given, the http.DefaultClient will be used.
func (s *Client) Doer(doer Doer) *Client {
	utilx.IfElse(
		utilx.Empty(doer),
		func() { s.httpClient = http.DefaultClient },
		func() { s.httpClient = doer },
	)

	return s
}

// Head sets the Client method to HEAD and sets the given pathURL.
func (s *Client) Head(pathURL string) *Client {
	s.method = "HEAD"
	return s.Path(pathURL)
}

// Get sets the Client method to GET and sets the given pathURL.
func (s *Client) Get(pathURL string) *Client {
	s.method = "GET"
	return s.Path(pathURL)
}

// Post sets the Client method to POST and sets the given pathURL.
func (s *Client) Post(pathURL string) *Client {
	s.method = "POST"
	return s.Path(pathURL)
}

// Put sets the Client method to PUT and sets the given pathURL.
func (s *Client) Put(pathURL string) *Client {
	s.method = "PUT"
	return s.Path(pathURL)
}

// Patch sets the Client method to PATCH and sets the given pathURL.
func (s *Client) Patch(pathURL string) *Client {
	s.method = "PATCH"
	return s.Path(pathURL)
}

// Delete sets the Client method to DELETE and sets the given pathURL.
func (s *Client) Delete(pathURL string) *Client {
	s.method = "DELETE"
	return s.Path(pathURL)
}

// Options sets the Client method to OPTIONS and sets the given pathURL.
func (s *Client) Options(pathURL string) *Client {
	s.method = "OPTIONS"
	return s.Path(pathURL)
}

// Trace sets the Client method to TRACE and sets the given pathURL.
func (s *Client) Trace(pathURL string) *Client {
	s.method = "TRACE"
	return s.Path(pathURL)
}

// Connect sets the Client method to CONNECT and sets the given pathURL.
func (s *Client) Connect(pathURL string) *Client {
	s.method = "CONNECT"
	return s.Path(pathURL)
}

// Add adds the key, value pair in Headers, appending values for existing keys
// to the key's values. Header keys are canonicalized.
func (s *Client) Add(key, value string) *Client {
	s.header.Add(key, value)
	return s
}

// Set sets the key, value pair in Headers, replacing existing values
// associated with key. Header keys are canonicalized.
func (s *Client) Set(key, value string) *Client {
	s.header.Set(key, value)
	return s
}

// SignProvider sets the Client's SignerProvider.
func (s *Client) SignProvider(signer SignerProvider) *Client {
	if utilx.Empty(signer) {
		return s
	}

	s.signer = signer

	return s
}

// SetBasicAuth sets the Authorization header to use HTTP Basic Authentication
func (s *Client) SetBasicAuth(username, password string) *Client {
	return s.Set("Authorization", "Basic "+basicAuth(username, password))
}

func basicAuth(username, password string) string {
	auth := username + ":" + password
	return base64.StdEncoding.EncodeToString([]byte(auth))
}

// Base sets the rawURL of the Client. The rawURL is resolved to an absolute
func (s *Client) Base(rawURL string) *Client {
	s.rawURL = rawURL
	return s
}

// Path extends the rawURL with the given path by resolving the reference to
// an absolute URL. If parsing errors occur, the rawURL is left unmodified.
func (s *Client) Path(path string) *Client {
	baseURL, baseErr := url.Parse(s.rawURL)
	pathURL, pathErr := url.Parse(path)

	if utilx.And(baseErr == nil, pathErr == nil) {
		s.rawURL = baseURL.ResolveReference(pathURL).String()
	}

	return s
}

// QueryStruct appends the queryStruct to the Client's queryStructs. The
func (s *Client) QueryStruct(queryStruct interface{}) *Client {
	if utilx.NotEmpty(queryStruct) {
		s.queryStructs = append(s.queryStructs, queryStruct)
	}

	return s
}

// Body sets the Client's body. The body is used on new requests (see Request()).
func (s *Client) Body(body io.Reader) *Client {
	if utilx.Empty(body) {
		return s
	}

	return s.BodyProvider(bodyProvider{body: body})
}

// BodyProvider sets the Client's body provider.
func (s *Client) BodyProvider(body BodyProvider) *Client {
	if utilx.Empty(body) {
		return s
	}

	s.bodyProvider = body

	ct := body.ContentType()
	if utilx.NotEmpty(ct) {
		s.Set(contentType, ct)
	}

	return s
}

// BodyJSON sets the Client's bodyJSON. The value pointed to by the bodyJSON
func (s *Client) BodyJSON(bodyJSON interface{}) *Client {
	if utilx.Empty(bodyJSON) {
		return s
	}

	return s.BodyProvider(jsonBodyProvider{payload: bodyJSON})
}

// BodyForm sets the Client's bodyForm. The value pointed to by the bodyForm
func (s *Client) BodyForm(bodyForm interface{}) *Client {
	if utilx.Empty(bodyForm) {
		return s
	}

	return s.BodyProvider(formBodyProvider{payload: bodyForm})
}

// Request returns a new http.Request created from the Client's properties.
func (s *Client) Request(ctx context.Context) (*http.Request, error) {
	reqURL, err := url.Parse(s.rawURL)
	if err != nil {
		return nil, err
	}

	err = addQueryStructs(reqURL, s.queryStructs)
	if err != nil {
		return nil, err
	}

	var body io.Reader
	if s.bodyProvider != nil {
		body, err = s.bodyProvider.Body()
		if err != nil {
			return nil, err
		}
	}
	req, err := http.NewRequestWithContext(ctx, s.method, reqURL.String(), body)
	if err != nil {
		return nil, err
	}

	addHeaders(req, s.header)

	if s.signer != nil {
		err = s.signer.Sign(req)
		if err != nil {
			return nil, err
		}
	}

	return req, err
}

func addQueryStructs(reqURL *url.URL, queryStructs []interface{}) error {
	urlValues, err := url.ParseQuery(reqURL.RawQuery)
	if err != nil {
		return err
	}

	for _, queryStruct := range queryStructs {
		queryValues, err := goquery.Values(queryStruct)
		if err != nil {
			return err
		}

		for key, values := range queryValues {
			for _, value := range values {
				urlValues.Add(key, value)
			}
		}
	}

	reqURL.RawQuery = urlValues.Encode()

	return nil
}

func addHeaders(req *http.Request, header http.Header) {
	for key, values := range header {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}
}

// ResponseDecoder sets the Client's response decoder.
func (s *Client) ResponseDecoder(decoder ResponseDecoder) *Client {
	if utilx.Empty(decoder) {
		return s
	}

	s.responseDecoder = decoder

	return s
}

// ReceiveSuccess creates a new HTTP request and returns the response. Success
func (s *Client) ReceiveSuccess(ctx context.Context, successV interface{}) (*http.Response, error) {
	return s.Receive(ctx, successV, nil)
}

// Receive creates a new HTTP request and returns the response. Success responses
func (s *Client) Receive(ctx context.Context, successV, failureV interface{}) (*http.Response, error) {
	req, err := s.Request(ctx)
	if err != nil {
		return nil, err
	}
	return s.Do(req, successV, failureV)
}

// Do sends an HTTP request and returns an HTTP response, following policy
func (s *Client) Do(req *http.Request, successV, failureV interface{}) (*http.Response, error) {
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return resp, err
	}
	defer resp.Body.Close()
	defer io.Copy(io.Discard, resp.Body)

	if resp.StatusCode == http.StatusNoContent || resp.ContentLength == 0 {
		return resp, nil
	}

	if successV != nil || failureV != nil {
		err = decodeResponse(resp, s.responseDecoder, successV, failureV)
	}
	return resp, err
}

func decodeResponse(resp *http.Response, decoder ResponseDecoder, successV, failureV interface{}) error {
	if code := resp.StatusCode; 200 <= code && code <= 299 {
		if successV != nil {
			return decoder.Decode(resp, successV)
		}
	} else {
		if failureV != nil {
			return decoder.Decode(resp, failureV)
		}
	}

	return nil
}
