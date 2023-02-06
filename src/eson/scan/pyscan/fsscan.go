package pyscan

import (
	"bufio"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"golang.org/x/exp/slices"

	ds "github.com/bmatcuk/doublestar/v4"
	svc "github.com/ludah65/eson/services"
	"github.com/ludah65/eson/utils"
	"github.com/zcalusic/sysinfo"
)

/*  meant to be a wrapper for all databases or services
    currently includes :
		1) ossindex (Sonatype Open Source Index)
		2) osvindex (Open Source Vulnerability database)
    find all python site-packages directories
*/

const (
	ecosystem   = "PyPI"
	pip_path    = "{{.Home}}/.local/lib"
	poetry_path = "{{.Home}}/.local/share/pypoetry"
	pipenv_path = "{{.Home}}/.local/share/virtualenvs"
	conda_path  = "{{.Home}/.conda"
)

var dms = []string{"sys", "pip", "poetry", "pipenv", "virtualenv", "conda"}

type pyPaths struct {
	System string
	Pip    string
	Poetry string
	Pipenv string
	Conda  []string
	home   string
	data   utils.Fdata
}

type pkgMeta struct {
	PyVersion string
	MetaPaths []string
}
type pkgPaths struct {
	System []pkgMeta
	Pip    []pkgMeta
	Poetry []pkgMeta
	Pipenv []pkgMeta
	Conda  []pkgMeta
}

type QueryMap map[string][]svc.VersionQuery
type DmQueryMap map[string]QueryMap

type DmQuery struct {
	Sys    QueryMap
	Pip    QueryMap
	Poetry DmQueryMap
	Pipenv DmQueryMap
	Conda  DmQueryMap
}

type PyFsArgs struct {
	DepManager []string // pipenv scan & virtualenv scan give the same thing
	All        bool
}

// initialize pyPaths struct to access scan methods
// and return scan results with structured queries
func (args PyFsArgs) NewFsScan() DmQuery {
	var p pyPaths

	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}
	p.home = homeDir

	p.data = utils.Fdata{
		"Home": homeDir,
	}

	if args.All == true {
		p.gen_search_paths()
		res, _ := p.allScan()
		// handle errors
		return res
	}

	var res DmQuery
	if slices.Contains(args.DepManager, "sys") == true {
		res, _ = p.systemScan(res)

	}

	if slices.Contains(args.DepManager, "pip") == true {
		res, _ = p.pipScan(res)
	}

	if slices.Contains(args.DepManager, "poetry") == true {
		res, _ = p.poetryScan(res)
	}

	if slices.Contains(args.DepManager, "virtualenv") == true ||
		slices.Contains(args.DepManager, "pipenv") == true {
		res, _ = p.pipenvScan(res)
	}

	if slices.Contains(args.DepManager, "conda") == true {
		res, _ = p.condaScan(res)
	}
	return res
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

func FsScan(root string, dm string) (DmQueryMap, error) {

	m := make(QueryMap)
	dmMap := make(DmQueryMap)
	pattern := "**/*.dist-info"
	exp := regexp.MustCompile(`python\d.\d`)

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() {
			matched, err := ds.PathMatch(pattern, path)
			if err != nil {
				log.Fatal(err)
			}
			if matched == true {
				pyversion := exp.FindAllString(path, -1)
				m[pyversion[0]] = append(m[pyversion[0]], ReadMetadata(path+"/METADATA"))

				if dm != "sys" && dm != "pip" {
					slice := strings.Split(path, "/")
					idx := slices.IndexFunc(slice, func(s string) bool { return s == "lib" })

					if dm == "conda" {
						dmMap[slice[idx-1]] = m
					} else {
						raw_venv := strings.Split(slice[idx-1], "-")
						venv_name := strings.Join(raw_venv[:len(raw_venv)-1], "-")
						dmMap[venv_name] = m
					}
				} else {
					dmMap[dm] = m
				}
			}
			return nil
		}
		return nil
	})
	return dmMap, err
}

func MergeResSlices(mapslice []DmQueryMap) DmQueryMap {
	merged := make(DmQueryMap)

	for i := range mapslice {
		for k1, v1 := range mapslice[i] {
			for k2, v2 := range v1 {
				merged[k1][k2] = append(merged[k1][k2], v2...)
			}
		}
	}
	return merged
}

func (p pyPaths) systemScan(res DmQuery) (DmQuery, error) {
	qmap, err := FsScan(p.System, "sys")
	res.Sys = qmap["sys"]
	return res, err
}

func (p pyPaths) pipScan(res DmQuery) (DmQuery, error) {
	qmap, err := FsScan(p.Pip, "pip")
	res.Pip = qmap["pip"]
	return res, err

}

func (p pyPaths) poetryScan(res DmQuery) (DmQuery, error) {
	qmap, err := FsScan(p.Pipenv, "poetry")
	res.Poetry = qmap
	return res, err

}

// pipenv scan will scan all virtualenvs
// same method can be used for both arguments
// and should only be run once if both pipenv and
// virtualenv are specified
func (p pyPaths) pipenvScan(res DmQuery) (DmQuery, error) {
	qmap, err := FsScan(p.Pipenv, "pipenv")
	res.Pipenv = qmap
	return res, err

}

// conda can have multiple paths to search - need to adjust
func (p pyPaths) condaScan(res DmQuery) (DmQuery, []error) {
	var scanResults []DmQueryMap
	var errs []error
	for _, path := range p.Conda {
		qmap, err := FsScan(path, "conda")
		scanResults = append(scanResults, qmap)
		errs = append(errs, err)
	}
	res.Conda = MergeResSlices(scanResults)
	return res, errs
}

func (p pyPaths) allScan() (DmQuery, []error) {
	var errors []error
	var errs []error
	var res DmQuery
	var err error

	res, err = systemScan(DmQuery)
	if err != nil {
		errs = append(errs, err)
	}

	res, err = pipScan(DmQuery)
	if err != nil {
		errs = append(errs, err)
	}

	res, err = poetryScan(DmQuery)
	if err != nil {
		errs = append(errs, err)
	}

	res, err = virtualenvScan(DmQuery)
	if err != nil {
		errs = append(errs, err)
	}

	res, errors = condaScan(DmQuery)
	if len(errors) > 0 {
		for _, e := range errors {
			errs = append(errs, e)
		}
	}

	return res, errors

}
