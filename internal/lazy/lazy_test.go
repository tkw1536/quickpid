//spellchecker:words lazy
package lazy_test

//spellchecker:words github quickpid internal lazy
import (
	"fmt"

	"github.com/tkw1536/quickpid/internal/lazy"
)

func ExampleLazy() {
	var l lazy.Lazy[int]

	// the first invocation to lazy will be called and set the value
	fmt.Println(l.Get(func() int { return 42 }))

	// the second invocation will not call init again, using the first value
	fmt.Println(l.Get(func() int { return 43 }))

	// Output: 42
	// 42
}

func ExampleLazy_nil() {
	var l lazy.Lazy[int]

	// passing nil as the initialization function causes the zero value to be set
	fmt.Println(l.Get(nil))

	// Output: 0
}
