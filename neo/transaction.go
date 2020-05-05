package neo

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
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
			decoder.Decode(&neo)

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
