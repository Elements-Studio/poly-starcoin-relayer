package db

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/celestiaorg/smt"
	csmt "github.com/celestiaorg/smt"
	gomysql "github.com/go-sql-driver/mysql"
	"golang.org/x/crypto/sha3"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

type MySqlDB struct {
	//rwlock   *sync.RWMutex
	db *gorm.DB
}

func NewMySqlDB(dsn string) (*MySqlDB, error) {
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			SingularTable: true,
		},
	})
	if err != nil {
		return nil, err
	}
	// Migrate the schema
	db.AutoMigrate(&ChainHeight{}, &StarcoinTxRetry{}, &StarcoinTxCheck{})
	db.Set("gorm:table_options", "CHARSET=latin1").AutoMigrate(&PolyTx{}, &SmtNode{})

	w := new(MySqlDB)
	w.db = db
	return w, nil
}

func (w *MySqlDB) PutStarcoinTxCheck(txHash string, v []byte) error {
	tx := StarcoinTxCheck{
		TxHash: txHash,
		TxData: hex.EncodeToString(v),
	}
	return w.db.Create(tx).Error
}

func (w *MySqlDB) DeleteStarcoinTxCheck(txHash string) error {
	tx := StarcoinTxCheck{
		TxHash: txHash,
	}
	return w.db.Delete(tx).Error
}

// Put Starcoin cross-chain Tx.(to poly) Retry
func (w *MySqlDB) PutStarcoinTxRetry(k []byte) error {
	hash := sha3.Sum256(k)
	tx := StarcoinTxRetry{
		TxHash: hex.EncodeToString(hash[:]),
		TxData: hex.EncodeToString(k),
	}
	return w.db.Create(tx).Error
}

func (w *MySqlDB) DeleteStarcoinTxRetry(k []byte) error {
	hash := sha3.Sum256(k)
	tx := StarcoinTxRetry{
		TxHash: hex.EncodeToString(hash[:]),
	}
	return w.db.Delete(tx).Error
}

func (w *MySqlDB) GetAllStarcoinTxCheck() (map[string][]byte, error) {
	var list []StarcoinTxCheck
	if err := w.db.Find(&list).Error; err != nil {
		return nil, err
	}
	m := make(map[string][]byte, len(list))
	for _, v := range list {
		m[v.TxHash], _ = hex.DecodeString(v.TxData)
	}
	return m, nil
}

func (w *MySqlDB) GetAllStarcoinTxRetry() ([][]byte, error) {
	var list []StarcoinTxRetry
	if err := w.db.Find(&list).Error; err != nil {
		return nil, err
	}
	m := make([][]byte, 0, len(list))
	for _, v := range list {
		bs, _ := hex.DecodeString(v.TxData)
		m = append(m, bs)
	}
	return m, nil
}

// Update poly height synced to Starcoin
func (w *MySqlDB) UpdatePolyHeight(h uint32) error {
	ch := ChainHeight{
		Key:    KEY_POLY_HEIGHT,
		Height: h,
	}
	return createOrUpdate(w.db, ch)
}

func (w *MySqlDB) GetPolyHeight() (uint32, error) {
	ch := ChainHeight{}
	if err := w.db.Where(&ChainHeight{
		Key: KEY_POLY_HEIGHT,
	}).First(&ch).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, err
		} else {
			//fmt.Println("errors.Is(err, gorm.ErrRecordNotFound)")
			return 0, nil
		}
	}
	return ch.Height, nil
}

func (w *MySqlDB) GetPolyTx(txHash string) (*PolyTx, error) {
	px := PolyTx{}
	if err := w.db.Where(&PolyTx{
		TxHash: txHash,
	}).First(&px).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		} else {
			//fmt.Println("errors.Is(err, gorm.ErrRecordNotFound)")
			return nil, nil
		}
	}
	return &px, nil
}

