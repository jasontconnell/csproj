package csproj

import (
    "errors"
    "fmt"
    "io/ioutil"
    "os"
    "path/filepath"
    "regexp"
    "strconv"
    "strings"
)

var nsreg *regexp.Regexp = regexp.MustCompile(`(?is)<rootnamespace>(.*?)</rootnamespace>`)
var frmwkreg *regexp.Regexp = regexp.MustCompile(`(?is)<targetframeworkversion>v(.*?)</targetframeworkversion>`)
var prefreg *regexp.Regexp = regexp.MustCompile(`(?is)<reference include="([a-z0-9\., =\-]*?)">(.*?)</reference>`)
var arefreg *regexp.Regexp = regexp.MustCompile(`(?is)<reference include="([a-z0-9\.]*?)" />`)
var pkgrefreg *regexp.Regexp = regexp.MustCompile(`(?is)<packagereference include="([a-z0-9\.]*?)">.*?<version>(.*?)</version>.*?</packagereference>`)
var projrefreg *regexp.Regexp = regexp.MustCompile(`(?is)<projectreference include="(.*?)">.*<name>(.*?)</name>.*</projectreference>`)
var hpreg *regexp.Regexp = regexp.MustCompile("(?is).*?<hintpath>(.*)</hintpath>.*?")
var pkgnamereg *regexp.Regexp = regexp.MustCompile(`(?is)..\\packages\\(.*?)\\.*`)
var privreg *regexp.Regexp = regexp.MustCompile("(?is).*?<private>(.*)</private>.*?")
var creg *regexp.Regexp = regexp.MustCompile(`(?is)<(none|compile|content) include="([a-z0-9\.\-\\ _]*?)" ?/>`)
var c2reg *regexp.Regexp = regexp.MustCompile(`(?is)<(none|compile|content) include="([a-z0-9\.\-\\ _]*?)">.*?</(none|compile|content)>`) // do we care about subtype?
var pkgreg *regexp.Regexp = regexp.MustCompile(`(?is).*?id="(.*?)" version="(.*?)" targetFramework="(.*?)"( developmentDependency="(.*?)")?.*?`)

func GetRootNamespace(path string) (string, error) {
    contents, err := ioutil.ReadFile(path)
    if err != nil {
        return "", err
    }

    rootns := loadRootNS(contents)
    return rootns, nil
}

func Load(path string) (Project, error) {
    contents, err := ioutil.ReadFile(path)
    if err != nil {
        return Project{}, err
    }

    targetFramework := loadTargetFramework(contents)
    references, projrefs := loadReferences(contents)
    files := loadFiles(contents)
    pkgsconfig := loadPackages(path)
    mapPackageToRefVersion(&references, pkgsconfig)

    dir, file := filepath.Split(path)
    rootns := loadRootNS(contents)

    _, key := filepath.Split(filepath.Dir(filepath.Dir(dir)))

    proj := Project{Key: key, RootNamespace: rootns, FrameworkVersion: targetFramework, Filename: file, FullPath: path, References: references, ProjectRefs: projrefs, Files: files, Packages: pkgsconfig}

    return proj, nil
}

func LoadAll(dir string) ([]Project, error) {
    projects := []Project{}

    // Find goes up one directory, this will typically already be in the directory whose children we want to search.
    // so just give a fake folder, it'll do filepath.Dir on it.
    paths := Find(filepath.Join(dir, "x"), "")

    for _, csp := range paths {
        p, err := Load(csp)
        if err != nil {
            return nil, err
        }

        projects = append(projects, p)
    }

    projects, err := MapProjectReferences(projects)
    if err != nil {
        return nil, err
    }

    return projects, nil
}

func MapProjectReferences(projects []Project) ([]Project, error) {
    list := []Project{}
    keymap := make(map[string]string)

    for _, p := range projects {
        keymap[p.RootNamespace] = p.Key
    }

    for _, p := range projects {
        for i := 0; i < len(p.ProjectRefs); i++ {
            if k, ok := keymap[p.ProjectRefs[i].Name]; ok {
                p.ProjectRefs[i].Key = k
            } else {
                return nil, errors.New(fmt.Sprintf("Project with namespace '%v' referenced but a repository was not found", p.ProjectRefs[i].Name))
            }
        }

        list = append(list, p)
    }

    return list, nil
}

func mapPackageToRefVersion(references *[]Reference, packages []Package) {
    pmap := make(map[string]Package)
    for _, pkg := range packages {
        pmap[pkg.Id] = pkg
    }

    refloop := *references

    for i := 0; i < len(refloop); i++ {
        if refloop[i].Version == "" {
            continue
        } // has a version already
        if refloop[i].Hint == "" {
            continue
        } // doesn't need it

        if pkg, ok := pmap[refloop[i].Name]; ok {
            (*references)[i].Version = pkg.Version
        }
    }
}

