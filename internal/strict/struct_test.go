package strict_test

import (
	"fmt"

	"github.com/tkw1536/quickpid/internal/strict"
)

func ExampleMustBeStruct() {
	fmt.Println(strict.MustBeStruct([]byte(`{}`)) == nil)
	fmt.Println(strict.MustBeStruct([]byte(`null`)) == nil)
	fmt.Println(strict.MustBeStruct([]byte(`[]`)) == nil)

	// Output:
	// true
	// false
	// false
}

