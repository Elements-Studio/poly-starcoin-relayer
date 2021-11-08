package db

import (
	"errors"

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
	db.AutoMigrate(&ChainHeight{})

	w := new(MySqlDB)
	w.db = db
	return w, nil
}

func (w *MySqlDB) PutStarcoinTxCheck(txHash string, v []byte) error {
	return nil //todo
}

func (w *MySqlDB) DeleteStarcoinTxCheck(txHash string) error {
	return nil //todo
}

// Put Starcoin cross-chain Tx.(to poly) Retry
func (w *MySqlDB) PutStarcoinTxRetry(k []byte) error {
	return nil //todo
}

func (w *MySqlDB) DeleteStarcoinTxRetry(k []byte) error {
	return nil //todo
}

func (w *MySqlDB) GetAllStarcoinTxCheck() (map[string][]byte, error) {
	return nil, nil //todo
}

func (w *MySqlDB) GetAllStarcoinTxRetry() ([][]byte, error) {
	return nil, nil //todo
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
