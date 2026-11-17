package client

import (
	"fmt"
	"net/http"
	"slices"
	"testing"
)

// twoPages fakes a list of two pages that, like Pocket ID, answers a page past the end with the last one.
func twoPages(requested *[]int) func(int) (*http.Response, error) {
	return func(page int) (*http.Response, error) {
		*requested = append(*requested, page)
		page = min(page, 2)
		return response(200, jsonHeader, fmt.Sprintf(`{"data":[{"id":"u%d"}],"pagination":{"currentPage":%[1]d,"totalPages":2}}`, page)), nil
	}
}

func TestAll(t *testing.T) {
	var requested []int
	var ids []string
	for u, err := range All[UserDto](twoPages(&requested)) {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, u.ID)
	}
	if !slices.Equal(ids, []string{"u1", "u2"}) || !slices.Equal(requested, []int{1, 2}) {
		t.Fatalf("got %v from pages %v", ids, requested)
	}

	requested = nil
	for range All[UserDto](twoPages(&requested)) {
		break
	}
	if !slices.Equal(requested, []int{1}) {
		t.Fatalf("breaking after the first item fetched pages %v", requested)
	}
}

func TestAllError(t *testing.T) {
	first := twoPages(new([]int))
	list := func(page int) (*http.Response, error) {
		if page > 1 {
			return response(502, nil, ""), nil
		}
		return first(page)
	}

	var ids []string
	var errs []error
	for u, err := range All[UserDto](list) {
		ids, errs = append(ids, u.ID), append(errs, err)
	}
	if !slices.Equal(ids, []string{"u1", ""}) || errs[0] != nil || AsAPIError(errs[1]) == nil {
		t.Fatalf("got %v, %v", ids, errs)
	}
}
