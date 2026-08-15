//spellchecker:words filter
package filter_test

//spellchecker:words context errors testing github quickpid internal filter
import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/tkw1536/quickpid/internal/filter"
)

func TestFilter_allMatch(t *testing.T) {
	t.Parallel()

	get := sliceGetter([]int{1, 2, 3, 4, 5})
	got, err := filter.Filter(context.Background(), get, alwaysTrue[int], 10, 10, 0)
	if err != nil {
		t.Fatalf("Filter() error = %v", err)
	}
	assertResult(t, got, []int{1, 2, 3, 4, 5}, 5, 0)
}

func TestFilter_excludesNonMatching(t *testing.T) {
	t.Parallel()

	get := sliceGetter([]int{1, 2, 3, 4, 5, 6})
	even := func(_ context.Context, v int) (bool, error) {
		return v%2 == 0, nil
	}
	got, err := filter.Filter(context.Background(), get, even, 10, 10, 0)
	if err != nil {
		t.Fatalf("Filter() error = %v", err)
	}
	assertResult(t, got, []int{2, 4, 6}, 3, 0)
}

func TestFilter_pagination(t *testing.T) {
	t.Parallel()

	get := sliceGetter([]int{1, 2, 3, 4, 5, 6, 7, 8})
	even := func(_ context.Context, v int) (bool, error) {
		return v%2 == 0, nil
	}

	got, err := filter.Filter(context.Background(), get, even, 10, 2, 1)
	if err != nil {
		t.Fatalf("Filter() error = %v", err)
	}
	// matches: 2,4,6,8 — offset 1, limit 2 → 4,6; Total still 4
	assertResult(t, got, []int{4, 6}, 4, 1)
}

func TestFilter_offsetPastEnd(t *testing.T) {
	t.Parallel()

	get := sliceGetter([]int{1, 2, 3})
	got, err := filter.Filter(context.Background(), get, alwaysTrue[int], 10, 2, 10)
	if err != nil {
		t.Fatalf("Filter() error = %v", err)
	}
	assertResult(t, got, nil, 3, 10)
}

func TestFilter_emptySource(t *testing.T) {
	t.Parallel()

	get := sliceGetter([]int{})
	got, err := filter.Filter(context.Background(), get, alwaysTrue[int], 5, 5, 0)
	if err != nil {
		t.Fatalf("Filter() error = %v", err)
	}
	assertResult(t, got, nil, 0, 0)
}

func TestFilter_noneMatch(t *testing.T) {
	t.Parallel()

	get := sliceGetter([]int{1, 3, 5})
	even := func(_ context.Context, v int) (bool, error) {
		return v%2 == 0, nil
	}
	got, err := filter.Filter(context.Background(), get, even, 5, 5, 0)
	if err != nil {
		t.Fatalf("Filter() error = %v", err)
	}
	assertResult(t, got, nil, 0, 0)
}

func TestFilter_scansBeyondFirstPage(t *testing.T) {
	t.Parallel()

	// Source has 7 items; backend page size is 3, result page size is also 3.
	get := sliceGetter([]int{1, 2, 3, 4, 5, 6, 7})
	got, err := filter.Filter(context.Background(), get, alwaysTrue[int], 3, 3, 0)
	if err != nil {
		t.Fatalf("Filter() error = %v", err)
	}
	assertResult(t, got, []int{1, 2, 3}, 7, 0)
}

var errStoreWantError = errors.New("store failed")

func TestFilter_getterError(t *testing.T) {
	t.Parallel()

	get := func(context.Context, int, int) ([]int, error) {
		return nil, errStoreWantError
	}
	_, err := filter.Filter(context.Background(), get, alwaysTrue[int], 5, 5, 0)
	if err == nil {
		t.Fatal("Filter() error = nil, want wrapped getter error")
	}
	if !errors.Is(err, errStoreWantError) {
		t.Fatalf("Filter() error = %v, want %v", err, errStoreWantError)
	}
}

var errConditionWantError = errors.New("condition failed")

func TestFilter_conditionError(t *testing.T) {
	t.Parallel()

	get := sliceGetter([]int{1, 2, 3})
	condition := func(_ context.Context, v int) (bool, error) {
		if v == 2 {
			return false, errConditionWantError
		}
		return true, nil
	}
	_, err := filter.Filter(context.Background(), get, condition, 5, 5, 0)
	if err == nil {
		t.Fatal("Filter() error = nil, want wrapped condition error")
	}
	if !errors.Is(err, errConditionWantError) {
		t.Fatalf("Filter() error = %v, want %v", err, errConditionWantError)
	}
}

func sliceGetter[V any](all []V) filter.PaginatedGetterFunc[V] {
	return func(_ context.Context, limit, offset int) ([]V, error) {
		if offset >= len(all) {
			return nil, nil
		}
		end := offset + limit
		if end > len(all) {
			end = len(all)
		}
		page := make([]V, end-offset)
		copy(page, all[offset:end])
		return page, nil
	}
}

func alwaysTrue[V any](context.Context, V) (bool, error) {
	return true, nil
}

func assertResult[V comparable](t *testing.T, got *filter.FilterResult[V], wantItems []V, wantTotal, wantOffset int) {
	t.Helper()
	if got == nil {
		t.Fatal("Filter() result = nil")
	}
	if got.Total != wantTotal {
		t.Errorf("Total = %d, want %d", got.Total, wantTotal)
	}
	if got.Offset != wantOffset {
		t.Errorf("Offset = %d, want %d", got.Offset, wantOffset)
	}
	if fmt.Sprint(got.Items) != fmt.Sprint(wantItems) {
		t.Errorf("Items = %v, want %v", got.Items, wantItems)
	}
}
