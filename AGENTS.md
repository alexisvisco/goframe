### Description 

GoFrame is a framework that is based on file generation. 
- To know what will be generated for the developer you can check the cli/generators package
- core module is it contains contracts, basic types and some helpers 
- db module is it contains database related code
- docs folder contains nextra documentation
- http module contains http helper to work with http requests and responses
- mail module contains mailer helpers
- storage module contains storage helpers that implement contracts for storage providers

GoFrame is divided into modules to allow an user to pick only the modules he needs if he don't want to use the whole framework.

There is a particularity the cli/main.go is the cli to generate a goframe project. 
But the actual cli with all generators and co is generated with generators and the path would be the cmd/cli/main.go. 
