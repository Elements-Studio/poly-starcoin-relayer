package db

import (
	"encoding/hex"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/elements-studio/poly-starcoin-relayer/config"
	"github.com/google/uuid"
	"github.com/starcoinorg/starcoin-go/client"
	"golang.org/x/crypto/sha3"
)

// func TestUpdatePolyHeight(t *testing.T) {
// 	err := testDB().UpdatePolyHeight(141)
// 	fmt.Println(err)
// }

func TestGetMethods(t *testing.T) {
	h, err := devNetDB().GetPolyHeight()
	if err != nil {
		t.FailNow()
	}
	fmt.Println(h)
	return
	p, err := devNetDB().GetPolyTx("foo", getTestFromChainId())
	if err != nil {
		t.FailNow()
	}
	if p.TxHash != "foo" {
		t.FailNow()
	}
	fmt.Println(p)
	testMySqlDB := devNetDB().(*MySqlDB)
	p2, err := testMySqlDB.getPolyTxBySmtTxPath("15291f67d99ea7bc578c3544dadfbb991e66fa69cb36ff70fe30e798e111ff5f")
	if err != nil {
		t.FailNow()
	}
	if p2.SmtTxPath != "15291f67d99ea7bc578c3544dadfbb991e66fa69cb36ff70fe30e798e111ff5f" {
		t.FailNow()
	}
	fmt.Println(p2)

}

func TestGetAllPolyTxRetry(t *testing.T) {
	db := testNetDB()

	// // insert a PolyTxRetry
	// uuid, _ := uuid.NewUUID()
	// v, _ := uuid.MarshalBinary()
	// r, err := NewPolyTxRetry(v, getTestFromChainId(), []byte{}, new(msg.Tx))
	// if err != nil {
	// 	t.FailNow()
	// }
	// err = db.PutPolyTxRetry(r)
	// if err != nil {
	// 	t.FailNow()
	// }

	list, err := db.GetAllPolyTxRetry()
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}
	fmt.Println(list)

	for _, m := range list {
		err := db.DeletePolyTxRetry(m.TxHash, m.FromChainID)
		if err != nil {
			fmt.Println(err)
			t.FailNow()
		}
	}
}

func TestPutStarcoinTxCheck(t *testing.T) {
	uuid, _ := uuid.NewUUID()
	k := strings.Replace(uuid.String(), "-", "", -1)
	v, _ := uuid.MarshalBinary()
	err := devNetDB().PutStarcoinTxCheck(k, v, client.Event{})
	if err != nil {
		t.FailNow()
	}
}

func TestGetAndDeleteAllStarcoinTxCheck(t *testing.T) {
	m, _ := devNetDB().GetAllStarcoinTxCheck()
	fmt.Println(m)
	for k, _ := range m {
		err := devNetDB().DeleteStarcoinTxCheck(k)
		if err != nil {
			t.FailNow()
		}
	}
}

func TestPutStarcoinTxRetry(t *testing.T) {
	uuid, _ := uuid.NewUUID()
	v, _ := uuid.MarshalBinary()
	err := devNetDB().PutStarcoinTxRetry(v, client.Event{})
	if err != nil {
		t.FailNow()
	}
}

func TestGetAndDeleteAllStarcoinTxRetry(t *testing.T) {
	cs, es, _ := devNetDB().GetAllStarcoinTxRetry()
	fmt.Println(cs)
	fmt.Println(es)
	for _, v := range cs {
		devNetDB().DeleteStarcoinTxRetry(v)
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
	//mysqldb := devNetDB().(*MySqlDB)
	mysqldb := testNetDB().(*MySqlDB)
	err := mysqldb.UpdatePolyTxNonMembershipProofByIndex(21)
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}
}

func TestComputePloyTxInclusionRootHash(t *testing.T) {
	mysqldb := devNetDB().(*MySqlDB)
	polyTx, err := mysqldb.GetPolyTxByIndex(3)
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}
	rootHash, err := mysqldb.computePloyTxInclusionRootHash(polyTx)
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}
	fmt.Println(hex.EncodeToString(rootHash)) //eef849c21d4ceb722c7f4546f87ef6a2bb822765117cf6b848298007884fa80f
	fmt.Println(rootHash)
}

func TestPutPolyTx(t *testing.T) {
	uuid, _ := uuid.NewUUID()
	v, _ := uuid.MarshalBinary()
	fromChainId := getTestFromChainId()
	tx, err := NewPolyTx(
		//TxHash:
		v,
		fromChainId,
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
	//SmtTxPath:   ...,
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

	// idx, err := testDB().PutPolyTx(tx) //hex.EncodeToString(v))
	// fmt.Println(idx, err)
	// if err != nil {
	// 	fmt.Println(err)
	// 	t.FailNow()
	// }

	//h := Hash256Hex(v)
	// m := NewPolyTxMapStore(testDB().(*MySqlDB), nil)
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

func TestRemovePolyTx(t *testing.T) {
	db := testNetDB()
	txHash := "90f0a7b0c9b4d5d556b8058caeaaf5fee2b249340efd3cfa6325cc44adaab4f7"
	fromChainId := 318
	polyTx, err := db.GetPolyTx(txHash, uint64(fromChainId))
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}
	err = db.RemovePolyTx(polyTx)
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}
}

