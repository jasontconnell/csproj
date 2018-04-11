package csproj

import (
    "os"
    "path/filepath"
)

func Find(relativeTo, exclude string) []string {
    paths := []string{}
    path := filepath.Dir(relativeTo) // others should be underneath
    filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
        if info.Name() == exclude {
            return nil
        }
        ext := filepath.Ext(p)
        if ext == ".csproj" {
            paths = append(paths, p)
        }
        return nil
    })

    return paths
}
