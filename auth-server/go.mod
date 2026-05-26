module ttuser/auth-server

go 1.18

require (
	github.com/golang-jwt/jwt/v5 v5.0.0
	github.com/google/uuid v1.3.0
	github.com/teou/inji v1.1.2
	golang.org/x/crypto v0.14.0
	google.golang.org/grpc v1.56.3
	ttuser/auth-client v0.0.0
	ttuser/data-store v0.0.0
	ttuser/event-producer v0.0.0
)

require (
	github.com/apache/rocketmq-client-go/v2 v2.1.2 // indirect
	github.com/emirpasic/gods v1.12.0 // indirect
	github.com/facebookgo/structtag v0.0.0-20150214074306-217e25fb9691 // indirect
	github.com/go-sql-driver/mysql v1.7.1 // indirect
	github.com/golang/mock v1.3.1 // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/konsorten/go-windows-terminal-sequences v1.0.1 // indirect
	github.com/modern-go/concurrent v0.0.0-20180228061459-e0a39a4cb421 // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/patrickmn/go-cache v2.1.0+incompatible // indirect
	github.com/pkg/errors v0.8.1 // indirect
	github.com/sirupsen/logrus v1.4.0 // indirect
	github.com/teou/implmap v0.0.0-20220223051011-e99c668c6344 // indirect
	github.com/teou/ordered_map v1.0.0 // indirect
	github.com/tidwall/gjson v1.13.0 // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.0 // indirect
	go.uber.org/atomic v1.5.1 // indirect
	golang.org/x/lint v0.0.0-20190930215403-16217165b5de // indirect
	golang.org/x/net v0.10.0 // indirect
	golang.org/x/sys v0.13.0 // indirect
	golang.org/x/term v0.13.0 // indirect
	golang.org/x/text v0.13.0 // indirect
	golang.org/x/tools v0.6.0 // indirect
	google.golang.org/genproto v0.0.0-20230410155749-daa745c078e1 // indirect
	google.golang.org/protobuf v1.31.0 // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.0.0 // indirect
	stathat.com/c/consistent v1.0.0 // indirect
)

replace (
	ttuser/auth-client => ../auth-client
	ttuser/data-store => ../data-store
	ttuser/event-producer => ../event-producer
)