func TestPushBackRemovePolyTx(t *testing.T) {
	db := testNetDB()
	removedId := 1
	err := db.PushBackRemovePolyTx(uint64(removedId))
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}
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
	fromChainID := getTestFromChainId()
	txHash := "testKey2"
	err := devNetDB().SetPolyTxStatus(txHash, fromChainID, STATUS_PROCESSED)
	if err != nil {
		t.FailNow()
	}
}

func TestGetFirstFailedPolyTx(t *testing.T) {
	px1, err := devNetDB().GetFirstFailedPolyTx()
	if err != nil {
		t.FailNow()
	}
	if px1 != nil {
		fmt.Printf("First failed PolyTx index: %v\n", px1.TxIndex)
	} else {
		fmt.Println(px1)
	}
	// // ////////////////////////
	// px2, err := testDB().GetFirstFailedPolyTx()
	// if err != nil {
	// 	t.FailNow()
	// }
	// if px2 != nil {
	// 	fmt.Printf("First failed PolyTx index: %v\n", px2.TxIndex)
	// } else {
	// 	fmt.Println(px2)
	// }
}

func TestGetFirstTimedOutPolyTx(t *testing.T) {
	px1, err := testNetDB().GetFirstTimedOutPolyTx()
	if err != nil {
		t.FailNow()
	}
	if px1 != nil {
		fmt.Printf("First timed-out PolyTx index: %v\n", px1.TxIndex)
	} else {
		fmt.Println(px1)
	}
}

func TestGetTimedOutOrFailedPolyTxList(t *testing.T) {
	pxList, err := testNetDB().GetTimedOutOrFailedPolyTxList()
	if err != nil {
		t.FailNow()
	}
	if pxList != nil {
		for _, px := range pxList {
			fmt.Printf("Current timed-out or failed PolyTx in list, index: %v, Starcoin TxHash: %s\n", px.TxIndex, px.StarcoinTxHash)
		}
	} else {
		fmt.Println(pxList)
	}
}

func TestUpdatePolyTransactionsToProcessedBeforeIndex(t *testing.T) {
	db := devNetDB().(*MySqlDB)
	err := db.updatePolyTransactionsToProcessedBeforeIndex(17)
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}
}

func TestSetPolyTxStatusProcessing(t *testing.T) {
	db := testNetDB()
	px_TxHash := "91139fd7194a2e7f95374bf797437faeda9d4bceef29dbfea296cc48d7929bf2"
	var fromChainId uint64 = 318 //getTestFromChainId()

	for i := 0; i < 2; i++ {
		go func() {
			//db.GetFirstFailedPolyTx()
			tx, err := db.GetPolyTx(px_TxHash, fromChainId)
			if err != nil {
				fmt.Println(err)
				return
			}
			if tx == nil || tx.Status == STATUS_CONFIRMED || tx.Status == STATUS_PROCESSED {
				fmt.Println("STATUS_PROCESSED")
				return
			}
			err = db.SetPolyTxStatusProcessing(px_TxHash, fromChainId)
			if err != nil { //if errors.Is(err, optimistic.NewOptimisticError()) {
				fmt.Println("------- SetPolyTxStatusProcessing error -------" + err.Error())
			} else {
				fmt.Println("--------------- SetPolyTxStatusProcessing ok -------------")
			}

			px_StarcoinTxHash := "0x81cd5df1aff45149129cb21c93956c5e3308329cda1f23c74977d030d5e7d441"
			err = db.SetProcessingPolyTxStarcoinTxHash(px_TxHash, fromChainId, px_StarcoinTxHash)
			if err != nil {
				fmt.Println("------- SetProcessingPolyTxStarcoinTxHash error -------" + err.Error())
			} else {
				fmt.Println("--------------- SetProcessingPolyTxStarcoinTxHash ok -------------")
			}

			time.Sleep(time.Millisecond * 500)
			// time.Sleep(time.Second * time.Duration(rand.Intn(5)))
		}()
	}
	time.Sleep(time.Second * 8)
}

func TestConcatFromChainIdAndTxHash(t *testing.T) {
	var txHash []byte = []byte("hello world") // len(txHash) == 11
	r := concatFromChainIDAndTxHash(getTestFromChainId(), txHash)
	fmt.Println(r) // [218 0 0 0 0 0 0 0 11 104 101 108 108 111 32 119 111 114 108 100]
}

// func TestMisc(t *testing.T) {
// 	currentMillis := time.Now().UnixNano() / 1000000
// 	fmt.Println(currentMillis)
// 	fmt.Println(currentTimeMillis())
// }

// var (
// 	testDB DB
// )

// func init() {
// 	//db, err := NewBoltDB("test-boltdb")
// 	db, err := NewMySqlDB("root:123456@tcp(127.0.0.1:3306)/poly_starcoin?charset=utf8mb4&parseTime=True&loc=Local")
// 	if err != nil {
// 		fmt.Println(err)
// 		//t.FailNow()
// 	}
// 	testDB = db
// }

func devNetDB() DB {
	config := config.NewServiceConfig("../config-devnet.json")
	fmt.Println(config)
	db, err := NewMySqlDB(config.MySqlDSN)
	if err != nil {
		fmt.Println(err)
		panic(1)
	}
	return db
}

func testNetDB() DB {
	config := config.NewServiceConfig("../config-testnet.json")
	fmt.Println(config)
	db, err := NewMySqlDB(config.MySqlDSN)
	if err != nil {
		fmt.Println(err)
		panic(1)
	}
	return db
}

func getTestFromChainId() uint64 {
	return uint64(218)
}
