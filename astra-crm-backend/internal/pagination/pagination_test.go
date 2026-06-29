package pagination

import "testing"

func TestNormalizeAppliesDefaultsAndMaxPageSize(t *testing.T) {
	params := Normalize(Params{Page: -1, PageSize: 1000})
	if params.Page != DefaultPage {
		t.Fatalf("page = %d, want %d", params.Page, DefaultPage)
	}
	if params.PageSize != MaxPageSize {
		t.Fatalf("pageSize = %d, want %d", params.PageSize, MaxPageSize)
	}
}

func TestFromSliceReturnsRequestedPageAndTotal(t *testing.T) {
	result := FromSlice([]int{1, 2, 3, 4, 5}, Params{Page: 2, PageSize: 2})
	if result.Total != 5 {
		t.Fatalf("total = %d, want 5", result.Total)
	}
	if result.Page != 2 || result.PageSize != 2 {
		t.Fatalf("pagination = %d/%d, want 2/2", result.Page, result.PageSize)
	}
	if len(result.Items) != 2 || result.Items[0] != 3 || result.Items[1] != 4 {
		t.Fatalf("items = %v, want [3 4]", result.Items)
	}
}
