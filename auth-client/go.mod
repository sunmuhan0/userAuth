module ttuser/auth-client

go 1.18

require (
	github.com/teou/inji v1.1.2
	google.golang.org/grpc v1.56.3
	google.golang.org/protobuf v1.31.0
	ttuser/config-client v0.0.0
	ttuser/pkg v0.0.0
)

replace ttuser/config-client => ../config-client

replace ttuser/pkg => ../pkg

require (
	github.com/facebookgo/structtag v0.0.0-20150214074306-217e25fb9691 // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/teou/implmap v0.0.0-20181215111212-373d77bc2b63 // indirect
	github.com/teou/ordered_map v1.0.0 // indirect
	golang.org/x/net v0.10.0 // indirect
	golang.org/x/sys v0.11.0 // indirect
	golang.org/x/text v0.9.0 // indirect
	google.golang.org/genproto v0.0.0-20230410155749-daa745c078e1 // indirect
)
