package csproj

type Project struct {
	Key              string
	FullPath         string
	Filename         string // includes extension
	FrameworkVersion string
	RootNamespace    string
	AssemblyName     string
	References       []Reference
	ProjectRefs      []ProjectReference
	Files            []File
	Packages         []Package
	Zip              string
}

type Reference struct {
	Name       string
	Hint       string
	FullRef    string
	Private    bool
	HasPrivate bool

	PackageName        string
	IsPackage          bool
	IsPackageReference bool
	Version            string
}

type ProjectReference struct {
	Key           string
	Name          string
	Path          string
	RootNamespace string
}

type File struct {
	Path    string
	Type    string // Include, None, Compile, etc
	SubType string
}

type Package struct {
	Id                    string
	Version               string
	TargetFramework       string
	DevelopmentDependency bool
	VersionVal            int64
}
