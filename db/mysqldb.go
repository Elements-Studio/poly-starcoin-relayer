package db

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	csmt "github.com/celestiaorg/smt"
	//optimistic "github.com/crossoverJie/gorm-optimistic"
	gomysql "github.com/go-sql-driver/mysql"
	"github.com/starcoinorg/starcoin-go/client"
	"golang.org/x/crypto/sha3"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

var (
	PolyTxExistsValue                = []byte{1}
	PolyTxExistsValueHashHex         = Hash256Hex(PolyTxExistsValue)
	SmtDefaultValue                  = []byte{} //defaut(empty) value
	PolyTxMaxProcessingSeconds int64 = 120      // TODO: is this ok?
	PolyTxMaxRetryCount              = 10       // TODO: is this ok?
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
	db.AutoMigrate(&ChainHeight{})
	db.Set("gorm:table_options", "CHARSET=latin1").AutoMigrate(&PolyTx{}, &SmtNode{}, &StarcoinTxRetry{}, &StarcoinTxCheck{}, &PolyTxRetry{})

	w := new(MySqlDB)
	w.db = db
	return w, nil
}

// Put Starcoin(to poly) cross-chain Tx. Retry.
// The parameter `k` is bytes of serialized Tx. info,
// and `event` is attached Starcoin Event info.
func (w *MySqlDB) PutStarcoinTxRetry(k []byte, event client.Event) error {
	hash := sha3.Sum256(k)
	j, err := json.Marshal(event)
	if err != nil {
		return err
	}
	tx := StarcoinTxRetry{
		CrossTransferDataHash: hex.EncodeToString(hash[:]),
		CrossTransferData:     hex.EncodeToString(k),
		StarcoinEvent:         string(j),
	}
	return w.db.Create(tx).Error
}

// Delete Starcoin(to poly) cross-chain Tx. Retry.
// The parameter `k` is bytes of serialized Tx. info.
func (w *MySqlDB) DeleteStarcoinTxRetry(k []byte) error {
	hash := sha3.Sum256(k)
	tx := StarcoinTxRetry{
		CrossTransferDataHash: hex.EncodeToString(hash[:]),
	}
	return w.db.Delete(tx).Error
}

// Get all Starcoin(to poly) cross-chain Tx. Retry list.
// Return list of bytes(serialized Tx. info) and list of attached Starcoin Event.
func (w *MySqlDB) GetAllStarcoinTxRetry() ([][]byte, []client.Event, error) {
	var list []StarcoinTxRetry
	if err := w.db.Find(&list).Error; err != nil {
		return nil, nil, err
	}
	cs := make([][]byte, 0, len(list))
	es := make([]client.Event, 0, len(list))
	for _, v := range list {
		bs, err := hex.DecodeString(v.CrossTransferData)
		if err != nil {
			return nil, nil, err
		}
		cs = append(cs, bs)
		e := &client.Event{}
		err = json.Unmarshal([]byte(v.StarcoinEvent), e)
		if err != nil {
			return nil, nil, err
		}
		es = append(es, *e)
	}
	return cs, es, nil
}

// Put Starcoin(to poly) cross-chain Tx. check.
// The parameter `txHash` is poly Tx. hash,
// `v` is bytes of serialized cross-chain Tx. info,
// and `event` is attached Starcoin Event info.
func (w *MySqlDB) PutStarcoinTxCheck(txHash string, v []byte, event client.Event) error {
	j, err := json.Marshal(event)
	if err != nil {
		return err
	}
	tx := StarcoinTxCheck{
		TxHash:            txHash,
		CrossTransferData: hex.EncodeToString(v),
		StarcoinEvent:     string(j),
	}
	return w.db.Create(tx).Error
}

// Delete Starcoin(to poly) cross-chain Tx. check by (poly)Tx. Hash.
func (w *MySqlDB) DeleteStarcoinTxCheck(txHash string) error {
	tx := StarcoinTxCheck{
		TxHash: txHash,
	}
	return w.db.Delete(tx).Error
}

