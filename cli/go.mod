module github.com/alexisvisco/goframe/cli

go 1.24

require (
	github.com/alexisvisco/goframe/core v0.0.0-20241022120000-abcdef123456
	github.com/alexisvisco/goframe/db v0.0.0-20241022120000-abcdef123456
	github.com/spf13/cobra v1.9.1
	github.com/stretchr/testify v1.10.0
	go.uber.org/fx v1.24.0
	golang.org/x/mod v0.17.0
	golang.org/x/tools v0.21.1-0.20240508182429-e35e4ccd0d2d
	gopkg.in/yaml.v3 v3.0.1
	gorm.io/gorm v1.25.5
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/spf13/pflag v1.0.6 // indirect
	go.uber.org/dig v1.19.0 // indirect
	go.uber.org/multierr v1.10.0 // indirect
	go.uber.org/zap v1.26.0 // indirect
	golang.org/x/sys v0.20.0 // indirect
	golang.org/x/text v0.25.0 // indirect
)

replace (
	github.com/alexisvisco/goframe/core => ../core
	github.com/alexisvisco/goframe/db => ../db
)
