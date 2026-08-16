//spellchecker:words filter
package filter

//spellchecker:words context
import (
	"context"
	"fmt"
)

// Filter walks every item from get, keeps those for which condition returns true, then
// applies limit/offset pagination over the matching items.
//
// Total is the number of items that satisfied condition across the entire source, not
// merely the current page. Items holds at most limit matches starting after offset
// matching items (so offset 0 is the first match). Offset in the result echoes the
// requested offset.
//
// Get is called repeatedly with page size equal to pageLimit until the source is exhausted.
// Errors from Get or condition are returned immediately and stop iteration.
func Filter[V any](
	ctx context.Context,
	get PaginatedGetterFunc[V],
	condition func(context.Context, V) (bool, error),
	pageLimit int,
	limit, offset int,
) (*FilterResult[V], error) {
	// The current api response
	//
	// The Total field will be updated as we encounter valid items.
	var res FilterResult[V]

	for value := range get.all(ctx, pageLimit) {
		if value.err != nil {
			return nil, fmt.Errorf("failed to get values: %w", value.err)
		}

		// skip invalid values
		include, err := condition(ctx, value.value)
		if err != nil {
			return nil, fmt.Errorf("failed to check condition: %w", err)
		}
		if !include {
			continue
		}
		res.Total++

		// only add values according to the underlying offset and limit.
		if res.Total > offset && len(res.Items) < limit {
			res.Items = append(res.Items, value.value)
		}
	}

	res.Offset = offset
	return &res, nil
}

// FilterResult is the result of a call to [Filter].
type FilterResult[V any] struct {
	// Items is the current page of items.
	Items []V
	// Total is the total number of items matching the condition.
	Total int
	// Offset is the offset of the current page.
	Offset int
}

// PaginatedGetterFunc fetches one page of items from an underlying backend.
//
// limit is the maximum number of items to return; offset is how many items to skip
// from the start of the full unfiltered sequence. An empty slice (and a nil error)
// means there are no further items.
type PaginatedGetterFunc[V any] func(ctx context.Context, limit, offset int) ([]V, error)

type valueOrError[V any] struct {
	err   error
	value V
}

// all returns a channel that yields all values from the paginated getter.
// pageLimit is the page size used for each call to the getter.
func (p PaginatedGetterFunc[V]) all(ctx context.Context, pageLimit int) <-chan valueOrError[V] {
	ch := make(chan valueOrError[V], pageLimit)
	go func() {
		defer close(ch)

		offset := 0

		for {
			// list the current page or bail out in case of an error.
			page, err := p(ctx, pageLimit, offset)
			if err != nil {
				ch <- valueOrError[V]{err: err}
				return
			}

			// yield individual items.
			for _, value := range page {
				ch <- valueOrError[V]{value: value}
			}

			// A short or empty page means the source is exhausted.
			if len(page) == 0 || len(page) < pageLimit {
				return
			}
			offset += len(page)
		}
	}()

	return ch
}
