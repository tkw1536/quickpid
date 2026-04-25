package bitstring_test

import (
	"fmt"
	"strings"

	"github.com/tkw1536/quickpid/internal/bitstring"
)

func ExampleNewReader() {

	r := bitstring.NewReader()

	bytes := make([]byte, 1)
	if _, err := r.Read(bytes); err != nil {
		panic(err)
	}

	var b strings.Builder
	for _, by := range bytes {
		fmt.Fprintf(&b, "%08b", by)
	}
	fmt.Println(b.String())

	// Output: 01000110
}
