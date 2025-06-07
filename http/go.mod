module github.com/alexisvisco/goframe/http

go 1.24

require (
	github.com/Oudwins/zog v0.21.1
	github.com/alexisvisco/goframe/core v0.0.0-20241022120000-abcdef123456
)

require golang.org/x/exp v0.0.0-20240613232115-7f521ea00fb8 // indirect

replace github.com/alexisvisco/goframe/core => ../core