// Get Starcoin(to poly) cross-chain Tx. check list.
// Return mappings of (poly)Tx. Hash to  serialized cross-chain Tx. and attached Starcoin Event.
func (w *MySqlDB) GetAllStarcoinTxCheck() (map[string]BytesAndEvent, error) {
	var list []StarcoinTxCheck
	if err := w.db.Find(&list).Error; err != nil {
		return nil, err
	}
	m := make(map[string]BytesAndEvent, len(list))
	for _, v := range list {
		bs, _ := hex.DecodeString(v.CrossTransferData)
		e := &client.Event{}
		err := json.Unmarshal([]byte(v.StarcoinEvent), e)
		if err != nil {
			return nil, err
		}
		m[v.TxHash] = BytesAndEvent{
			Bytes: bs,
			Event: *e,
		}
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

func (w *MySqlDB) GetPolyTxRetry(txHash string, fromChainID uint64) (*PolyTxRetry, error) {
	r := PolyTxRetry{}
	if err := w.db.Where(&PolyTxRetry{
		TxHash:      txHash,
		FromChainID: fromChainID,
	}).First(&r).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		} else {
			//fmt.Println("errors.Is(err, gorm.ErrRecordNotFound)")
			return nil, nil
		}
	}
	return &r, nil
}

func (w *MySqlDB) GetAllPolyTxRetryNotPaid() ([]*PolyTxRetry, error) {
	var list []*PolyTxRetry
	if err := w.db.Where(&PolyTxRetry{
		FeeStatus: FEE_STATUS_NOT_PAID,
	}).Find(&list).Error; err != nil {
		return nil, err
	}
	m := make([]*PolyTxRetry, 0, len(list))
	for _, v := range list {
		m = append(m, v)
	}
	return m, nil
}

func (w *MySqlDB) DeletePolyTxRetry(txHash string, fromChainID uint64) error {
	tx := &PolyTxRetry{
		TxHash:      txHash,
		FromChainID: fromChainID,
	}
	return w.db.Delete(tx).Error
}

func (w *MySqlDB) PutPolyTxRetry(tx *PolyTxRetry) error {
	err := w.db.Create(tx).Error
	if err != nil {
		return err
	}
	return nil
}

func (w *MySqlDB) IncreasePolyTxRetryCheckFeeCount(txHash string, fromChainID uint64, oldCount int) error {
	px := PolyTxRetry{}
	if err := w.db.Where(&PolyTxRetry{
		TxHash:        txHash,
		FromChainID:   fromChainID,
		CheckFeeCount: oldCount,
	}).First(&px).Error; err != nil {
		return err
	}
	px.CheckFeeCount = px.CheckFeeCount + 1
	px.UpdatedAt = CurrentTimeMillis() // UpdateWithOptimistic need this!
	return w.db.Save(px).Error
	// // use optimistic lock here
	// return optimistic.UpdateWithOptimistic(w.db, &px, func(model optimistic.Lock) optimistic.Lock {
	// 	return model
	// }, 1, 1)
}

func (w *MySqlDB) SetPolyTxRetryFeeStatus(txHash string, fromChainID uint64, status string) error {
	px := PolyTxRetry{}
	if err := w.db.Where(&PolyTxRetry{
		TxHash:      txHash,
		FromChainID: fromChainID,
	}).First(&px).Error; err != nil {
		return err
	}
	px.FeeStatus = status
	//px.UpdatedAt = currentTimeMillis()
	return w.db.Save(px).Error
}

