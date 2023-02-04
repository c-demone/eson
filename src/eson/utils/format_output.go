package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"text/template"
)

// https://github.com/jedib0t/go-pretty/tree/main/table
func PrettyStuct(data interface{}) (string, error) {
	val, err := json.MarshalIndent(data, "", "   ")
	if err != nil {
		return "", err
	}

	return string(val), nil
}

type Fdata map[string]interface{}

func Fstring(format string, data Fdata) (string, error) {
	t, err := template.New("fstring").Parse(format)
	if err != nil {
		return "", fmt.Errorf("error creating template: %v", err)
	}
	output := new(bytes.Buffer)
	if err := t.Execute(output, data); err != nil {
		return "", fmt.Errorf("error executing template: %v", err)
	}
	return output.String(), nil
}
