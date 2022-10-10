package services

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/nscuro/ossindex-client"
)

type ossReports map[string]ComponentReport

type OssIndex struct {
	Username    string
	Token       string
	Coordinates []string
	Results     ossReports
}

// ossindex.ComponentReport here for reference can be removed
// and called from ossindex-client package
type ComponentReport struct {
	Coordinates     string                   `json:"coordinates"`
	Description     string                   `json:"description"`
	Reference       string                   `json:"reference"`
	Vulnerabilities []ossindex.Vulnerability `json:"vulnerabilities"`
}

func (oi OssIndex) getReports() {
	var (
		client  *ossindex.Client
		err     error
		results ossReports
	)

	if oi.Username != "" && oi.Token != "" {
		client, err = ossindex.NewClient(ossindex.WithAuthentication(oi.Username, oi.token))
	} else {
		client, err = ossindex.NewClient()
	}

	if err != nil {
		log.Fatalf("failed to initialize client: %v", err)
	}

	reports, err := client.GetComponentReports(context.Background(), oi.Coordinates)
	if err != nil {
		log.Fatalf("failed to get component reports: %v", err)
	}

	for _, report := range reports {

		// split report.Coordinates into package name-version
		// type:namespace/name@version?qualifiers#subpath
		key := strings.Split(strings.Split(report.Coordinates, "?")[0], ":")[1]
		key = strings.Replace(strings.Split(key, "/")[1], "@", "-", 1)

		if len(report.Vulnerabilities) != 0 {
			results[key] = report
		}
	}

	oi.Results = results
}

func (oi OssIndex) showResults() {

	if len(oi.Results) == 0 {
		oi.getReports()
	}
	for _, report := range oi.Results {

		fmt.Printf(" > %s\n", report.Coordinates)

		for _, vulnerability := range report.Vulnerabilities {
			fmt.Printf("   - %s\n", vulnerability.Title)
		}
	}
}

/*
func (oi ossIndex) tabulateResults() {

}*/
