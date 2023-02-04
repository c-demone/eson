package scan

import (
	"bufio"
	"log"
	"os"
	"strings"

	svc "github.com/ludah65/eson/services"
	"github.com/ludah65/eson/utils"
	"github.com/zcalusic/sysinfo"
)

/*  meant to be a wrapper for all databases or services
    currently includes :
		1) ossindex (Sonatype Open Source Index)
		2) osvindex (Open Source Vulnerability database)
find all python site-packages directories

** directories is <package-name>-<version).dist-info **
These services are created for each language and should return a structure
with all packages and version - list of svc.VersionQuery that can be submitted
*/

const (
	ecosystem   = "PyPI"
	pip_path    = "{{.Home}}/.local/share/lib"
	poetry_path = "{{.Home}}/.local/share/pypoetry"
	pipenv_path = "{{.Home}}/.local/share/virtualenvs"
	conda_path  = "{{.Home}/.conda"
)

/* Arguments to trigger search
type CmdArgs struct {

}*/

type pyPaths struct {
	System string
	Pip    string
	Poetry string
	Pipenv string
	Conda  []string
}

type queryByVersion struct {
	PyVersion      string
	VersionQueries []svc.VersionQuery
}

type Libraries struct {
	Queries []queryByVersion
}

func (p pyPaths) gen_search_paths() pyPaths {
	p.System = dist()

	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}

	data := utils.Fdata{
		"Home": homeDir,
	}

	p.Pip, err = utils.Fstring(pip_path, data)
	if err != nil {
		log.Fatal(err)
	}

	p.Poetry, err = utils.Fstring(poetry_path, data)
	if err != nil {
		log.Fatal(err)
	}

	p.Pipenv, err = utils.Fstring(pipenv_path, data)
	if err != nil {
		log.Fatal(err)
	}

	p.Conda, err = get_conda_envs()
	if err != nil {
		log.Fatal(err)
	}

	return p

}

func dist() string {

	var syspath string
	var si sysinfo.SysInfo

	vendor := &si.OS.Vendor

	if string(*vendor) == "ubuntu" {
		syspath = "/usr/lib"
	}

	if string(*vendor) == "redhat" {
		syspath = "/usr/lib64"
	}

	return syspath
}

func get_conda_envs(home string) ([]string, error) {

	data := utils.Fdata{
		"Home": home,
	}

	cpath, err := utils.Fstring(conda_path, data)
	if err != nil {
		log.Fatal(err)
	}

	// if ~/.conda doesn't exist, return empty slice and nil
	if _, err := os.Stat(cpath); os.IsNotExist(err) {
		var nilSlice []string
		return nilSlice, nil
	}

	file, err := os.Open(cpath + "environments.txt")
	if err != nil {
		log.Fatal(err)
	}

	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, strings.TrimSuffix(scanner.Text(), "\n"))
	}

	return lines, scanner.Err()
}

// Read python package METADATA file for name and version
func ReadMetadata(path string) svc.VersionQuery {
	var pkgdata svc.VersionQuery
	pkgdata.Package.Ecosystem = ecosystem

	file, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}

	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		val := strings.Split(strings.TrimSuffix(scanner.Text(), "\n"), ": ")

		if val[0] == "Name" {
			pkgdata.Package.Name = val[1]
		}

		if val[0] == "Version" {
			pkgdata.Version = val[1]
		}

	}

	return pkgdata
}
