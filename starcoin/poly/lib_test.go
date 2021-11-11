package poly

import (
	"fmt"
	"testing"
)

func TestParseModuleId(t *testing.T) {
	m := ParseModuleId("0x00000000000000000000000000000001::STC")
	fmt.Println(m)
}
