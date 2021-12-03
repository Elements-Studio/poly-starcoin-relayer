package db

import (
	"encoding/hex"
	"errors"

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
	db.Set("gorm:table_options", "CHARSET=latin1").AutoMigrate(&PolyTx{})

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
	ch := ChainHeight{
		Key: KEY_POLY_HEIGHT,
	}
	if err := w.db.First(&ch).Error; err != nil {
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
	px := PolyTx{
		TxHash: txHash,
	}
	if err := w.db.First(&px).Error; err != nil {
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
	var lastTx PolyTx
	var lastIndex uint64
	err := w.db.Last(&lastTx).Error
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, err
		} else {
			lastIndex = 0
		}
	} else {
		lastIndex = lastTx.TxIndex
	}
	// tx := PolyTx{
	// 	TxIndex: lastIndex + 1,
	// 	TxHash:  txHash,
	// }
	tx.TxIndex = lastIndex + 1
	err = w.db.Create(tx).Error
	if err != nil {
		return 0, err
	}
	return tx.TxIndex, err
}

func (w *MySqlDB) Close() {
	//
}

func createOrUpdate(db *gorm.DB, dest interface{}) error {
	if err := db.Save(dest).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		} else {
			db.Create(dest)
		}
	}
	return nil
}
