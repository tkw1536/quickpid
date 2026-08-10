package openapi_test

//spellchecker:words openapi
import (
	"fmt"

	"github.com/tkw1536/quickpid/internal/openapi"
)

func ExampleSetServersPath() {
	input := []byte(`openapi: 3.0.3
info:
  title: Example
  version: 1.0.0
servers:
  - url: https://old.example.com
paths: {}
`)
	out, err := openapi.SetServersPath(input, "https://api.example.com")
	if err != nil {
		panic(err)
	}
	fmt.Print(string(out))

	// Output:
	// openapi: 3.0.3
	// info:
	//     title: Example
	//     version: 1.0.0
	// servers:
	//     - url: https://api.example.com
	// paths: {}
}
