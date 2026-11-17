package client

import (
	"iter"
	"net/http"
)

// All yields every item of a paginated list, fetching pages 1, 2, … with list until the last
// page or the first error, as in
//
//	users := client.All[client.UserDto](func(page int) (*http.Response, error) {
//		return c.ListUsers(ctx, &client.ListUsersParams{PaginationPage: &page, PaginationLimit: new(100)})
//	})
func All[T any](list func(page int) (*http.Response, error)) iter.Seq2[T, error] {
	return func(yield func(T, error) bool) {
		for page := 1; ; page++ {
			p, err := Decode[struct {
				Data       []T                `json:"data"`
				Pagination PaginationResponse `json:"pagination"`
			}](list(page))
			if err != nil {
				yield(*new(T), err)
				return
			}
			for _, item := range p.Data {
				if !yield(item, nil) {
					return
				}
			}
			// Pocket ID answers a page past the end with the last page, never an empty one.
			if int64(p.Pagination.CurrentPage) >= p.Pagination.TotalPages {
				return
			}
		}
	}
}
