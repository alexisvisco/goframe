module github.com/alexisvisco/goframe/cache

go 1.24

require (
	github.com/alexisvisco/goframe/core v0.0.0-20241022120000-abcdef123456
	github.com/alexisvisco/goframe/db v0.0.0-20250703071310-7ec00ae0bf76
	github.com/lib/pq v1.10.9
	gorm.io/gorm v1.25.5
)

require (
	github.com/Oudwins/zog v0.21.3 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	golang.org/x/exp v0.0.0-20240613232115-7f521ea00fb8 // indirect
)

replace github.com/alexisvisco/goframe/core => ../core
