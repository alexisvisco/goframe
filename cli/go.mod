module github.com/alexisvisco/goframe/cli

go 1.24

require (
	github.com/alexisvisco/goframe/core v0.0.0-20241022120000-abcdef123456
	github.com/alexisvisco/goframe/db v0.0.0-20241022120000-abcdef123456
	github.com/spf13/cobra v1.9.1
        github.com/stretchr/testify v1.10.0
        golang.org/x/mod v0.17.0
        golang.org/x/tools v0.17.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/spf13/pflag v1.0.6 // indirect
	golang.org/x/text v0.25.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace (
	github.com/alexisvisco/goframe/core => ../core
	github.com/alexisvisco/goframe/db => ../db
)
