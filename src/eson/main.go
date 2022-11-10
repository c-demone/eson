package main

import (
	"fmt"

	svc "github.com/ludah65/eson/services"
	"github.com/ludah65/eson/utils"
)

func main() {

	query := svc.VersionQuery{
		Version: "4.1.1", // 2.4.1
		Package: svc.PkgData{
			Name:      "django", // jinja2
			Ecosystem: "PyPI",
		},
	}

	result := query.Post()
	prettyStruct, _ := utils.PrettyStuct(result)
	fmt.Print(prettyStruct)

}
