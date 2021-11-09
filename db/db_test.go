package db

import (
	"encoding/hex"
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"
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

func GetPolyHeight(t *testing.T) {
	h, err := testDB.GetPolyHeight()
	if err != nil {
		t.FailNow()
	}
	fmt.Println(h)
}

func TestPutStarcoinTxCheck(t *testing.T) {
	uuid, _ := uuid.NewUUID()
	k := strings.Replace(uuid.String(), "-", "", -1)
	v, _ := uuid.MarshalBinary()
	testDB.PutStarcoinTxCheck(k, v)
}

func TestGetAndDeleteAllStarcoinTxCheck(t *testing.T) {
	m, _ := testDB.GetAllStarcoinTxCheck()
	fmt.Println(m)
	for k, _ := range m {
		testDB.DeleteStarcoinTxCheck(k)
	}
}

func TestPutStarcoinTxRetry(t *testing.T) {
	uuid, _ := uuid.NewUUID()
	v, _ := uuid.MarshalBinary()
	testDB.PutStarcoinTxRetry(v)
}

func TestGetAndDeleteAllStarcoinTxRetry(t *testing.T) {
	m, _ := testDB.GetAllStarcoinTxRetry()
	fmt.Println(m)
	for _, v := range m {
		testDB.DeleteStarcoinTxRetry(v)
	}
}

func TestPutPolyTx(t *testing.T) {
	uuid, _ := uuid.NewUUID()
	v, _ := uuid.MarshalBinary()
	idx, _ := testDB.PutPolyTx(hex.EncodeToString(v))
	fmt.Println(idx)
}
