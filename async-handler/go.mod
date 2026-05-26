module ttuser/async-handler

go 1.21

require (
	github.com/apache/rocketmq-client-go/v2 v2.1.2
	github.com/teou/inji v1.1.2
	ttuser/config-client v0.0.0
	ttuser/pkg v0.0.0
)

replace (
	ttuser/config-client => ../config-client
	ttuser/pkg => ../pkg
)

require (
	github.com/emirpasic/gods v1.12.0 // indirect
	github.com/facebookgo/structtag v0.0.0-20150214074306-217e25fb9691 // indirect
	github.com/golang/mock v1.6.0 // indirect
	github.com/google/uuid v1.3.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/konsorten/go-windows-terminal-sequences v1.0.1 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/patrickmn/go-cache v2.1.0+incompatible // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/sirupsen/logrus v1.4.0 // indirect
	github.com/teou/implmap v0.0.0-20220223051011-e99c668c6344 // indirect
	github.com/teou/ordered_map v1.0.0 // indirect
	github.com/tidwall/gjson v1.13.0 // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.0 // indirect
	go.uber.org/atomic v1.10.0 // indirect
	go.uber.org/multierr v1.9.0 // indirect
	go.uber.org/zap v1.24.0 // indirect
	golang.org/x/crypto v0.31.0 // indirect
	golang.org/x/sys v0.28.0 // indirect
	golang.org/x/term v0.27.0 // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.0.0 // indirect
	stathat.com/c/consistent v1.0.0 // indirect
)
