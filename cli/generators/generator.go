package generators

import "github.com/alexisvisco/goframe/core/configuration"

type fileinfo struct {
	dir  bool
	path string
	kind uint8
}

type Generator struct {
	GoModuleName string
	DatabaseType configuration.DatabaseType
	ORMType      string
	Maintainer   bool
	Web          bool
	Docker       bool

	filesCreated []fileinfo
}

func (g *Generator) Databases() *GenerateDatabaseFiles {
	return &GenerateDatabaseFiles{g: g}
}
