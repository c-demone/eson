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
	conda_path  = "{{.Home}}/.conda"
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
	var res DmQuery

	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}
	p.home = homeDir

	p.data = utils.Fdata{
		"Home": homeDir,
	}

	p = p.gen_search_paths(args)

	if args.All == true {
		res, _ = p.allScan()
		// handle errors
		return res
	}

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

func (p pyPaths) gen_search_paths(args PyFsArgs) pyPaths {
	var err error

	if slices.Contains(args.DepManager, "sys") == true || args.All == true {
		p.System = dist()
	}

	if slices.Contains(args.DepManager, "pip") == true || args.All == true {
		p.Pip, err = utils.Fstring(pip_path, p.data)
		if err != nil {
			log.Fatal(err)
		}
	}

	if slices.Contains(args.DepManager, "poetry") == true || args.All == true {
		p.Poetry, err = utils.Fstring(poetry_path, p.data)
		if err != nil {
			log.Fatal(err)
		}
	}

	if slices.Contains(args.DepManager, "pipenv") == true ||
		slices.Contains(dms, "virtualenv") == true ||
		args.All == true {
		p.Pipenv, err = utils.Fstring(pipenv_path, p.data)
		if err != nil {
			log.Fatal(err)
		}
	}

	if slices.Contains(args.DepManager, "conda") == true || args.All == true {
		p.Conda, err = get_conda_envs(p.data)
		if err != nil {
			log.Fatal(err)
		}
	}
	return p
}

func dist() string {

	var syspath string
	var si sysinfo.SysInfo

	si.GetSysInfo()
	vendor := &si.OS.Vendor

	if string(*vendor) == "ubuntu" {
		syspath = "/usr/lib"
	}

	if string(*vendor) == "redhat" {
		syspath = "/usr/lib64"
	}
	return syspath
}

func get_conda_envs(data utils.Fdata) ([]string, error) {

	cpath, err := utils.Fstring(conda_path, data)
	if err != nil {
		log.Fatal(err)
	}

	// if ~/.conda doesn't exist, return empty slice and nil
	if _, err := os.Stat(cpath); os.IsNotExist(err) {
		var nilSlice []string
		return nilSlice, nil
	}

	file, err := os.Open(cpath + "/environments.txt")
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
				//pyversion := exp.FindAllString(path, -1)
				if _, err := os.Stat(path + "/METADATA"); os.IsNotExist(err) {
					return nil
				} else {
					if strings.Contains(path, "pkgs") == false {
						pyversion := exp.FindAllString(path, -1)
						m[pyversion[0]] = append(m[pyversion[0]], ReadMetadata(path+"/METADATA"))
					}
				}
				if dm != "sys" && dm != "pip" {
					slice := strings.Split(path, "/")
					idx := slices.IndexFunc(slice, func(s string) bool { return s == "lib" })

					if dm == "conda" {
						cidx := slices.IndexFunc(slice, func(s string) bool { return s == ".conda" })
						if cidx < 0 {
							dmMap[strings.Join(slice[:idx], "/")] = m
						} else {
							dmMap[slice[idx-1]] = m
						}
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
			merged[k1] = QueryMap{}
			for k2, v2 := range v1 {
				merged[k1][k2] = append(merged[k1][k2], v2...)
			}
		}
	}
	return merged
}

func (p pyPaths) systemScan(res DmQuery) (DmQuery, error) {
	if _, err := os.Stat(p.System); os.IsNotExist(err) {
		return res, err
	} else {
		qmap, err := FsScan(p.System, "sys")
		res.Sys = qmap["sys"]
		return res, err
	}
}

func (p pyPaths) pipScan(res DmQuery) (DmQuery, error) {
	if _, err := os.Stat(p.Pip); os.IsNotExist(err) {
		return res, err
	} else {
		qmap, err := FsScan(p.Pip, "pip")
		res.Pip = qmap["pip"]
		return res, err
	}
}

func (p pyPaths) poetryScan(res DmQuery) (DmQuery, error) {
	if _, err := os.Stat(p.Poetry); os.IsNotExist(err) {
		return res, err
	} else {
		qmap, err := FsScan(p.Poetry, "poetry")
		res.Poetry = qmap
		return res, err
	}
}

// works for both pipenv and virtualenv arguments
func (p pyPaths) pipenvScan(res DmQuery) (DmQuery, error) {
	if _, err := os.Stat(p.Pipenv); os.IsNotExist(err) {
		return res, err
	} else {
		qmap, err := FsScan(p.Pipenv, "pipenv")
		res.Pipenv = qmap
		return res, err
	}
}

func (p pyPaths) condaScan(res DmQuery) (DmQuery, []error) {
	var errs []error
	if _, err := os.Stat(p.Conda[0]); os.IsNotExist(err) {
		errs = append(errs, err)
		return res, errs
	} else {
		var scanResults []DmQueryMap
		for _, path := range p.Conda {
			qmap, err := FsScan(path, "conda")
			scanResults = append(scanResults, qmap)
			errs = append(errs, err)
		}
		res.Conda = MergeResSlices(scanResults)
		return res, errs
	}
}

func (p pyPaths) allScan() (DmQuery, []error) {
	var errors []error
	var errs []error
	var res DmQuery
	var err error

	res, err = p.systemScan(res)
	if err != nil {
		errs = append(errs, err)
	}

	res, err = p.pipScan(res)
	if err != nil {
		errs = append(errs, err)
	}

	res, err = p.poetryScan(res)
	if err != nil {
		errs = append(errs, err)
	}

	res, err = p.pipenvScan(res)
	if err != nil {
		errs = append(errs, err)
	}

	res, errors = p.condaScan(res)
	if len(errors) > 0 {
		for _, e := range errors {
			errs = append(errs, e)
		}
	}
	return res, errors
}

// run scan on all queries and tabulate data for output
//func (res DmQuery) Scan(outformat string) {
// outformat = table || csv:/path/to/write || json:/path/to/write
// for csv and json if no ":" in outformat, then write to current
// working directory
//}
