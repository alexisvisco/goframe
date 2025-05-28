module github.com/alexisvisco/goframe/storage

go 1.24

require (
	github.com/alexisvisco/goframe/core v0.0.0-20241022120000-abcdef123456
	github.com/alexisvisco/goframe/http v0.0.0-20241022120000-abcdef123456
	github.com/gabriel-vasile/mimetype v1.4.9
	github.com/minio/minio-go v6.0.14+incompatible
	github.com/minio/minio-go/v7 v7.0.92
	github.com/nrednav/cuid2 v1.0.1
)

require (
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/go-ini/ini v1.67.0 // indirect
	github.com/goccy/go-json v0.10.5 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/klauspost/compress v1.18.0 // indirect
	github.com/klauspost/cpuid/v2 v2.2.10 // indirect
	github.com/minio/crc64nvme v1.0.1 // indirect
	github.com/minio/md5-simd v1.1.2 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/philhofer/fwd v1.1.3-0.20240916144458-20a13a1f6b7c // indirect
	github.com/rs/xid v1.6.0 // indirect
	github.com/tinylib/msgp v1.3.0 // indirect
	golang.org/x/crypto v0.37.0 // indirect
	golang.org/x/net v0.39.0 // indirect
	golang.org/x/sys v0.32.0 // indirect
	golang.org/x/text v0.25.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace (
	github.com/alexisvisco/goframe/core => ../core
	github.com/alexisvisco/goframe/http => ../http
)
