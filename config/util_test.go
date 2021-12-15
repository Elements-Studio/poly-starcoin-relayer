package config

import (
	"fmt"
	"testing"
)

func TestReplaceJsonEnvs(t *testing.T) {
	filePath := "../config-devnet.json"
	j, err := ReadFile(filePath)
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}
	j = replaceJsonEnvs(j)
	fmt.Println(string(j))
}
