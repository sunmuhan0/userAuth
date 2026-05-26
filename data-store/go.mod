module ttuser/data-store

go 1.18

require (
	github.com/go-sql-driver/mysql v1.7.1
	github.com/teou/implmap v0.0.0-20220223051011-e99c668c6344
	github.com/teou/inji v1.1.2
	ttuser/config-client v0.0.0
)

require (
	github.com/facebookgo/structtag v0.0.0-20150214074306-217e25fb9691 // indirect
	github.com/teou/ordered_map v1.0.0 // indirect
)

replace ttuser/config-client => ../config-client
