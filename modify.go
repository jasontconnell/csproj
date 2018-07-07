package csproj

func (p *Project) AddFile(path, t, subtype string) {
	f := File{Path: path, Type: t, SubType: subtype}
	p.Files = append(p.Files, f)
}

func (p *Project) GetFileMap() map[string]bool {
	m := make(map[string]bool)
	for _, f := range p.Files {
		m[f.Path] = true
	}
	return m
}
