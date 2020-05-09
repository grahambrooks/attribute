package neo

import (
	"encoding/json"
	"fmt"
	"github.com/grahambrooks/attribute/scan/tag"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type TransactionResponse struct {
	Commit string `json:"commit"`
}

type TransactionalClient struct {
	*NeoClient
	TransactionUrl       string
	CommitTransactionUrl string
	Requests             TransactionRequest
}

type TransactionRequest struct {
	Statements []Statement `json:"statements"`
}

func (tc *TransactionalClient) Request(s ...Statement) {
	tc.Requests.Statements = append(tc.Requests.Statements, s...)
}

func (tc *TransactionalClient) CommitTransaction() error {
	if len(tc.Requests.Statements) > 0 {
		reader, err := tc.encode(&tc.Requests)
		if err != nil {
			log.Printf("Encoding failed %v", err)
		}

		response, err := tc.httpRequest(http.MethodPost, tc.CommitTransactionUrl, reader)

		if err != nil {
			log.Printf("NewContributor failed %v", err)
		}

		if response.StatusCode != http.StatusOK && response.StatusCode != http.StatusCreated {
			decoder := json.NewDecoder(response.Body)

			var neo NeoResultResponse
			_ = decoder.Decode(&neo)

			log.Printf("Decoded response %#v %d", neo, response.StatusCode)
			if len(neo.Results) > 0 {
				if len(neo.Results[0].Errors) > 0 {
					if neo.Results[0].Errors[0].Code == "Neo.ClientError.Transaction.TransactionAccessedConcurrently" {
						log.Printf("Transaction failed - waiting %d", response.StatusCode)
						return fmt.Errorf("concurrency failure comitting transaction %d", response.StatusCode)
					}
				}
			}
			return fmt.Errorf("neo4j Transaction failure %d", response.StatusCode)
		}
	}

	return nil
}

func makeTagParams(tags []tag.Tag) string {
	var b strings.Builder

	for _, t := range tags {
		_, _ = fmt.Fprintf(&b, ", %s: $%s", t.Key, t.Key)
	}
	return b.String()
}

func makeStatement(format string, tags []tag.Tag) string {
	return fmt.Sprintf(format, makeTagParams(tags))
}

func (tc *TransactionalClient) NewRepository(request NewRepositoryRequest) {
	r := TransactionRequest{}

	statement := Statement{
		Statement:  makeStatement(`MERGE (n:Repository { name: $name, origin: $origin, commits: $commits%s }) RETURN n`, request.Tags),
		Parameters: make(map[string]string),
	}
	statement.Parameters["name"] = request.Name
	statement.Parameters["origin"] = request.Origin
	statement.Parameters["commits"] = strconv.Itoa(request.CommitCount)
	for _, t := range request.Tags {
		statement.Parameters[t.Key] = t.Value
	}
	r.Statements = append(r.Statements, statement)

	tc.Request(statement)
}

func (tc *TransactionalClient) NewContributor(request NewContributorRequest) {
	contributor := Statement{
		Statement:  makeStatement(`MERGE (n:Contributor {name: $name, email: $email%s })
ON CREATE SET n.commits = $commits
ON MATCH SET n.commits = n.commits + $commits
RETURN n`, request.Tags),
		Parameters: make(map[string]string),
	}
	contributor.Parameters["name"] = request.Name
	contributor.Parameters["email"] = request.Email
	contributor.Parameters["commits"] = strconv.Itoa(request.CommitCount)
	for _, t := range request.Tags {
		contributor.Parameters[t.Key] = t.Value
	}

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