func (w *MySqlDB) GetPolyTx(txHash string, fromChainID uint64) (*PolyTx, error) {
	px := PolyTx{}
	if err := w.db.Where(&PolyTx{
		TxHash:      txHash,
		FromChainID: fromChainID,
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

func (w *MySqlDB) GetPolyTxByIndex(idx uint64) (*PolyTx, error) {
	px := PolyTx{}
	if err := w.db.Where(&PolyTx{
		TxIndex: idx,
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

func (w *MySqlDB) SetPolyTxStatus(txHash string, fromChainID uint64, status string) error {
	px := PolyTx{}
	if err := w.db.Where(&PolyTx{
		TxHash:      txHash,
		FromChainID: fromChainID,
	}).First(&px).Error; err != nil {
		return err
	}
	px.Status = status
	//px.UpdatedAt = currentTimeMillis()
	return w.db.Save(px).Error
}

func (w *MySqlDB) SetPolyTxStatusProcessing(txHash string, fromChainID uint64) error {
	px := PolyTx{}
	if err := w.db.Where(&PolyTx{
		TxHash:      txHash,
		FromChainID: fromChainID,
	}).First(&px).Error; err != nil {
		return err
	}
	if px.Status == STATUS_CONFIRMED || px.Status == STATUS_PROCESSED {
		return fmt.Errorf("PolyTx status is already '%s', TxHash: %s, FromChainID: %d", px.Status, px.TxHash, px.FromChainID)
	}
	if px.Status == STATUS_PROCESSING {
		// when re-process, set StarcoinTxHash to empty first, then send new Starcoin transaction and set new hash
		if px.StarcoinTxHash == "" {
			if !(px.UpdatedAt < CurrentTimeMillis()-PolyTxMaxProcessingSeconds*1000) {
				return fmt.Errorf("PolyTx.StarcoinTxHash is already empty, TxHash: %s, FromChainID: %d", px.TxHash, px.FromChainID)
			}
		}
	}
	px.Status = STATUS_PROCESSING
	px.StarcoinTxHash = ""
	//fmt.Println("px.StarcoinTxHash = " + px.StarcoinTxHash)
	px.RetryCount = px.RetryCount + 1
	px.UpdatedAt = CurrentTimeMillis() // UpdateWithOptimistic need this!
	return w.db.Save(px).Error
	// // use optimistic lock here
	// return optimistic.UpdateWithOptimistic(w.db, &px, func(model optimistic.Lock) optimistic.Lock {
	// 	return model
	// }, 1, 1)
}

func (w *MySqlDB) SetProcessingPolyTxStarcoinTxHash(txHash string, fromChainID uint64, starcoinTxHash string) error {
	px := PolyTx{}
	if err := w.db.Where(&PolyTx{
		TxHash:      txHash,
		FromChainID: fromChainID,
	}).First(&px).Error; err != nil {
		return err
	}
	if starcoinTxHash == "" {
		return fmt.Errorf("Try to set empty starcoinTxHash. PolyTx status is '%s', TxHash: %s, FromChainID: %d", px.Status, px.TxHash, px.FromChainID)
	}
	if px.Status == STATUS_CONFIRMED || px.Status == STATUS_PROCESSED {
		return fmt.Errorf("PolyTx status is already '%s', TxHash: %s, FromChainID: %d", px.Status, px.TxHash, px.FromChainID)
	}
	// if px.Status != STATUS_PROCESSING {
	// 	return fmt.Errorf("PolyTx status is not PROCESSING. Status: '%s', TxHash: %s, FromChainID: %d", px.Status, px.TxHash, px.FromChainID)
	// }
	//if px.Status == STATUS_PROCESSING {

	// // when re-process, set StarcoinTxHash to empty first, then send new Starcoin transaction and set new hash
	// if px.StarcoinTxHash != "" {
	// 	//if !(px.UpdatedAt < currentTimeMillis()-PolyTxMaxProcessingSeconds*1000) {
	// 	return fmt.Errorf("PolyTx.StarcoinTxHash is already not empty, TxHash: %s, FromChainID: %d", px.TxHash, px.FromChainID)
	// 	//}
	// }
	px.Status = STATUS_PROCESSING
	px.StarcoinTxHash = starcoinTxHash
	px.UpdatedAt = CurrentTimeMillis() // UpdateWithOptimistic need this!
	return w.db.Save(px).Error
	// // use optimistic lock here
	// return optimistic.UpdateWithOptimistic(w.db, &px, func(model optimistic.Lock) optimistic.Lock {
	// 	return model
	// }, 1, 1)
}

func (w *MySqlDB) SetPolyTxStatusProcessed(txHash string, fromChainID uint64, starcoinTxHash string) error {
	px := PolyTx{}
	if err := w.db.Where(&PolyTx{
		TxHash:      txHash,
		FromChainID: fromChainID,
	}).First(&px).Error; err != nil {
		return err
	}
	px.Status = STATUS_PROCESSED
	px.StarcoinTxHash = starcoinTxHash
	//px.UpdatedAt = currentTimeMillis()
	go func() {
		w.updatePolyTransactionsToProcessedBeforeIndex(px.TxIndex)
	}()
	return w.db.Save(px).Error
}

func (w *MySqlDB) GetFirstFailedPolyTx() (*PolyTx, error) {
	var list []PolyTx
	//err := w.db.Where("updated_at < ?", currentTimeMillis()-PolyTxMaxProcessingSeconds*1000).Not(map[string]interface{}{"status": []string{STATUS_PROCESSED, STATUS_CONFIRMED}}).Limit(1).Find(&list).Error
	notFailedStatuses := []string{STATUS_PROCESSED, STATUS_CONFIRMED, STATUS_TIMEDOUT}
	err := w.db.Where("retry_count < ?", PolyTxMaxRetryCount).Not(map[string]interface{}{"status": notFailedStatuses}).Limit(1).Find(&list).Error
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		} else {
			return nil, nil
		}
	}
	if len(list) == 0 {
		return nil, nil
	}
	first := list[0]
	if first.UpdatedAt < CurrentTimeMillis()-PolyTxMaxProcessingSeconds*1000 {
		return &first, nil
	} else {
		return nil, nil
	}
}

func (w *MySqlDB) GetFirstTimedOutPolyTx() (*PolyTx, error) {
	var list []PolyTx
	timedOutStatuses := []string{STATUS_TIMEDOUT}
	err := w.db.Where(map[string]interface{}{"status": timedOutStatuses}).Limit(1).Find(&list).Error
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		} else {
			return nil, nil
		}
	}
	if len(list) == 0 {
		return nil, nil
	}
	first := list[0]
	if first.UpdatedAt < CurrentTimeMillis()-PolyTxMaxProcessingSeconds*1000 {
		return &first, nil
	} else {
		return nil, nil
	}
}

func (w *MySqlDB) GetTimedOutOrFailedPolyTxList() ([]*PolyTx, error) {
	lastIndex, _, err := w.getLastPolyTx()
	if err != nil {
		return nil, err
	}
	var list []*PolyTx
	indexDiffLimit := uint64(50) // TODO: is this ok??
	indexAfter := uint64(1)
	if lastIndex > indexDiffLimit {
		indexAfter = lastIndex - indexDiffLimit
	}
	limit := 10
	statuses := []string{STATUS_TIMEDOUT, STATUS_FAILED}
	err = w.db.Where("tx_index <= ? and tx_index >= ? and status IN ?", lastIndex, indexAfter, statuses).Limit(limit).Find(&list).Error
	if err != nil {
		return nil, err
	}
	return list, nil
}

func (w *MySqlDB) updatePolyTransactionsToProcessedBeforeIndex(index uint64) error {
	var list []PolyTx
	indexDiffLimit := uint64(50) // TODO: is this ok??
	indexAfter := uint64(1)
	if index > indexDiffLimit {
		indexAfter = index - indexDiffLimit
	}
	limit := 10
	toBeProcessedStatuses := []string{STATUS_PROCESSING, STATUS_FAILED, STATUS_TIMEDOUT}
	err := w.db.Where("tx_index < ? and tx_index >= ? and status IN ?", index, indexAfter, toBeProcessedStatuses).Limit(limit).Find(&list).Error
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		} else {
			return nil
		}
	}
	if len(list) == 0 {
		return nil
	}
	for _, v := range list {
		v.Status = STATUS_PROCESSED
		err := w.db.Save(v).Error
		_ = err //ignore error?
	}
	return nil
}

