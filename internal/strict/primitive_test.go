package strict_test

import (
	"encoding/json"
	"fmt"

	"github.com/tkw1536/quickpid/internal/strict"
)

func ExampleString() {
	var ok strict.String
	_ = json.Unmarshal([]byte(`"hello"`), &ok)
	fmt.Println(ok)

	var bad strict.String
	err := json.Unmarshal([]byte(`null`), &bad)
	fmt.Println(err)

	// Output:
	// hello
	// can only unmarshal string literal
}

func ExampleBool() {
	var okTrue strict.Bool
	_ = json.Unmarshal([]byte(`true`), &okTrue)
	fmt.Println(okTrue)

	var okFalse strict.Bool
	_ = json.Unmarshal([]byte(`false`), &okFalse)
	fmt.Println(okFalse)

	var bad strict.Bool
	err := json.Unmarshal([]byte(`null`), &bad)
	fmt.Println(err)

	// Output:
	// true
	// false
	// can only unmarshal boolean literal
}
