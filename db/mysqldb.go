package db

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	csmt "github.com/celestiaorg/smt"
	"github.com/elements-studio/poly-starcoin-relayer/tools"
	gomysql "github.com/go-sql-driver/mysql"
	"github.com/starcoinorg/starcoin-go/client"
	"golang.org/x/crypto/sha3"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
	//optimistic "github.com/crossoverJie/gorm-optimistic"
)

var (
	PolyTxExistsValue                      = []byte{1}
	PolyTxExistsValueHashHex               = Hash256Hex(PolyTxExistsValue)
	SmtDefaultValue                        = []byte{} // SMT defaut(empty) value
	PolyTxMaxProcessingSeconds       int64 = 120      // Poly(to Starcoin) Transaction max processing time in seconds
	PolyTxMaxRetryCount                    = 10       // Poly(to Starcoin) Transaction max retry count
	GasSubsidyTxMaxProcessingSeconds int64 = 120      // Gas subsidy (Starcoin)Transaction max processing time in seconds
	GasSubsidyTxMaxRetryCount              = 10       // Gas subsidy (Starcoin)Transaction max retry count
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
	db.Set("gorm:table_options", "CHARSET=latin1").AutoMigrate(&PolyTx{}, &SmtNode{}, &StarcoinTxRetry{}, &StarcoinTxCheck{}, &PolyTxRetry{}, &RemovedPolyTx{}, &GasSubsidy{})

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
	tx := &StarcoinTxRetry{
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
	tx := &StarcoinTxCheck{
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
	ch := &ChainHeight{
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

func (w *MySqlDB) GetAllPolyTxRetry() ([]*PolyTxRetry, error) {
	var list []*PolyTxRetry
	if err := w.db.Where(&PolyTxRetry{ //TODO: Maybe ignore some statuses...
		//FeeStatus: FEE_STATUS_NOT_PAID,
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
	return w.db.Save(&px).Error
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
	return w.db.Save(&px).Error
}

func (w *MySqlDB) UpdatePolyTxStarcoinStatus(txHash string, fromChainID uint64, status string, msg string) error {
	px := PolyTxRetry{}
	if err := w.db.Where(&PolyTxRetry{
		TxHash:      txHash,
		FromChainID: fromChainID,
	}).First(&px).Error; err != nil {
		return err
	}
	px.StarcoinStatus = status
	px.CheckStarcoinCount = px.CheckStarcoinCount + 1
	px.CheckStarcoinMessage = msg
	//px.UpdatedAt = currentTimeMillis()
	return w.db.Save(&px).Error
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

func (w *MySqlDB) GetPreviousePolyTx(idx uint64) (*PolyTx, error) {
	var list []PolyTx
	err := w.db.Where("tx_index < ?", idx).Order("tx_index desc").Limit(1).Find(&list).Error
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
	return &first, nil
}

func (w *MySqlDB) SetPolyTxStatus(txHash string, fromChainID uint64, oldStatus string, status string) error {
	px := PolyTx{}
	if err := w.db.Where(&PolyTx{
		TxHash:      txHash,
		FromChainID: fromChainID,
	}).First(&px).Error; err != nil {
		return err
	}
	// px.Status = status
	// //px.UpdatedAt = currentTimeMillis()
	// return w.db.Save(&px).Error
	fieldMap := map[string]interface{}{
		"status":     status,
		"updated_at": CurrentTimeMillis(),
	}
	return w.optimisticUpdatePolyTx(&px, oldStatus, fieldMap)
}

// Set PolyTx status to PROCESSING(and StarcoinTxHash to empty).
// When re-process a PolyTx, set it's StarcoinTxHash to empty first, then send new Starcoin transaction and set new StarcoinTxHash ot the PolyTx.
func (w *MySqlDB) SetPolyTxStatusProcessing(txHash string, fromChainID uint64, oldStatus string) error {
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
		if px.StarcoinTxHash == "" {
			// Only timed-out PROCESSING PolyTx can be re-process
			if !(px.UpdatedAt < CurrentTimeMillis()-PolyTxMaxProcessingSeconds*1000) {
				return fmt.Errorf("PolyTx.StarcoinTxHash is already empty, TxHash: %s, FromChainID: %d", px.TxHash, px.FromChainID)
			}
		}
	}
	// px.Status = STATUS_PROCESSING
	// px.StarcoinTxHash = ""
	// px.RetryCount = px.RetryCount + 1
	// px.UpdatedAt = CurrentTimeMillis() // UpdateWithOptimistic need this!
	// return w.db.Save(&px).Error
	fieldMap := map[string]interface{}{
		"starcoin_tx_hash": "",
		"retry_count":      px.RetryCount + 1,
		"status":           STATUS_PROCESSING,
		"updated_at":       CurrentTimeMillis(),
	}
	return w.optimisticUpdatePolyTx(&px, oldStatus, fieldMap)
}

func (w *MySqlDB) SetProcessingPolyTxStarcoinTxHash(txHash string, fromChainID uint64, starcoinTxHash string) error {
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
	if starcoinTxHash == "" {
		return fmt.Errorf("Try to set empty starcoinTxHash. PolyTx status is '%s', TxHash: %s, FromChainID: %d", px.Status, px.TxHash, px.FromChainID)
	}
	// /////////// update //////////////
	// px.Status = STATUS_PROCESSING
	// px.StarcoinTxHash = starcoinTxHash
	// px.UpdatedAt = CurrentTimeMillis() // UpdateWithOptimistic need this!
	// return w.db.Save(&px).Error
	fieldMap := map[string]interface{}{
		"starcoin_tx_hash": starcoinTxHash,
		"status":           STATUS_PROCESSING,
		"updated_at":       CurrentTimeMillis(),
	}
	return w.optimisticUpdatePolyTx(&px, STATUS_PROCESSING, fieldMap)
}

func (w *MySqlDB) SetPolyTxStatusProcessed(txHash string, fromChainID uint64, oldStatus string, starcoinTxHash string) error {
	px := PolyTx{}
	if err := w.db.Where(&PolyTx{
		TxHash:      txHash,
		FromChainID: fromChainID,
	}).First(&px).Error; err != nil {
		return err
	}
	// ////////// optimistic update ///////////
	// px.Status = STATUS_PROCESSED
	// px.StarcoinTxHash = starcoinTxHash
	fieldMap := map[string]interface{}{
		"starcoin_tx_hash": starcoinTxHash,
		"status":           STATUS_PROCESSED,
		"updated_at":       CurrentTimeMillis(),
	}
	err := w.optimisticUpdatePolyTx(&px, oldStatus, fieldMap)
	// //////////// async update /////////////
	go func() {
		w.updatePolyTransactionsToProcessedBeforeIndex(px.TxIndex)
	}()
	// ///////////////////////////////////////
	return err
}

func (w MySqlDB) optimisticUpdatePolyTx(px *PolyTx, oldStatus string, fieldMap map[string]interface{}) error {
	sql := "from_chain_id = ? and tx_hash = ? and status = ?"
	if px.StarcoinTxHash == "" {
		sql = sql + " and (starcoin_tx_hash is null or starcoin_tx_hash = '')"
	} else {
		sql = sql + " and starcoin_tx_hash = '" + strings.Replace(px.StarcoinTxHash, "'", "", -1) + "'"
	}
	if px.UpdatedAt > 0 {
		sql = sql + " and updated_at = " + strconv.FormatInt(px.UpdatedAt, 10) + " "
	}
	db := w.db.Table("poly_tx").Where(
		sql,
		px.FromChainID, px.TxHash, oldStatus,
	).Updates(fieldMap)
	if db.Error != nil {
		return db.Error
	}
	rowsAffected := db.RowsAffected
	if rowsAffected == 0 {
		return fmt.Errorf("optimistic lock error. TxIndex: %d, fromChainId: %d, txHash: %s", px.TxIndex, px.FromChainID, px.TxHash)
	}
	return nil
}

func (w *MySqlDB) GetFirstFailedPolyTx() (*PolyTx, error) {
	var list []PolyTx
	//err := w.db.Where("updated_at < ?", currentTimeMillis()-PolyTxMaxProcessingSeconds*1000).Not(map[string]interface{}{"status": []string{STATUS_PROCESSED, STATUS_CONFIRMED}}).Limit(1).Find(&list).Error
	notFailedStatuses := []string{STATUS_PROCESSED, STATUS_CONFIRMED, STATUS_TIMEDOUT, STATUS_TO_BE_REMOVED}
	// Ignore PolyTx failed to many times...
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

func (w *MySqlDB) GetFirstPolyTxToBeRemoved() (*PolyTx, error) {
	var list []PolyTx
	status := []string{STATUS_TO_BE_REMOVED}
	err := w.db.Where(map[string]interface{}{"status": status}).Limit(1).Find(&list).Error
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
	//if first.UpdatedAt < CurrentTimeMillis()-PolyTxMaxProcessingSeconds*1000 {
	return &first, nil
	//} else {
	//	return nil, nil
	//}
}

func (w *MySqlDB) GetFirstRemovedPolyTxToBePushedBack() (*RemovedPolyTx, error) {
	var list []RemovedPolyTx
	status := []string{STATTUS_TO_BE_PUSHED_BACK}
	err := w.db.Where(map[string]interface{}{"status": status}).Limit(1).Find(&list).Error
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
	return &first, nil
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
		err := w.db.Save(&v).Error
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
	if lastTx != nil && lastTx.SmtNonMembershipRootHash == "" {
		// If previouse SmtNonMembershipRootHash is empty, ignore set current proof
	} else {
		err = w.setPolyTxNonMembershipProof(tx, lastTx)
		if err != nil {
			return 0, err
		}
	}

	//tx.UpdatedAt = currentTimeMillis()
	tx.Status = STATUS_CREATED

	err = w.db.Create(tx).Error

	if err != nil {
		return 0, err
	}
	return tx.TxIndex, nil
}

func (w *MySqlDB) RemovePolyTx(tx *PolyTx) error {
	if tx.Status != STATUS_TO_BE_REMOVED {
		return fmt.Errorf("PolyTx(index: '%d') status is not TO_BE_REMOVED, it is: %s", tx.TxIndex, tx.Status)
	}
	r := tx.ToRemovedPolyTx()

	err := w.db.Transaction(func(dbtx *gorm.DB) error {
		dbErr := dbtx.Save(r).Error
		if dbErr != nil {
			return dbErr
		}
		dbErr = dbtx.Delete(tx).Error
		if dbErr != nil {
			return dbErr
		}
		// reset non-membership proof after removed PolyTx
		dbErr = dbtx.Table("poly_tx").Where("tx_index > ?", tx.TxIndex).Updates(map[string]interface{}{"smt_non_membership_root_hash": ""}).Error
		if dbErr != nil {
			return dbErr
		}
		return nil
	})
	return err
}

func (w *MySqlDB) PushBackRemovePolyTx(id uint64) error {
	px := RemovedPolyTx{}
	if err := w.db.Where(&RemovedPolyTx{
		ID: id,
	}).First(&px).Error; err != nil {
		return err
	}
	p := px.ToPolyTx()

	err := w.db.Transaction(func(dbtx *gorm.DB) error {
		idx, putErr := w.PutPolyTx(p)
		_ = idx
		if putErr != nil {
			return putErr
		}
		delErr := dbtx.Delete(px).Error
		if delErr != nil {
			return delErr
		}
		return nil
	})

	return err
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
	preTx, err := w.GetPreviousePolyTx(idx) //GetPolyTxByIndex(idx - 1)
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
// preTx can be nil.
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

func (w *MySqlDB) GetPolyTxListNotHaveGasSubsidy(chainId uint64, updatedAfter int64) ([]*PolyTx, error) {
	var list []*PolyTx
	sql := fmt.Sprintf(`SELECT 
    p.*
FROM
    poly_tx p
        LEFT JOIN
    gas_subsidy s ON p.from_chain_id = s.from_chain_id
        AND p.tx_hash = s.tx_hash
WHERE
    (p.status = '%s' OR p.status = '%s')
        AND p.from_chain_id = ?
        AND p.updated_at > ?
        AND (s.from_chain_id IS NULL
        AND s.tx_hash IS NULL)
	`,
		STATUS_PROCESSED, STATUS_CONFIRMED)
	if err := w.db.Raw(sql, chainId, updatedAfter).Scan(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

func (w *MySqlDB) PutGasSubsidy(gasSubsidy *GasSubsidy) error {
	err := w.db.Create(gasSubsidy).Error
	if err != nil {
		var mysqlErr *gomysql.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 { // if it is Duplicate-entry DB error
			return nil //IGNORE
		}
		return err
	}
	return nil
}

func (w *MySqlDB) GetFirstNotSentGasSubsidy() (*GasSubsidy, error) {
	var list []GasSubsidy
	sentStatuses := []string{STATUS_PROCESSED, STATUS_CONFIRMED, STATUS_TIMEDOUT, STATUS_TO_BE_REMOVED}
	err := w.db.Where("starcoin_tx_hash is null or starcoin_tx_hash = ''").Not(map[string]interface{}{"status": sentStatuses}).Limit(1).Find(&list).Error
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
	return &first, nil
}

func (w *MySqlDB) GetFirstTimedOutGasSubsidy() (*GasSubsidy, error) {
	var list []GasSubsidy
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
	if first.UpdatedAt < CurrentTimeMillis()-GasSubsidyTxMaxProcessingSeconds*1000 {
		return &first, nil
	} else {
		return nil, nil
	}
}

func (w *MySqlDB) GetFirstFailedGasSubsidy() (*GasSubsidy, error) {
	var list []GasSubsidy
	//err := w.db.Where("updated_at < ?", currentTimeMillis()-GasSubsidyTxMaxProcessingSeconds*1000).Not(map[string]interface{}{"status": []string{STATUS_PROCESSED, STATUS_CONFIRMED}}).Limit(1).Find(&list).Error
	notFailedStatuses := []string{STATUS_PROCESSED, STATUS_CONFIRMED, STATUS_TIMEDOUT, STATUS_TO_BE_REMOVED}
	err := w.db.Where("retry_count < ?", GasSubsidyTxMaxRetryCount).Not(map[string]interface{}{"status": notFailedStatuses}).Limit(1).Find(&list).Error
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
	if first.UpdatedAt < CurrentTimeMillis()-GasSubsidyTxMaxProcessingSeconds*1000 {
		return &first, nil
	} else {
		return nil, nil
	}
}

func (w *MySqlDB) SetGasSubsidyStarcoinTxInfo(txHash string, fromChainID uint64, oldStatus string, starcoinTxHash []byte, senderAddress []byte, senderSeqNum uint64) error {
	gasSubsidy := new(GasSubsidy)
	if err := w.db.Where(&GasSubsidy{TxHash: txHash, FromChainID: fromChainID}).First(gasSubsidy).Error; err != nil {
		return err
	}
	fieldMap := map[string]interface{}{
		"starcoin_tx_hash":       tools.EncodeToHex(starcoinTxHash),
		"sender_address":         tools.EncodeToHex(senderAddress),
		"sender_sequence_number": senderSeqNum,
		"status":                 STATUS_PROCESSING,
		"retry_count":            gasSubsidy.RetryCount + 1,
		"updated_at":             CurrentTimeMillis(),
	}
	return w.optimisticUpdateGasSubsidy(gasSubsidy, oldStatus, fieldMap)
}

func (w *MySqlDB) SetGasSubsidyStatusProcessed(txHash string, fromChainID uint64, oldStatus string) error {
	return w.SetGasSubsidyStatus(txHash, fromChainID, oldStatus, STATUS_PROCESSED)
}

func (w *MySqlDB) SetGasSubsidyStatus(txHash string, fromChainID uint64, oldStatus string, status string) error {
	// s := GasSubsidy{}
	// if err := w.db.Where(&GasSubsidy{
	// 	TxIndex:     gasSubsidy.TxIndex,
	// 	TxHash:      gasSubsidy.TxHash,
	// 	FromChainID: gasSubsidy.FromChainID,
	// }).First(&s).Error; err != nil {
	// 	return err
	// }
	gasSubsidy := new(GasSubsidy)
	if err := w.db.Where(&GasSubsidy{TxHash: txHash, FromChainID: fromChainID}).First(gasSubsidy).Error; err != nil {
		return err
	}
	// s.Status = status
	// s.UpdatedAt = CurrentTimeMillis()
	// return w.db.Save(&s).Error
	fieldMap := map[string]interface{}{
		"status":     status,
		"updated_at": CurrentTimeMillis(),
	}
	return w.optimisticUpdateGasSubsidy(gasSubsidy, oldStatus, fieldMap)
}

func (w *MySqlDB) optimisticUpdateGasSubsidy(gasSubsidy *GasSubsidy, oldStatus string, fieldMap map[string]interface{}) error {
	sql := "tx_index = ? and from_chain_id = ? and tx_hash = ? and status = ?"
	if gasSubsidy.StarcoinTxHash == "" {
		sql = sql + " and (starcoin_tx_hash is null or starcoin_tx_hash = '')"
	} else {
		sql = sql + " and starcoin_tx_hash = '" + strings.Replace(gasSubsidy.StarcoinTxHash, "'", "", -1) + "'"
	}
	db := w.db.Table("gas_subsidy").Where(
		sql,
		gasSubsidy.TxIndex, gasSubsidy.FromChainID, gasSubsidy.TxHash, oldStatus,
	).Updates(fieldMap)
	if db.Error != nil {
		return db.Error
	}
	rowsAffected := db.RowsAffected
	if rowsAffected == 0 {
		return fmt.Errorf("optimistic lock error. TxIndex: %d, fromChainId: %d, txHash: %s", gasSubsidy.TxIndex, gasSubsidy.FromChainID, gasSubsidy.TxHash)
	}
	return nil
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
	return fmt.Errorf("(m *PolyTxMapStore) Delete - NOT IMPLEMENTED ERROR")
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

// The 'Set' function updates the `value` of the `key``.
func (m *SmtNodeMapStore) Set(key []byte, value []byte) error {
	h := hex.EncodeToString(key)
	d := hex.EncodeToString(value)
	n := &SmtNode{
		Hash: h,
		Data: d,
	}
	err := m.db.db.Create(n).Error
	var mysqlErr *gomysql.MySQLError
	if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 { // if it is Duplicate-entry DB error
		oldData, err := m.Get(key)
		if err != nil {
			return err
		}
		if bytes.Equal(value, oldData) { // if it is really duplicate entry
			return nil
		} else {
			return fmt.Errorf("reset value is not allowed, key: %s, value: %s, old value: %s", h, d, hex.EncodeToString(oldData))
		}
	}
	return err
}

// The 'Delete' function deletes a key.
func (m *SmtNodeMapStore) Delete(key []byte) error {
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
