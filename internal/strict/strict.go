//
//spellchecker:words strict
package strict

func OptionalStringToPointer(value Optional[String]) *string {
	return (*string)(value.ToPointer())
}

func OptionalBoolToPointer(value Optional[Bool]) *bool {
	return (*bool)(value.ToPointer())
}
