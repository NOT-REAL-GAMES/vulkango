package vulkango

/*
#cgo LDFLAGS: -lvulkango

#include "vulkango.h"


*/
import "C"
import "fmt"

func main() {
	fmt.Printf("%d", C.VK_SUCCESS)
}
