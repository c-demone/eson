package utils

import (
	"log"

	"github.com/goark/go-cvss/v3/metric"
)

// convert cvss/v3 vector string to severity and score
func CvssScore(vector string) (metric.Severity, float64) {
	bm, err := metric.NewBase().Decode(vector)
	if err != nil {
		log.Fatal(err)
	}

	return bm.Severity(), bm.Score()
}
