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

// func TestUpdatePolyHeight(t *testing.T) {
// 	err := testDB.UpdatePolyHeight(141)
// 	fmt.Println(err)
// }

func TestGetMethods(t *testing.T) {
	h, err := testDB.GetPolyHeight()
	if err != nil {
		t.FailNow()
	}
	fmt.Println(h)
	p, err := testDB.GetPolyTx("foo")
	if err != nil {
		t.FailNow()
	}
	if p.TxHash != "foo" {
		t.FailNow()
	}
	fmt.Println(p)
	testMySqlDB := testDB.(*MySqlDB)
	p2, err := testMySqlDB.getPolyTxByTxHashHash("15291f67d99ea7bc578c3544dadfbb991e66fa69cb36ff70fe30e798e111ff5f")
	if err != nil {
		t.FailNow()
	}
	if p2.TxHashHash != "15291f67d99ea7bc578c3544dadfbb991e66fa69cb36ff70fe30e798e111ff5f" {
		t.FailNow()
	}
	fmt.Println(p2)

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
	h := Sha256HashHex(v)
	tx, err := NewPolyTx(
		//TxHash:
		hex.EncodeToString(v),
		//Proof:
		v,
		//Header:
		v,
		//HeaderProof:
		v,
		//AnchorHeader:
		v,
		//HeaderSig:
		v)
	//SmtRootHash:  hex.EncodeToString(v),
	//TxHashHash:   h,
	//SmtProofSideNodes:  hex.EncodeToString(v),
	if err != nil {
		t.FailNow()
	}

	p, err := tx.GetPolyTxProof()
	if err != nil {
		t.FailNow()
	}
	fmt.Println(p)

	idx, err := testDB.PutPolyTx(tx) //hex.EncodeToString(v))
	fmt.Println(idx, err)
	if err != nil {
		t.FailNow()
	}

	m := NewPolyTxMapStore(testDB.(*MySqlDB), nil)
	hh, _ := hex.DecodeString(h)
	d, err := m.Get(hh)
	fmt.Println(d, err)
	if err != nil {
		t.FailNow()
	}

	err = m.Set(hh, PolyTxExistsValue)
	if err != nil {
		t.FailNow()
	}

}
