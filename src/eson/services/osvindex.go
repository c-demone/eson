package services

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
)

const (
	osvUrl = "https://api.osv.dev/v1/query"
	header = "application/json"
)

type response map[string]interface{}

type Query interface {
	Post() OsvResponse
}

type PkgData struct {
	Name      string `json:"name"`
	Ecosystem string `json:"ecosystem"`
}
type VersionQuery struct {
	Version string  `json:"version"`
	Package PkgData `json:"package"`
}

type CommitQuery struct {
	Commit string `json:"commit"`
}

type severity struct {
	Type  string `json:"type"`
	Score string `json:"score"`
}

type pkg struct {
	Ecosystem string `json:"ecosystem"`
	Name      string `json:"name"`
	Purl      string `json:"purl"`
}

type events struct {
	Introduced   string `json:"introduced"`
	Fixed        string `json:"fixed"`
	LastAffected string `json:"last_affected"`
	Limit        string `json:"limit"`
}

type ranges struct {
	Type       string   `json:"type"`
	Repo       string   `json:"repo"`
	Events     []events `json:"events"`
	DbSpecific response `json:"database_sepcific"`
}

type affected struct {
	Package     pkg      `json:"package"`
	Ranges      ranges   `json:"ranges"`
	Versions    []string `json:"versions"`
	EcoSpecific response `json:"ecosystem_specific"`
	DbSpecific  response `json:"database_specific"`
}

type references struct {
	Type string `json:"references"`
	Url  string `json:"url"`
}

type credits struct {
	Name    string   `json:"name"`
	Contact []string `json:"contact"`
}

type OsvReport struct {
	SchemaVersion string       `json:"schema_version"`
	Id            string       `json:"id"`
	Modified      string       `json:"modified"`
	Published     string       `json:"published`
	Withdrawn     string       `json:"withdrawn"`
	Aliases       string       `json:"aliases"`
	Related       string       `json:"related"`
	Summary       string       `json:"summary"`
	Details       string       `json:"details"`
	Serverity     []severity   `json:"severity"`
	Affected      []affected   `json:"affected"`
	References    []references `json:"references"`
	Credits       []credits    `json:"credits"`
	DbSpecific    response     `json:"database_specific"`
}

type OsvResponse struct {
	Vulns []OsvReport `json:"vulns"`
}

func (q VersionQuery) Post() OsvResponse {

	// Marshal data struct to json
	data, err := json.Marshal(q)

	if err != nil {
		log.Fatal(err)
	}

	// Build http request
	req, err := http.NewRequest("POST", osvUrl, bytes.NewBuffer(data))
	if err != nil {
		log.Fatal(err)
	}
	// Set basic request header
	req.Header.Set("Content-Type", header)

	// Initiate http client
	client := &http.Client{}

	// Send post request and store response
	resp, err := client.Do(req)

	if err != nil {
		log.Fatal(err)
	}

	defer resp.Body.Close()

	// Decode response body onto target struct
	var result OsvResponse
	json.NewDecoder(resp.Body).Decode(&result)

	return result
}

func (q CommitQuery) Post() OsvResponse {

	// Marshal data struct to json
	data, err := json.Marshal(q)
	if err != nil {
		log.Fatal(err)
	}

	// Build http request with basic header
	req, err := http.NewRequest("POST", osvUrl, bytes.NewBuffer(data))

	if err != nil {
		log.Fatal(err)
	}
	req.Header.Set("Content-Type", header)

	// Initiate http client
	client := &http.Client{}

	// Send post request and store response
	resp, err := client.Do(req)

	if err != nil {
		log.Fatal(err)
	}

	defer resp.Body.Close()

	// Decode response body onto target struct
	var result OsvResponse
	json.NewDecoder(resp.Body).Decode(&result)

	return result
}

func extractAndAssignResp(req *http.Request, v interface{}) {
	// call like
	// var commitResp CommitResp
	// extractAndAssignInfo(req, &commitResp)
	decoder := json.NewDecoder(req.Body)
	err := decoder.Decode(v)
	if err != nil {
		log.Fatal(err)
	}
}
