package db

import (
	"encoding/hex"
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/starcoinorg/starcoin-go/client"
	"golang.org/x/crypto/sha3"
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
	return
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
	err := testDB.PutStarcoinTxCheck(k, v, client.Event{})
	if err != nil {
		t.FailNow()
	}
}

func TestGetAndDeleteAllStarcoinTxCheck(t *testing.T) {
	m, _ := testDB.GetAllStarcoinTxCheck()
	fmt.Println(m)
	for k, _ := range m {
		err := testDB.DeleteStarcoinTxCheck(k)
		if err != nil {
			t.FailNow()
		}
	}
}

func TestPutStarcoinTxRetry(t *testing.T) {
	uuid, _ := uuid.NewUUID()
	v, _ := uuid.MarshalBinary()
	err := testDB.PutStarcoinTxRetry(v, client.Event{})
	if err != nil {
		t.FailNow()
	}
}

func TestGetAndDeleteAllStarcoinTxRetry(t *testing.T) {
	cs, es, _ := testDB.GetAllStarcoinTxRetry()
	fmt.Println(cs)
	fmt.Println(es)
	for _, v := range cs {
		testDB.DeleteStarcoinTxRetry(v)
	}
}

// func TestHash(t *testing.T) {
// 	h := "2d052233fd5ae70d16898ca3eb40f55adbccc3dfe34e362c4bec50ec161c3461"
// 	txhash, _ := hex.DecodeString(h)
// 	hasher := New256Hasher()
// 	hasher.Write(txhash)
// 	hh := hasher.Sum(nil)
// 	fmt.Println(hex.EncodeToString(hh))
// }

func TestUpdatePolyTxNonMembershipProofByIndex(t *testing.T) {
	mysqldb := testDB.(*MySqlDB)
	err := mysqldb.UpdatePolyTxNonMembershipProofByIndex(2)
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}
}

func TestCalculatePloyTxInclusionRootHash(t *testing.T) {
	mysqldb := testDB.(*MySqlDB)
	polyTx, err := mysqldb.GetPolyTxByIndex(3)
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}
	rootHash, err := mysqldb.calculatePloyTxInclusionRootHash(polyTx)
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}
	fmt.Println(hex.EncodeToString(rootHash)) //eef849c21d4ceb722c7f4546f87ef6a2bb822765117cf6b848298007884fa80f
}

func TestPutPolyTx(t *testing.T) {
	uuid, _ := uuid.NewUUID()
	v, _ := uuid.MarshalBinary()

	tx, err := NewPolyTx(
		//TxHash:
		v,
		//Proof:
		v,
		//Header:
		v,
		//HeaderProof:
		v,
		//AnchorHeader:
		v,
		//HeaderSig:
		v,
		hex.EncodeToString(v),
	)
	//SmtRootHash:  hex.EncodeToString(v),
	//TxHashHash:   h,
	//SmtProofSideNodes:  hex.EncodeToString(v),
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}
	//return

	p, err := tx.GetPolyTxProof()
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}
	fmt.Println(p)
	//return

	// idx, err := testDB.PutPolyTx(tx) //hex.EncodeToString(v))
	// fmt.Println(idx, err)
	// if err != nil {
	// 	fmt.Println(err)
	// 	t.FailNow()
	// }

	//h := Hash256Hex(v)
	// m := NewPolyTxMapStore(testDB.(*MySqlDB), nil)
	// hh, _ := hex.DecodeString(h)
	// d, err := m.Get(hh)
	// fmt.Println(d, err)
	// if err != nil {
	// 	t.FailNow()
	// }

	// err = m.Set(hh, PolyTxExistsValue)
	// if err != nil {
	// 	t.FailNow()
	// }

}

func TestHasher(t *testing.T) {
	// Move version println:
	// [debug] (&) [1]
	// [debug] (&) [104, 101, 108, 108, 111, 119, 111, 114, 108, 100]
	// [debug] (&) [39, 103, 241, 92, 138, 242, 242, 199, 34, 93, 82, 115, 253, 214, 131, 237, 199, 20, 17, 10, 152, 125, 16, 84, 105, 124, 52, 138, 237, 78, 108, 199]
	// [debug] (&) [146, 218, 217, 68, 62, 77, 214, 215, 10, 127, 17, 135, 33, 1, 235, 255, 135, 226, 23, 152, 228, 251, 178, 111, 164, 191, 89, 14, 180, 64, 231, 27]
	oneByte := []byte{1}
	helloworld := []byte("helloworld")
	fmt.Println(oneByte)
	fmt.Println(helloworld)
	hasher := sha3.New256()
	hasher.Write(oneByte)
	fmt.Println(hasher.Sum(nil))
	fmt.Println(Hash256(helloworld))

	h, _ := hex.DecodeString("655e5461d6f009e968b1416ddde8407545144e98206f05bad4fdcba587907fbe")
	fmt.Println(hex.EncodeToString(Hash256(h))) //b908e1ffba13efa46efa2b92e8bb97cfa8a75d133e660631b247ca67f9da7f93

}

func TestSetPolyTxStatus(t *testing.T) {
	txHash := "testKey2"
	err := testDB.SetPolyTxStatus(txHash, STATUS_PROCESSED)
	if err != nil {
		t.FailNow()
	}
}

func TestGetFirstFailedPolyTx(t *testing.T) {
	px, err := testDB.GetFirstFailedPolyTx()
	if err != nil {
		t.FailNow()
	}
	fmt.Println(px)
}

// func TestMisc(t *testing.T) {
// 	currentMillis := time.Now().UnixNano() / 1000000
// 	fmt.Println(currentMillis)
// 	fmt.Println(currentTimeMillis())
// }
