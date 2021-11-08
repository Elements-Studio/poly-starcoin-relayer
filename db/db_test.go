package db

import (
	"fmt"
	"testing"
)

var (
	testDB DB
)

func init() {
	//db, err := NewBoltDB("test-boltdb")
	db, err := NewMySqlDB("root:123456@tcp(127.0.0.1:3306)/poly_starcoin?charset=utf8mb4&parseTime=True&loc=Local")
	if err != nil {
		fmt.Println(err)
		//t.FailNow()
	}
	testDB = db
}

func TestUpdatePolyHeight(t *testing.T) {
	err := testDB.UpdatePolyHeight(141)
	fmt.Println(err)
}

func TestGet(t *testing.T) {
	h, err := testDB.GetPolyHeight()
	if err != nil {
		t.FailNow()
	}
	fmt.Println(h)
}
