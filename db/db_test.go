package db

import (
	"fmt"
	"testing"
)

func TestUpdatePolyHeight(t *testing.T) {
	var db DB
	db, err := NewBoltDB("test-boltdb")
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}
	err = db.UpdatePolyHeight(100)
	fmt.Println(err)
}
