package db

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/celestiaorg/smt"
)

func TestDBSmtNodeMapStore(t *testing.T) {
	// Initialise two new key-value store to store the nodes and values of the tree
	//nodeStore := smt.NewSimpleMap()
	nodeStore := NewSmtNodeMapStore(testDB.(*MySqlDB))
	valueStore := smt.NewSimpleMap()
	smt := smt.NewSparseMerkleTree(nodeStore, valueStore, New256Hasher())
	var value []byte
	var has bool
	var err error

	// Test getting an empty key.
	value, err = smt.Get([]byte("testKey"))
	if err != nil {
		t.Errorf("returned error when getting empty key: %v", err)
	}
	if !bytes.Equal(SmtDefaultValue, value) {
		t.Error("did not get default value when getting empty key")
	}
	has, err = smt.Has([]byte("testKey"))
	if err != nil {
		t.Errorf("returned error when checking presence of empty key: %v", err)
	}
	if has {
		t.Error("did not get 'false' when checking presence of empty key")
	}

	// Test updating the empty key.
	_, err = smt.Update([]byte("testKey"), []byte("testValue"))
	if err != nil {
		t.Errorf("returned error when updating empty key: %v", err)
	}
	value, err = smt.Get([]byte("testKey"))
	if err != nil {
		t.Errorf("returned error when getting non-empty key: %v", err)
	}
	if !bytes.Equal([]byte("testValue"), value) {
		t.Error("did not get correct value when getting non-empty key")
	}
	has, err = smt.Has([]byte("testKey"))
	if err != nil {
		t.Errorf("returned error when checking presence of non-empty key: %v", err)
	}
	if !has {
		t.Error("did not get 'true' when checking presence of non-empty key")
	}

	// Test updating the non-empty key.
	_, err = smt.Update([]byte("testKey"), []byte("testValue2"))
	if err != nil {
		t.Errorf("returned error when updating non-empty key: %v", err)
	}
	value, err = smt.Get([]byte("testKey"))
	if err != nil {
		t.Errorf("returned error when getting non-empty key: %v", err)
	}
	if !bytes.Equal([]byte("testValue2"), value) {
		t.Error("did not get correct value when getting non-empty key")
	}

	// Test updating a second empty key where the path for both keys share the
	// first 2 bits (when using SHA256).
	_, err = smt.Update([]byte("foo"), []byte("testValue"))
	if err != nil {
		t.Errorf("returned error when updating empty second key: %v", err)
	}
	value, err = smt.Get([]byte("foo"))
	if err != nil {
		t.Errorf("returned error when getting non-empty second key: %v", err)
	}
	if !bytes.Equal([]byte("testValue"), value) {
		t.Error("did not get correct value when getting non-empty second key")
	}

	// Test updating a third empty key.
	_, err = smt.Update([]byte("testKey2"), []byte("testValue"))
	if err != nil {
		t.Errorf("returned error when updating empty third key: %v", err)
	}
	value, err = smt.Get([]byte("testKey2"))
	if err != nil {
		t.Errorf("returned error when getting non-empty third key: %v", err)
	}
	if !bytes.Equal([]byte("testValue"), value) {
		t.Error("did not get correct value when getting non-empty third key")
	}
	value, err = smt.Get([]byte("testKey"))
	if err != nil {
		t.Errorf("returned error when getting non-empty key: %v", err)
	}
	if !bytes.Equal([]byte("testValue2"), value) {
		t.Error("did not get correct value when getting non-empty key")
	}

	// // Test that a tree can be imported from a MapStore.
	// smt2 := ImportSparseMerkleTree(smn, smv, sha3.New256(), smt.Root())
	// value, err = smt2.Get([]byte("testKey"))
	// if err != nil {
	// 	t.Error("returned error when getting non-empty key")
	// }
	// if !bytes.Equal([]byte("testValue2"), value) {
	// 	t.Error("did not get correct value when getting non-empty key")
	// }
}

func TestPrintOneByteHash(t *testing.T) {
	fmt.Println(PolyTxExistsValueHashHex)
	// 2767f15c8af2f2c7225d5273fdd683edc714110a987d1054697c348aed4e6cc7
	// sha256:
	// 4bf5122f344554c53bde2ebb8cd2b7e3d1600ad631c385a5d7cce23c7785459a
}

func TestPrintLeafDataHash(t *testing.T) {
	h, _ := hex.DecodeString("0076d3bc41c9f588f7fcd0d5bf4718f8f84b1c41b20882703100b9eb9413807c012767f15c8af2f2c7225d5273fdd683edc714110a987d1054697c348aed4e6cc7")

	fmt.Println(Hash256Hex(h))
}

