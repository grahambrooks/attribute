package neo

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

type NeoClient struct {
	Client   *http.Client
	Url      string
	Username string
	Password string
}

func (c *NeoClient) httpRequest(method string, url string, body io.Reader) (*http.Response, error) {
	log.Printf("Request %s", url)
	req, _ := http.NewRequest(method, url, body)

	req.SetBasicAuth(c.Username, c.Password)
	req.Header.Set("Accept", "application/json; charset=UTF-8")
	req.Header.Set("Content-Type", "application/json")

	return c.Client.Do(req)

}

func (c *NeoClient) BeginTransaction(_ string) (*TransactionalClient, error) {
	request, err := c.httpRequest(http.MethodPost, c.Url+"/db/neo4j/tx", strings.NewReader(`{"statements": []}`))

	if err != nil {
		return nil, err
	}

	var transaction TransactionResponse

	err = c.decode(request.Body, &transaction)
	if err != nil {
		return nil, err
	}

	location, err := request.Location()
	if err != nil {
		return nil, err
	}
	return &TransactionalClient{
		NeoClient:            c,
		TransactionUrl:       location.String(),
		CommitTransactionUrl: transaction.Commit,
	}, nil
}

type ClientOptions struct {
	Host     string
	Username string
	Password string
}

func NewNeoClient(options ClientOptions) NeoClient {
	return NeoClient{
		Client:   &http.Client{},
		Url:      options.Host,
		Username: options.Username,
		Password: options.Password,
	}
}

type NeoResultResponse struct {
	Results []NeoResult `json:"results"`
}

type NeoError struct {
	Code string
}

type NeoResult struct {
	Errors []NeoError
}

type Statement struct {
	Statement  string            `json:"statement"`
	Parameters map[string]string `json:"parameters,omitempty"`
}

func (c *NeoClient) Transaction(transaction func(client *TransactionalClient) error) error {
	t, err := c.BeginTransaction("neo4j")
	if err == nil {
		if transaction(t) == nil {
			return t.CommitTransaction()
		}
	}
	return err
}

type NewRepositoryRequest struct {
	Name   string `json:"name"`
	Origin string `json:"origin"`
}

type NewContributorRequest struct {
	Origin  string
	Name    string
	Email   string
	When    time.Time
	Message string
}

func (c *NeoClient) decode(readCloser io.ReadCloser, transaction *TransactionResponse) error {
	decoder := json.NewDecoder(readCloser)
	return decoder.Decode(transaction)
}

func (c *NeoClient) encode(transaction *TransactionRequest) (io.Reader, error) {
	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	err := encoder.Encode(transaction)
	return bytes.NewReader(buffer.Bytes()), err
}