func (w *MySqlDB) PutPolyTx(tx *PolyTx) (uint64, error) {
	lastTx := &PolyTx{}
	var lastIndex uint64
	err := w.db.Last(lastTx).Error
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, err
		} else {
			lastIndex = 0
			lastTx = nil
		}
	} else {
		lastIndex = lastTx.TxIndex
	}
	// tx := PolyTx{
	// 	TxIndex: lastIndex + 1,
	// 	TxHash:  txHash,
	// }
	tx.TxIndex = lastIndex + 1
	err = w.updatePolyTxNonMembershipProof(tx, lastTx)
	if err != nil {
		return 0, err
	}
	err = w.db.Create(tx).Error
	if err != nil {
		return 0, err
	}
	return tx.TxIndex, err
}

func (w *MySqlDB) updatePolyTxNonMembershipProof(tx *PolyTx, preTx *PolyTx) error {
	nodeStore := NewSmtNodeMapStore(w)
	valueStore := NewPolyTxMapStore(w, tx)
	var smt *csmt.SparseMerkleTree
	if preTx == nil {
		smt = csmt.NewSparseMerkleTree(nodeStore, valueStore, New256Hasher())
	} else {
		preRootHash, err := hex.DecodeString(preTx.SmtNonMembershipRootHash)
		if err != nil {
			return err
		}
		smt = csmt.ImportSparseMerkleTree(nodeStore, valueStore, New256Hasher(), preRootHash)
		_, err = smt.Update([]byte(preTx.TxHash), PolyTxExistsValue)
		if err != nil {
			return err
		}
	}
	tx.SmtNonMembershipRootHash = hex.EncodeToString(smt.Root()) //string `gorm:"size:66"`
	proof, err := smt.ProveUpdatable([]byte(tx.TxHash))
	if err != nil {
		return err
	}
	sns, err := EncodeSmtProofSideNodes(proof.SideNodes)
	if err != nil {
		return err
	}
	tx.SmtProofSideNodes = sns //string `gorm:"size:18000"`

	// NonMembershipLeafData is the data of the unrelated leaf at the position
	// of the key being proven, in the case of a non-membership proof. For
	// membership proofs, is nil.
	tx.SmtProofNonMembershipLeafData = hex.EncodeToString(proof.NonMembershipLeafData) //string `gorm:"size:132"`
	tx.SmtProofSiblingData = hex.EncodeToString(proof.SiblingData)
	return nil
}

func EncodeSmtProofSideNodes(sideNodes [][]byte) (string, error) {
	ss := make([]string, 0, len(sideNodes))
	for _, s := range sideNodes {
		ss = append(ss, hex.EncodeToString(s))
	}
	r, err := json.Marshal(ss)
	return string(r), err
}

func DecodeSmtProofSideNodes(s string) ([][]byte, error) {
	ss := &[]string{}
	err := json.Unmarshal([]byte(s), ss)
	if err != nil {
		return nil, err
	}
	bs := make([][]byte, 0, len(*ss))
	for _, v := range *ss {
		b, err := hex.DecodeString(v)
		if err != nil {
			return nil, err
		}
		bs = append(bs, b)
	}
	return bs, nil
}

func (w *MySqlDB) getPolyTxByTxHashHash(txHashHash string) (*PolyTx, error) {
	px := PolyTx{}
	if err := w.db.Where(&PolyTx{
		TxHashHash: txHashHash,
	}).First(&px).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		} else {
			//fmt.Println("errors.Is(err, gorm.ErrRecordNotFound)")
			return nil, nil
		}
	}
	return &px, nil
}

func (w *MySqlDB) Close() {
	//
}

func createOrUpdate(db *gorm.DB, dest interface{}) error {
	if err := db.Save(dest).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		} else {
			return db.Create(dest).Error
		}
	}
	return nil
}

var (
	PolyTxExistsValue        = []byte{1}
	PolyTxExistsValueHashHex = Hash256Hex(PolyTxExistsValue)
	SmtDefaultValue          = []byte{} //defaut(empty) value
)