func (w *MySqlDB) PutPolyTx(tx *PolyTx) (uint64, error) {
	lastIndex, lastTx, err := w.getLastPolyTx()
	if err != nil {
		return 0, err
	}
	// tx := PolyTx{
	// 	TxIndex: lastIndex + 1,
	// 	TxHash:  txHash,
	// }
	tx.TxIndex = lastIndex + 1
	err = w.setPolyTxNonMembershipProof(tx, lastTx)
	if err != nil {
		return 0, err
	}

	//tx.UpdatedAt = currentTimeMillis()
	tx.Status = STATUS_CREATED

	err = w.db.Create(tx).Error

	if err != nil {
		return 0, err
	}
	return tx.TxIndex, nil
}

func (w *MySqlDB) getLastPolyTx() (uint64, *PolyTx, error) {
	lastTx := &PolyTx{}
	var lastIndex uint64
	err := w.db.Last(lastTx).Error
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) { // error only once
			return 0, nil, err
		} else {
			lastIndex = 0
			lastTx = nil
		}
	} else {
		lastIndex = lastTx.TxIndex
	}
	return lastIndex, lastTx, nil
}

func IsDuplicatePolyTxError(db DB, tx *PolyTx, err error) (bool, error) {
	var mysqlErr *gomysql.MySQLError
	if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 { // Duplicate entry error
		oldData, getErr := db.GetPolyTx(tx.TxHash, tx.FromChainID)
		if getErr != nil {
			return false, getErr
		}
		if bytes.Equal([]byte(tx.PolyTxProof), []byte(oldData.PolyTxProof)) {
			return true, nil
		} else {
			return false, err
		}
	} else {
		return false, err
	}
}

func IsDuplicatePolyTxRetryError(db DB, r *PolyTxRetry, err error) (bool, error) {
	var mysqlErr *gomysql.MySQLError
	if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 { // Duplicate entry error
		oldData, getErr := db.GetPolyTxRetry(r.TxHash, r.FromChainID)
		if getErr != nil {
			return false, getErr
		}
		if bytes.Equal([]byte(r.BridgeTransaction), []byte(oldData.BridgeTransaction)) {
			return true, nil
		} else {
			return false, err
		}
	} else {
		return false, err
	}
}

