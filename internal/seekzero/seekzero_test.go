//spellchecker:words seekzero
package seekzero_test

//spellchecker:words strings github quickpid internal seekzero
import (
	"fmt"
	"io"
	"strings"

	"github.com/tkw1536/quickpid/internal/seekzero"
)

func ExampleNewOnceSeekStartReader() {
	r := seekzero.NewOnceSeekStartReader(strings.NewReader("hello"))

	// Read the first three bytes; they are also buffered.
	first := make([]byte, 3)
	if _, err := io.ReadFull(r, first); err != nil {
		panic(err)
	}
	fmt.Println(string(first))

	// Reset and read the whole input from the start.
	if err := r.SeekToStart(); err != nil {
		panic(err)
	}
	all, err := io.ReadAll(r)
	if err != nil {
		panic(err)
	}
	fmt.Println(string(all))

	// Output:
	// hel
	// hello
}