type PolyTxMapStore struct {
	db            *MySqlDB
	currentPolyTx *PolyTx
}

func NewPolyTxMapStore(db *MySqlDB, currentTx *PolyTx) *PolyTxMapStore {
	return &PolyTxMapStore{
		db:            db,
		currentPolyTx: currentTx,
	}
}

func (m *PolyTxMapStore) Get(key []byte) ([]byte, error) { // Get gets the value for a key.
	h := hex.EncodeToString(key)
	if m.currentPolyTx != nil && strings.EqualFold(m.currentPolyTx.TxHashHash, h) {
		return PolyTxExistsValue, nil
	}
	polyTx, err := m.db.getPolyTxByTxHashHash(h)
	if err != nil {
		return nil, err
	}
	if polyTx != nil {
		return PolyTxExistsValue, nil
	}
	return nil, &smt.InvalidKeyError{Key: key}
}

func (m *PolyTxMapStore) Set(key []byte, value []byte) error { // Set updates the value for a key.
	if !bytes.Equal(PolyTxExistsValue, value) {
		return fmt.Errorf("invalid value error(must be [1])")
	}
	h := hex.EncodeToString(key)
	if m.currentPolyTx != nil && strings.EqualFold(m.currentPolyTx.TxHashHash, h) {
		return nil
	}
	_, err := m.Get(key)
	return err
}

func (m *PolyTxMapStore) Delete(key []byte) error { // Delete deletes a key.
	return fmt.Errorf("NOT IMPLEMENTED ERROR")
}

type SmtNodeMapStore struct {
	db *MySqlDB
}

func NewSmtNodeMapStore(db *MySqlDB) *SmtNodeMapStore {
	return &SmtNodeMapStore{
		db: db,
	}
}

func (m *SmtNodeMapStore) Get(key []byte) ([]byte, error) { // Get gets the value for a key.
	h := hex.EncodeToString(key)
	n := SmtNode{}
	if err := m.db.db.Where(&SmtNode{
		Hash: h,
	}).First(&n).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		} else {
			//fmt.Println("errors.Is(err, gorm.ErrRecordNotFound)")
			return nil, &smt.InvalidKeyError{Key: key}
		}
	}
	d, err := hex.DecodeString(n.Data)
	if err != nil {
		return nil, err
	}
	//fmtPrintlnNodeData(d)
	return d, nil
}

func (m *SmtNodeMapStore) Set(key []byte, value []byte) error { // Set updates the value for a key.
	h := hex.EncodeToString(key)
	d := hex.EncodeToString(value)
	n := SmtNode{
		Hash: h,
		Data: d,
	}
	err := m.db.db.Create(n).Error
	var mysqlErr *gomysql.MySQLError
	if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
		oldData, err := m.Get(key)
		if err != nil {
			return err
		}
		if bytes.Equal(value, oldData) {
			return nil
		} else {
			return fmt.Errorf("reset value is not allowed, key: %s, value: %s, old value: %s", h, d, hex.EncodeToString(oldData))
		}
	}
	return err
}

func (m *SmtNodeMapStore) Delete(key []byte) error { // Delete deletes a key.
	// h := hex.EncodeToString(key)
	// n := SmtNode{
	// 	Hash: h,
	// }
	// return m.db.db.Delete(n).Error
	return nil
}

func fmtPrintlnNodeData(d []byte) {
	r := hex.EncodeToString(d[33:65])
	if strings.EqualFold(r, PolyTxExistsValueHashHex) {
		r = "hashOf([]byte{1})"
	}
	fmt.Println("-------- parse node data --------")
	fmt.Printf("prefix: %s, left hash(or leaf path): %s, right hash(or value hash): %s\n",
		hex.EncodeToString(d[0:1]), hex.EncodeToString(d[1:33]), r)
}
