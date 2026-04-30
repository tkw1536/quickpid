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

func ExampleUnmarshalStruct() {
	type Payload struct {
		Name string `json:"name"`
	}

	ok, err := strict.UnmarshalStruct[Payload]([]byte(`{"name":"alice"}`))
	fmt.Println(ok.Name, err == nil)

	_, err = strict.UnmarshalStruct[Payload]([]byte(`null`))
	fmt.Println(err != nil)

	_, err = strict.UnmarshalStruct[Payload]([]byte(`{"name":"alice","extra":1}`))
	fmt.Println(err != nil)

	// Output:
	// alice true
	// true
	// true
}