// calculate SMT root hash after current transaction included.
func (w *MySqlDB) computePloyTxInclusionRootHash(tx *PolyTx) ([]byte, error) {
	nodeStore := NewSmtNodeMapStore(w)
	valueStore := NewPolyTxMapStore(w, tx)
	nonMemberRootHash, err := hex.DecodeString(tx.SmtNonMembershipRootHash)
	if err != nil {
		return nil, err
	}
	smTree := csmt.ImportSparseMerkleTree(nodeStore, valueStore, New256Hasher(), nonMemberRootHash)
	k, err := tx.GetSmtTxKey()
	if err != nil {
		return nil, err
	}
	newRootHash, err := smTree.Update(k, PolyTxExistsValue)
	if err != nil {
		return nil, err
	}
	return newRootHash, nil
}

func (w *MySqlDB) UpdatePolyTxNonMembershipProofByIndex(idx uint64) error {
	tx, err := w.GetPolyTxByIndex(idx)
	if tx == nil || err != nil {
		return fmt.Errorf("cannot get PolyTx by index: %d", idx)
	}
	preTx, err := w.GetPolyTxByIndex(idx - 1)
	if err != nil {
		return fmt.Errorf("cannot get PolyTx by index: %d", idx-1)
	}
	err = w.setPolyTxNonMembershipProof(tx, preTx)
	if err != nil {
		return fmt.Errorf("updatePolyTxNonMembershipProof error: %s", err.Error())
	}
	return w.db.Save(tx).Error
}

// set PolyTx non-membership proof info.
func (w *MySqlDB) setPolyTxNonMembershipProof(tx *PolyTx, preTx *PolyTx) error {
	nodeStore := NewSmtNodeMapStore(w)
	valueStore := NewPolyTxMapStore(w, tx)
	var smTree *csmt.SparseMerkleTree
	if preTx == nil {
		smTree = csmt.NewSparseMerkleTree(nodeStore, valueStore, New256Hasher())
	} else {
		preRootHash, err := hex.DecodeString(preTx.SmtNonMembershipRootHash)
		if err != nil {
			return err
		}
		smTree = csmt.ImportSparseMerkleTree(nodeStore, valueStore, New256Hasher(), preRootHash)
		preTxKey, err := preTx.GetSmtTxKey()
		if err != nil {
			return err
		}
		_, err = smTree.Update(preTxKey, PolyTxExistsValue)
		if err != nil {
			return err
		}
	}
	tx.SmtNonMembershipRootHash = hex.EncodeToString(smTree.Root()) //string `gorm:"size:66"`
	k, err := tx.GetSmtTxKey()
	if err != nil {
		return err
	}
	proof, err := smTree.ProveUpdatable(k)
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

func (w *MySqlDB) getPolyTxBySmtTxPath(path string) (*PolyTx, error) {
	px := PolyTx{}
	if err := w.db.Where(&PolyTx{
		SmtTxPath: path,
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
	path := hex.EncodeToString(key)
	// fmt.Println("------------------- *PolyTxMapStore.Get -------------------")
	// fmt.Println(m.currentPolyTx)
	// fmt.Println(h)
	// fmt.Println("------------------- *PolyTxMapStore.Get -------------------")
	if m.currentPolyTx != nil && strings.EqualFold(m.currentPolyTx.SmtTxPath, path) {
		return PolyTxExistsValue, nil
	}
	polyTx, err := m.db.getPolyTxBySmtTxPath(path)
	if err != nil {
		return nil, err
	}
	if polyTx != nil {
		return PolyTxExistsValue, nil
	}
	return nil, &csmt.InvalidKeyError{Key: key}
}

func (m *PolyTxMapStore) Set(key []byte, value []byte) error { // Set updates the value for a key.
	if !bytes.Equal(PolyTxExistsValue, value) {
		return fmt.Errorf("invalid value error(must be [1])")
	}
	path := hex.EncodeToString(key)
	if m.currentPolyTx != nil && strings.EqualFold(m.currentPolyTx.SmtTxPath, path) {
		return nil
	}
	_, err := m.Get(key) // It must already exists.
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
			return nil, &csmt.InvalidKeyError{Key: key}
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
	if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 { // Duplicate entry error
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

// func fmtPrintlnNodeData(d []byte) {
// 	r := hex.EncodeToString(d[33:65])
// 	if strings.EqualFold(r, PolyTxExistsValueHashHex) {
// 		r = "hashOf([]byte{1})"
// 	}
// 	fmt.Println("-------- parse node data --------")
// 	fmt.Printf("prefix: %s, left hash(or leaf path): %s, right hash(or value hash): %s\n",
// 		hex.EncodeToString(d[0:1]), hex.EncodeToString(d[1:33]), r)
// }
