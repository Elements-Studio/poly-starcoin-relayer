package db

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/celestiaorg/smt"
	rsmt "github.com/elements-studio/poly-starcoin-relayer/smt"
)

func TestDBSmtNodeMapStore(t *testing.T) {
	// Initialise two new key-value store to store the nodes and values of the tree
	//nodeStore := smt.NewSimpleMap()
	nodeStore := NewSmtNodeMapStore(testDB().(*MySqlDB))
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
	addTestPolyTx(testDB(), "foo")
	addTestPolyTx(testDB(), "testKey")
	addTestPolyTx(testDB(), "testKey2")
	addTestPolyTx(testDB(), "testKey3")
	addTestPolyTx(testDB(), "testKey4")
	addTestPolyTx(testDB(), "testKey5")
	addTestPolyTx(testDB(), "testKey6")
	addTestPolyTx(testDB(), "testKey7")
	addTestPolyTx(testDB(), "testKey8")
	addTestPolyTx(testDB(), "testKey9")
	if true {
		return
	}
	fromChainID := getTestFromChainId()
	key := "foo"
	var tx *PolyTx
	tx, err := testDB().GetPolyTx(key, fromChainID)
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
	// keyHash, err := hex.DecodeString(tx.SmtTxPath)
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
	nodeStore := NewSmtNodeMapStore(testDB().(*MySqlDB))
	valueStore := NewPolyTxMapStore(testDB().(*MySqlDB), nil)
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

	polyTx, err := NewPolyTx([]byte(key), getTestFromChainId(), nil, nil, nil, nil, nil, key)
	if err != nil {
		panic(err)
	}
	_, err = db.PutPolyTx(polyTx)
	if err != nil {
		panic(err)
	}
}

func TestUpdateRoot_1(t *testing.T) {
	path, _ := hex.DecodeString("8b4a296734b97f3c2028326c695f076e35de3183ada9d07cb7b9a32f1451d71f")
	value := PolyTxExistsValue
	sideNodes, err := DecodeSmtProofSideNodes(`
	["6f9bb267d56d0feecdd121f682df52b22d366fa7652975bec3ddabe457207eab"]
	`)
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}
	oldLeafData, _ := hex.DecodeString("0080be6638e99f15d7942bd0130b9118125010293dcc2054fdbf26bf997d0173f42767f15c8af2f2c7225d5273fdd683edc714110a987d1054697c348aed4e6cc7")

	r, err := rsmt.UpdateRoot(path, value, sideNodes, oldLeafData)
	if err != nil {
		t.FailNow()
	}
	fmt.Println(hex.EncodeToString(r)) //755e48a4526b0c5b3f7e26d00da398ffec97dc784777e16132681aa208b16be3
}

func TestUpdateRoot_2(t *testing.T) {
	path, _ := hex.DecodeString("c6281edc54637499646ddbd7e93636f91b8d3bb6974d7191452983fa6a015278") // hash of string "testKey3"
	value := PolyTxExistsValue
	sideNodes, err := DecodeSmtProofSideNodes(`
	["a18880b51b4475f45c663c66e9baff5bfdf01f9e552c9cfd84cfeb2494ea0bbd","da3c17cfd8be129f09b61272f8afcf42bf5b77cf7e405f5aa20c30684a205488"]
	`)
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}
	oldLeafData, _ := hex.DecodeString("00c0359bc303b37a066ce3a91aa14628accb3eb5dd6ed2c49c93f7bc60d29c797e2767f15c8af2f2c7225d5273fdd683edc714110a987d1054697c348aed4e6cc7")

	r, err := rsmt.UpdateRoot(path, value, sideNodes, oldLeafData)
	if err != nil {
		t.FailNow()
	}
	fmt.Println(hex.EncodeToString(r)) //7a379f33e0def9fe3555bc83b4f67f0b8ac23927352829603bff53c03fc58992
}