func loadPackages(path string) []Package {
    packages := []Package{}
    pkgcfg := filepath.Join(filepath.Dir(path), "packages.config")
    if _, err := os.Stat(pkgcfg); os.IsNotExist(err) {
        return packages
    }

    pkgscontent, err := ioutil.ReadFile(pkgcfg)
    if err != nil {
        return packages
    }

    pkgmatch := pkgreg.FindAllSubmatch(pkgscontent, -1)
    for _, m := range pkgmatch {
        id := string(m[1])
        version := string(m[2])
        frmwork := string(m[3])
        dev := string(m[5]) == "true"
        versionVal := int64(0)

        // deduct some kind of way to put a numeric value to the version string, to determine highest version
        // major gets highest weight (* 100000), minor gets 2nd weight (* 1000), and 3 digits of the last part gets * 10
        pts := strings.Split(version, ".")
        if len(pts) > 3 {
            pts = pts[:3]
        }
        m := int64(1)
        cv := int64(0)
        for i := len(pts) - 1; i >= 0; i-- {
            pt := pts[i]
            if len(pt) > 2 {
                pt = string(pt[:2])
            }
            parsed, _ := strconv.ParseInt(pt, 10, 64)
            cv = cv + (parsed * m)
            m = m * 100
        }

        versionVal = cv

        pkg := Package{Id: id, Version: version, TargetFramework: frmwork, DevelopmentDependency: dev, VersionVal: versionVal}
        packages = append(packages, pkg)
    }

    return packages
}

func loadTargetFramework(contents []byte) string {
    tfmatches := frmwkreg.FindAllSubmatch(contents, 1)
    var tf string
    if len(tfmatches) > 0 {
        tf = string(tfmatches[0][1])
    }
    return tf
}


func loadRootNS(contents []byte) string {
    nsmatches := nsreg.FindAllSubmatch(contents, 1)
    var ns string
    if len(nsmatches) > 0 {
        ns = string(nsmatches[0][1])
    }
    return ns
}

func loadReferences(contents []byte) ([]Reference, []ProjectReference) {
    refs := []Reference{}
    projrefs := []ProjectReference{}
    prefmatches := prefreg.FindAllSubmatch(contents, -1)

    for _, m := range prefmatches {
        name := string(m[1])
        hint := ""
        private := false

        hmatches := hpreg.FindAllSubmatch(m[2], -1)
        if len(hmatches) > 0 {
            hint = string(hmatches[0][1])
        }

        privmatches := privreg.FindAllSubmatch(m[2], -1)
        hasPrivate := false
        if len(privmatches) > 0 {
            spriv := string(privmatches[0][1])
            private = strings.ToLower(spriv) == "true"
            hasPrivate = true
        }

        isPkg := strings.Index(hint, "\\packages\\") != -1

        pkgname := ""
        if isPkg {
            pnmatches := pkgnamereg.FindAllSubmatch([]byte(hint), 1)
            if len(pnmatches) > 0 {
                n := string(pnmatches[0][1])
                npts := strings.Split(n, ".")
                for _, pt := range npts {
                    _, xerr := strconv.Atoi(pt)
                    if xerr != nil {
                        pkgname = pkgname + pt + "."
                    }
                }

                pkgname = string(pkgname[0 : len(pkgname)-1])
            }
        }

        assemblyName := strings.Split(name, ",")[0]
        ref := Reference{Name: assemblyName, Hint: hint, FullRef: name, Private: private, HasPrivate: hasPrivate, IsPackage: isPkg, PackageName: pkgname}

        refs = append(refs, ref)
    }

    // find all assembly name references
    amatches := arefreg.FindAllSubmatch(contents, -1)
    for _, m := range amatches {
        name := string(m[1])
        ref := Reference{Name: name, Hint: "", FullRef: name, HasPrivate: false}
        refs = append(refs, ref)
    }

    // find all packagereference references
    pkgmatch := pkgrefreg.FindAllSubmatch(contents, -1)
    for _, m := range pkgmatch {
        name := string(m[1])
        version := string(m[2])
        ref := Reference{Name: name, Hint: "", FullRef: name, HasPrivate: false, Version: version, IsPackage: true}
        refs = append(refs, ref)
    }

    projmatch := projrefreg.FindAllSubmatch(contents, -1)
    for _, m := range projmatch {
        path := string(m[1])
        name := string(m[2])

        pref := ProjectReference{Path: path, Name: name}
        projrefs = append(projrefs, pref)
    }

    return refs, projrefs
}

func loadFiles(contents []byte) []File {
    cmatches := creg.FindAllSubmatch(contents, -1)
    files := []File{}

    for _, m := range cmatches {
        t := string(m[1])
        f := string(m[2])

        file := File{Type: t, Path: f}
        files = append(files, file)
    }

    c2matches := c2reg.FindAllSubmatch(contents, -1)
    for _, m := range c2matches {
        t := string(m[1])
        f := string(m[2])

        file := File{Type: t, Path: f}
        files = append(files, file)
    }

    return files
}
