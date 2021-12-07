package db

import (
	"bytes"
	"crypto/sha256"
	"testing"

	"github.com/celestiaorg/smt"
)

func TestDBSmtNodeMapStore(t *testing.T) {
	// Initialise two new key-value store to store the nodes and values of the tree
	//nodeStore := smt.NewSimpleMap()
	nodeStore := NewSmtNodeMapStore(testDB.(*MySqlDB))
	valueStore := smt.NewSimpleMap()
	smt := smt.NewSparseMerkleTree(nodeStore, valueStore, sha256.New())
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
	// smt2 := ImportSparseMerkleTree(smn, smv, sha256.New(), smt.Root())
	// value, err = smt2.Get([]byte("testKey"))
	// if err != nil {
	// 	t.Error("returned error when getting non-empty key")
	// }
	// if !bytes.Equal([]byte("testValue2"), value) {
	// 	t.Error("did not get correct value when getting non-empty key")
	// }
}

func TestDBMapStores(t *testing.T) {
	addTestPolyTx(testDB, "testKey")
	addTestPolyTx(testDB, "foo")
	addTestPolyTx(testDB, "testKey2")

	// Initialise two new key-value store to store the nodes and values of the tree
	//nodeStore := smt.NewSimpleMap()
	nodeStore := NewSmtNodeMapStore(testDB.(*MySqlDB))
	valueStore := NewPolyTxMapStore(testDB.(*MySqlDB))
	smt := smt.NewSparseMerkleTree(nodeStore, valueStore, sha256.New())
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
	// smt2 := ImportSparseMerkleTree(smn, smv, sha256.New(), smt.Root())
	// value, err = smt2.Get([]byte("testKey"))
	// if err != nil {
	// 	t.Error("returned error when getting non-empty key")
	// }
	// if !bytes.Equal([]byte("testValue2"), value) {
	// 	t.Error("did not get correct value when getting non-empty key")
	// }
}

func addTestPolyTx(db DB, key string) {
	polyTx, err := NewPolyTx(key, nil, nil, nil, nil, nil)
	if err != nil {
		panic(err)
	}
	db.PutPolyTx(polyTx)
}
