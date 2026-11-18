package client

import (
	"net/http"
	"strings"
)

// operationMux matches a request to its pattern in operations.
var operationMux = func() *http.ServeMux {
	mux := http.NewServeMux()
	for pattern := range operations {
		mux.Handle(pattern, http.NotFoundHandler())
	}
	return mux
}()

// OperationName returns the operation req calls and its route template, such as
// "GetUserByID" and "/api/admin/users/{id}", for labelling metrics, spans and logs. Both
// are empty if no operation matches, such as when a proxy prefix is part of the path.
func OperationName(req *http.Request) (name, route string) {
	_, pattern := operationMux.Handler(req)
	if name = operations[pattern]; name != "" {
		_, route, _ = strings.Cut(pattern, " ")
	}
	return name, route
}
