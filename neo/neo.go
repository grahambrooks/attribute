package neo

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
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

type TransactionResponse struct {
	Commit string `json:"commit"`
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

type TransactionalClient struct {
	*NeoClient
	TransactionUrl       string
	CommitTransactionUrl string
	Requests             TransactionRequest
}

func (tc *TransactionalClient) Request(s ...Statement) {
	tc.Requests.Statements = append(tc.Requests.Statements, s...)

	if len(tc.Requests.Statements) > 100 {
		tc.sendRequests()
	}
}

func (tc *TransactionalClient) sendRequests() {
	if len(tc.Requests.Statements) > 0 {
		reader, err := tc.encode(&tc.Requests)
		if err != nil {
			log.Printf("Encoding failed %v", err)
		}

		response, err := tc.httpRequest(http.MethodPost, tc.TransactionUrl, reader)

		if err != nil {
			log.Printf("NewContributor failed %v", err)
		}

		if response.StatusCode != http.StatusOK && response.StatusCode != http.StatusCreated {
			decoder := json.NewDecoder(response.Body)

			var neo NeoResultResponse
			decoder.Decode(&neo)

			log.Printf("Decoded response %#v %d", neo, response.StatusCode)
			if len(neo.Results) > 0 {
				if len(neo.Results[0].Errors) > 0 {
					if neo.Results[0].Errors[0].Code == "Neo.ClientError.Transaction.TransactionAccessedConcurrently" {
						log.Printf("Transaction failed - waiting %d", response.StatusCode)
						time.Sleep(500 * time.Millisecond)
						return
					}
				}
			}
		}
	}
	tc.Requests = TransactionRequest{}
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

type TransactionRequest struct {
	Statements []Statement `json:"statements"`
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

func (tc *TransactionalClient) CommitTransaction() error {
	tc.sendRequests()
	time.Sleep(1000 * time.Millisecond)
	response, err := tc.httpRequest(http.MethodPost, tc.CommitTransactionUrl, strings.NewReader(`{ "statements": [] }`))

	_, _ = io.Copy(os.Stdout, response.Body)
	return err
}

func (tc *TransactionalClient) NewRepository(request NewRepositoryRequest) {
	r := TransactionRequest{}

	statement := Statement{
		Statement:  `MERGE (n:Repository { name: $name, origin: $origin }) RETURN n`,
		Parameters: make(map[string]string),
	}
	statement.Parameters["name"] = request.Name
	statement.Parameters["origin"] = request.Origin
	r.Statements = append(r.Statements, statement)

	tc.Request(statement)
}

type NewContributorRequest struct {
	Origin  string
	Name    string
	Email   string
	When    time.Time
	Message string
}

func (tc *TransactionalClient) NewContributor(request NewContributorRequest) {
	contributor := Statement{
		Statement: fmt.Sprintf(`MERGE (n:Contributor {name: '%s', email: '%s'}) RETURN n`, request.Name, request.Email),
	}

	//	eachCommitQuery := `MATCH (a:Contributor),(b:Repository)
	//WHERE a.email = $email AND b.name = $name
	//CREATE (a)-[r:Contributes { name: a.name + '<->' + b.name, when: $when, message: $message }]->(b)
	//RETURN type(r), r.name`
	simpleContributorQuery := `MATCH (a:Contributor),(b:Repository)
WHERE a.email = $email AND b.name = $name
MERGE (a)-[r:Contributes { name: a.name + '<->' + b.name }]->(b)
RETURN type(r), r.name`

	statement := Statement{
		Statement:  simpleContributorQuery,
		Parameters: make(map[string]string),
	}
	statement.Parameters["email"] = request.Email
	statement.Parameters["name"] = request.Origin
	statement.Parameters["when"] = request.When.Format(time.RFC1123Z)
	statement.Parameters["message"] = request.Message

	tc.Request(contributor, statement)
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