func TestDBMapStores(t *testing.T) {
	addTestPolyTx(testDB, "foo")
	addTestPolyTx(testDB, "testKey")
	addTestPolyTx(testDB, "testKey2")
	addTestPolyTx(testDB, "testKey3")
	addTestPolyTx(testDB, "testKey4")
	addTestPolyTx(testDB, "testKey5")
	addTestPolyTx(testDB, "testKey6")
	addTestPolyTx(testDB, "testKey7")
	addTestPolyTx(testDB, "testKey8")
	addTestPolyTx(testDB, "testKey9")
	key := "foo"
	var tx *PolyTx
	tx, err := testDB.GetPolyTx(key)
	if err != nil {
		t.FailNow()
	}
	proof, err := tx.GetNonMembershipProof()
	if err != nil {
		t.FailNow()
	}
	rootHash, err := hex.DecodeString(tx.SmtNonMembershipRootHash)
	if err != nil {
		t.FailNow()
	}
	// keyHash, err := hex.DecodeString(tx.TxHashHash)
	// if err != nil {
	// 	t.FailNow()
	// }
	v := smt.VerifyProof(*proof, rootHash, []byte(key), SmtDefaultValue, New256Hasher())
	if !v {
		t.FailNow()
	}

	//return

	// Initialise two new key-value store to store the nodes and values of the tree
	//nodeStore := smt.NewSimpleMap()
	nodeStore := NewSmtNodeMapStore(testDB.(*MySqlDB))
	valueStore := NewPolyTxMapStore(testDB.(*MySqlDB), nil)
	smt := smt.NewSparseMerkleTree(nodeStore, valueStore, New256Hasher())
	var value []byte
	var has bool
	//var err error

	// Test getting an empty key.
	value, err = smt.Get([]byte("testKey"))
	if err != nil {
		t.Errorf("returned error when getting empty key: %v", err)
	}
	if !bytes.Equal(SmtDefaultValue, value) {
		t.Error("did not get default value when getting empty key")
	}
	has, err = smt.Has([]byte("testKey"))
	if err != nil {
		t.Errorf("returned error when checking presence of empty key: %v", err)
	}
	if has {
		t.Error("did not get 'false' when checking presence of empty key")
	}

	// Test updating the empty key.
	_, err = smt.Update([]byte("testKey"), PolyTxExistsValue)
	if err != nil {
		t.Errorf("returned error when updating empty key: %v", err)
	}
	value, err = smt.Get([]byte("testKey"))
	if err != nil {
		t.Errorf("returned error when getting non-empty key: %v", err)
	}
	if !bytes.Equal(PolyTxExistsValue, value) {
		t.Error("did not get correct value when getting non-empty key")
	}
	has, err = smt.Has([]byte("testKey"))
	if err != nil {
		t.Errorf("returned error when checking presence of non-empty key: %v", err)
	}
	if !has {
		t.Error("did not get 'true' when checking presence of non-empty key")
	}

	// Test updating the non-empty key.
	_, err = smt.Update([]byte("testKey"), PolyTxExistsValue)
	if err != nil {
		t.Errorf("returned error when updating non-empty key: %v", err)
	}
	value, err = smt.Get([]byte("testKey"))
	if err != nil {
		t.Errorf("returned error when getting non-empty key: %v", err)
	}
	if !bytes.Equal(PolyTxExistsValue, value) {
		t.Error("did not get correct value when getting non-empty key")
	}

	// Test updating a second empty key where the path for both keys share the
	// first 2 bits (when using SHA256).
	_, err = smt.Update([]byte("foo"), PolyTxExistsValue)
	if err != nil {
		t.Errorf("returned error when updating empty second key: %v", err)
	}
	value, err = smt.Get([]byte("foo"))
	if err != nil {
		t.Errorf("returned error when getting non-empty second key: %v", err)
	}
	if !bytes.Equal(PolyTxExistsValue, value) {
		t.Error("did not get correct value when getting non-empty second key")
	}

	// Test updating a third empty key.
	_, err = smt.Update([]byte("testKey2"), PolyTxExistsValue)
	if err != nil {
		t.Errorf("returned error when updating empty third key: %v", err)
	}
	value, err = smt.Get([]byte("testKey2"))
	if err != nil {
		t.Errorf("returned error when getting non-empty third key: %v", err)
	}
	if !bytes.Equal(PolyTxExistsValue, value) {
		t.Error("did not get correct value when getting non-empty third key")
	}
	value, err = smt.Get([]byte("testKey"))
	if err != nil {
		t.Errorf("returned error when getting non-empty key: %v", err)
	}
	if !bytes.Equal(PolyTxExistsValue, value) {
		t.Error("did not get correct value when getting non-empty key")
	}

	// // Test that a tree can be imported from a MapStore.
	// smt2 := ImportSparseMerkleTree(smn, smv, sha3.New256(), smt.Root())
	// value, err = smt2.Get([]byte("testKey"))
	// if err != nil {
	// 	t.Error("returned error when getting non-empty key")
	// }
	// if !bytes.Equal([]byte("testValue2"), value) {
	// 	t.Error("did not get correct value when getting non-empty key")
	// }
}

func addTestPolyTx(db DB, key string) {
	polyTx, err := NewPolyTx([]byte(key), nil, nil, nil, nil, nil, key)
	if err != nil {
		panic(err)
	}
	_, err = db.PutPolyTx(polyTx)
	if err != nil {
		panic(err)
	}
}
