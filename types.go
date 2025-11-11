package vulkango

import "C"

import "fmt"

type Result int32

const (
	SUCCESS Result = 0
	// ... other result codes
)

func (r Result) Error() string {
	// Convert result codes to strings
	switch r {
	case SUCCESS:
		return "SUCCESS"
	default:
		return fmt.Sprintf("VkResult(%d)", r)
	}
}
