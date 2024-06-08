module github.com/workoak/wop/wog

go 1.22.1

require (
	github.com/bufbuild/protocompile v0.14.0
	github.com/golang/protobuf v1.5.4
	github.com/jhump/protoreflect v1.16.0
	github.com/santhosh-tekuri/jsonschema/v6 v6.0.0
	github.com/workoak/wop/wou v0.0.0-00010101000000-000000000000
)

require (
	golang.org/x/exp v0.0.0-20240604190554-fc45aab8b7f8 // indirect
	golang.org/x/sync v0.7.0 // indirect
	golang.org/x/text v0.16.0 // indirect
)

require (
//	github.com/leaanthony/clir v1.7.0
	google.golang.org/protobuf v1.34.2-0.20240529085009-ca837e5c658b
//github.com/golang/protobuf v1.5.0
//google.golang.org/genproto v0.0.0-20200526211855-cb27e3aa2013 // indirect
//google.golang.org/protobuf v1.34.1
)

replace github.com/workoak/wop/wou => ../wou
