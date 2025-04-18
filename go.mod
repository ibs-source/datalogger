module github.com/ibs-source/datalogger

go 1.23.1

retract (
	[v0.1.4, v1.0.2] // The tags contain some bugs and have been retracted in the next version
)

require (
	github.com/google/uuid v1.6.0
	github.com/redis/go-redis/v9 v9.7.0
	github.com/sirupsen/logrus v1.9.3
)

require (
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	golang.org/x/sys v0.0.0-20220715151400-c0bba94af5f8 // indirect
)
