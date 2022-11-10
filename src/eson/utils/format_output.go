package utils

import (
	"encoding/json"
)

// https://github.com/jedib0t/go-pretty/tree/main/table

func PrettyStuct(data interface{}) (string, error) {
	val, err := json.MarshalIndent(data, "", "   ")
	if err != nil {
		return "", err
	}

	return string(val), nil
}
