module github.com/mainvec/mvep/mvgen

go 1.24.0

toolchain go1.24.3

require (
	github.com/bufbuild/protocompile v0.14.1
	github.com/golang/protobuf v1.5.4
	github.com/jhump/protoreflect v1.17.0
	github.com/santhosh-tekuri/jsonschema/v6 v6.0.2
)

require (
	golang.org/x/sync v0.17.0 // indirect
	golang.org/x/text v0.30.0 // indirect
)

require (
	github.com/mainvec/mvpgo v0.4.1
	github.com/mainvec/ugo v0.5.1
	google.golang.org/protobuf v1.36.10
)

replace github.com/mainvec/mvpgo => ../../mvpgo
