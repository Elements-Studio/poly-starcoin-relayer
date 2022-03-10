package db

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/starcoinorg/starcoin-go/client"
)

func TestCleanBoltDB(t *testing.T) {
	db := testNetBoltDB(t)
	testCleanBoltDBStarcoinTxRetry(db, t)
	testCleanBoltDBStarcoinTxCheck(db, t)
}

func TestBoltDBPutStarcoinTxRetry(t *testing.T) {
	db := testNetBoltDB(t)

	uuid, _ := uuid.NewUUID()
	k, _ := uuid.MarshalBinary()
	event := client.Event{
		BlockNumber: "1",
		Data:        hex.EncodeToString(k),
	}
	// test Put
	err := db.PutStarcoinTxRetry(k, event)
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}
	testCleanBoltDBStarcoinTxRetry(db, t)
}

func testCleanBoltDBStarcoinTxRetry(db *BoltDB, t *testing.T) {
	// test Get All
	bytesList, eventList, err := db.GetAllStarcoinTxRetry()
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}
	//fmt.Println(eventList)
	for _, event := range eventList {
		e, _ := json.Marshal(event)
		fmt.Println(string(e))
	}

	// test Delete
	for _, bytes := range bytesList {
		fmt.Println(bytes)
		fmt.Println(hex.EncodeToString(bytes))
		delErr := db.DeleteStarcoinTxRetry(bytes)
		if delErr != nil {
			fmt.Println(delErr)
			t.FailNow()
		}
	}
}

func TestBoltDBPutStarcoinCheck(t *testing.T) {
	db := testNetBoltDB(t)

	uuid, _ := uuid.NewUUID()
	v, _ := uuid.MarshalBinary()
	event := client.Event{
		BlockNumber: "1",
		Data:        hex.EncodeToString(v),
	}
	txHash := hex.EncodeToString(v)

	// test Put
	err := db.PutStarcoinTxCheck(txHash, v, event)
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}

	testCleanBoltDBStarcoinTxCheck(db, t)
}

func testCleanBoltDBStarcoinTxCheck(db *BoltDB, t *testing.T) {
	// test Get all
	checkMap, err := db.GetAllStarcoinTxCheck()
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}
	for k, be := range checkMap {
		fmt.Println(k)
		j, _ := json.Marshal(be)
		fmt.Println(string(j))
		fmt.Println(hex.EncodeToString(be.Bytes))

		// test Delete
		db.DeleteStarcoinTxCheck(k)
	}
}

func TestBoltDBUpdatePolyHeight(t *testing.T) {
	db := testNetBoltDB(t)
	db.UpdatePolyHeight(1)
	r, err := db.GetPolyHeight()
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}
	fmt.Println(r)
}

func testNetBoltDB(t *testing.T) *BoltDB {
	path := "../db-testnet"
	db, err := NewBoltDB(path)
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}
	return db
}
