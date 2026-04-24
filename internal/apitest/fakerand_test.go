package apitest

import (
	"fmt"
	"strings"
)

func ExampleNewFakeRandReader() {

	r := NewFakeRandReader()

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
