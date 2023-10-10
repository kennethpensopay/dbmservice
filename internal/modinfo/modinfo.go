package modinfo

import (
	"crypto/md5"
	"encoding/hex"
	"golang.org/x/mod/modfile"
	"log"
	"os"
	"strings"
)

type ModuleInfo struct {
	Name string
}

func ModInfo() *ModuleInfo {
	goModBytes, err := os.ReadFile("go.mod")
	if err != nil {
		log.Fatalf("Could not read go.mod file.")
	}

	modName := strings.TrimSpace(modfile.ModulePath(goModBytes))
	if modName == "" {
		log.Println("Could not find module name for this application.")
	}
	return &ModuleInfo{
		Name: modName,
	}
}

func (m *ModuleInfo) ModuleNameAsMD5Sum() string {
	md5Sum := md5.Sum([]byte(m.Name))
	return hex.EncodeToString(md5Sum[:])
}
