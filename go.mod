module github.com/akaritrading/prices

go 1.15

require (
	github.com/akaritrading/libs v0.0.5
	github.com/go-chi/chi v4.1.2+incompatible
	github.com/gorilla/websocket v1.4.2
	github.com/pkg/errors v0.8.1
)

replace github.com/akaritrading/libs v0.0.5 => ../libs

// replace github.com/akaritrading/prices/pkg v0.0.0 => ./pkg
