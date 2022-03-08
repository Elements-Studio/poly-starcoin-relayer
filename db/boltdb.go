/*
* Copyright (C) 2020 The poly network Authors
* This file is part of The poly network library.
*
* The poly network is free software: you can redistribute it and/or modify
* it under the terms of the GNU Lesser General Public License as published by
* the Free Software Foundation, either version 3 of the License, or
* (at your option) any later version.
*
* The poly network is distributed in the hope that it will be useful,
* but WITHOUT ANY WARRANTY; without even the implied warranty of
* MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
* GNU Lesser General Public License for more details.
* You should have received a copy of the GNU Lesser General Public License
* along with The poly network . If not, see <http://www.gnu.org/licenses/>.
 */
package db

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"path"
	"strings"
	"sync"

	"github.com/boltdb/bolt"
	"github.com/starcoinorg/starcoin-go/client"
)

const MAX_NUM = 1000

var (
	BKTStarcoinTxCheck = []byte("StarcoinTxCheck")
	BKTStarcoinTxRetry = []byte("StarcoinTxRetry")
	BKTHeight          = []byte("Height")
)

type BoltDB struct {
	rwlock   *sync.RWMutex
	db       *bolt.DB
	filePath string
}

func NewBoltDB(filePath string) (*BoltDB, error) {
	if !strings.Contains(filePath, ".bin") {
		filePath = path.Join(filePath, "bolt.bin")
	}
	w := new(BoltDB)
	db, err := bolt.Open(filePath, 0644, &bolt.Options{InitialMmapSize: 500000})
	if err != nil {
		return nil, err
	}
	w.db = db
	w.rwlock = new(sync.RWMutex)
	w.filePath = filePath

	if err = db.Update(func(btx *bolt.Tx) error {
		_, err := btx.CreateBucketIfNotExists(BKTStarcoinTxCheck)
		if err != nil {
			return err
		}

		return nil
	}); err != nil {
		return nil, err
	}

	if err = db.Update(func(btx *bolt.Tx) error {
		_, err := btx.CreateBucketIfNotExists(BKTStarcoinTxRetry)
		if err != nil {
			return err
		}

		return nil
	}); err != nil {
		return nil, err
	}

	if err = db.Update(func(btx *bolt.Tx) error {
		_, err := btx.CreateBucketIfNotExists(BKTHeight)
		if err != nil {
			return err
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return w, nil
}

func (w *BoltDB) PutStarcoinTxCheck(txHash string, v []byte, e client.Event) error {
	return fmt.Errorf("NOT IMPLEMENTED ERROR")
	w.rwlock.Lock()
	defer w.rwlock.Unlock()

	k, err := hex.DecodeString(txHash)
	if err != nil {
		return err
	}
	return w.db.Update(func(btx *bolt.Tx) error {
		bucket := btx.Bucket(BKTStarcoinTxCheck)
		err := bucket.Put(k, v)
		if err != nil {
			return err
		}

		return nil
	})
}

func (w *BoltDB) DeleteStarcoinTxCheck(txHash string) error {
	w.rwlock.Lock()
	defer w.rwlock.Unlock()

	k, err := hex.DecodeString(txHash)
	if err != nil {
		return err
	}
	return w.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(BKTStarcoinTxCheck)
		err := bucket.Delete(k)
		if err != nil {
			return err
		}
		return nil
	})
}

func (w *BoltDB) PutStarcoinTxRetry(k []byte, event client.Event) error {
	return fmt.Errorf("NOT IMPLEMENTED ERROR")
	w.rwlock.Lock()
	defer w.rwlock.Unlock()

	return w.db.Update(func(btx *bolt.Tx) error {
		bucket := btx.Bucket(BKTStarcoinTxRetry)
		err := bucket.Put(k, []byte{0x00})
		if err != nil {
			return err
		}

		return nil
	})
}

func (w *BoltDB) DeleteStarcoinTxRetry(k []byte) error {
	w.rwlock.Lock()
	defer w.rwlock.Unlock()

	return w.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(BKTStarcoinTxRetry)
		err := bucket.Delete(k)
		if err != nil {
			return err
		}
		return nil
	})
}

func (w *BoltDB) GetAllStarcoinTxCheck() (map[string]BytesAndEvent, error) {
	return nil, fmt.Errorf("NOT IMPLEMENTED ERROR")
	// w.rwlock.Lock()
	// defer w.rwlock.Unlock()

	// checkMap := make(map[string][]byte)
	// err := w.db.Update(func(tx *bolt.Tx) error {
	// 	bw := tx.Bucket(BKTStarcoinTxCheck)
	// 	bw.ForEach(func(k, v []byte) error {
	// 		_k := make([]byte, len(k))
	// 		_v := make([]byte, len(v))
	// 		copy(_k, k)
	// 		copy(_v, v)
	// 		checkMap[hex.EncodeToString(_k)] = _v
	// 		if len(checkMap) >= MAX_NUM {
	// 			return fmt.Errorf("max num")
	// 		}
	// 		return nil
	// 	})
	// 	return nil
	// })
	// if err != nil {
	// 	return nil, err
	// }
	// return checkMap, nil
}

func (w *BoltDB) GetAllStarcoinTxRetry() ([][]byte, []client.Event, error) {
	return nil, nil, fmt.Errorf("NOT IMPLEMENTED ERROR")
	w.rwlock.Lock()
	defer w.rwlock.Unlock()

	retryList := make([][]byte, 0)
	err := w.db.Update(func(tx *bolt.Tx) error {
		bw := tx.Bucket(BKTStarcoinTxRetry)
		bw.ForEach(func(k, _ []byte) error {
			_k := make([]byte, len(k))
			copy(_k, k)
			retryList = append(retryList, _k)
			if len(retryList) >= MAX_NUM {
				return fmt.Errorf("max num")
			}
			return nil
		})
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	return retryList, nil, nil
}

func (w *BoltDB) UpdatePolyHeight(h uint32) error {
	w.rwlock.Lock()
	defer w.rwlock.Unlock()

	raw := make([]byte, 4)
	binary.LittleEndian.PutUint32(raw, h)

	return w.db.Update(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(BKTHeight)
		return bkt.Put([]byte(KEY_POLY_HEIGHT), raw)
	})
}

func (w *BoltDB) GetPolyHeight() (uint32, error) {
	w.rwlock.RLock()
	defer w.rwlock.RUnlock()

	var h uint32
	err := w.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(BKTHeight)
		raw := bkt.Get([]byte(KEY_POLY_HEIGHT))
		if len(raw) == 0 {
			h = 0
			return nil
		}
		h = binary.LittleEndian.Uint32(raw)
		return nil
	})
	if err != nil {
		return 0, err
	}
	return h, nil
}

func (w *BoltDB) GetPolyTxRetry(txHash string, fromChainID uint64) (*PolyTxRetry, error) {
	return nil, fmt.Errorf("NOT IMPLEMENTED ERROR")
}

func (w *BoltDB) GetAllPolyTxRetry() ([]*PolyTxRetry, error) {
	return nil, fmt.Errorf("NOT IMPLEMENTED ERROR")
}

func (w *BoltDB) DeletePolyTxRetry(txHash string, fromChainID uint64) error {
	return fmt.Errorf("NOT IMPLEMENTED ERROR")
}

func (w *BoltDB) PutPolyTxRetry(tx *PolyTxRetry) error {
	return fmt.Errorf("NOT IMPLEMENTED ERROR")
}

func (w *BoltDB) IncreasePolyTxRetryCheckFeeCount(txHash string, fromChainID uint64, oldCount int) error {
	return fmt.Errorf("NOT IMPLEMENTED ERROR")
}

func (w *BoltDB) SetPolyTxRetryFeeStatus(txHash string, fromChainID uint64, status string) error {
	return fmt.Errorf("NOT IMPLEMENTED ERROR")
}

func (w *BoltDB) UpdatePolyTxStarcoinStatus(txHash string, fromChainID uint64, status string, msg string) error {
	return fmt.Errorf("NOT IMPLEMENTED ERROR")
}

func (w *BoltDB) GetPolyTx(txHash string, fromChainID uint64) (*PolyTx, error) {
	return nil, fmt.Errorf("NOT IMPLEMENTED ERROR")
}

func (w *BoltDB) PutPolyTx(tx *PolyTx) (uint64, error) {
	return 0, fmt.Errorf("NOT IMPLEMENTED ERROR")
}

func (w *BoltDB) RemovePolyTx(tx *PolyTx) error {
	return fmt.Errorf("NOT IMPLEMENTED ERROR")
}

func PushBackRemovePolyTx(id uint64) error {
	return fmt.Errorf("NOT IMPLEMENTED ERROR")
}

func (w *BoltDB) UpdatePolyTxNonMembershipProofByIndex(idx uint64) error {
	return fmt.Errorf("NOT IMPLEMENTED ERROR")
}

func (w *BoltDB) SetPolyTxStatus(txHash string, fromChainID uint64, status string) error {
	return fmt.Errorf("NOT IMPLEMENTED ERROR")
}

func (w *BoltDB) SetPolyTxStatusProcessing(txHash string, fromChainID uint64) error {
	return fmt.Errorf("NOT IMPLEMENTED ERROR")
}

func (w *BoltDB) SetProcessingPolyTxStarcoinTxHash(txHash string, fromChainID uint64, starcoinTxHash string) error {
	return fmt.Errorf("NOT IMPLEMENTED ERROR")
}

func (w *BoltDB) SetPolyTxStatusProcessed(txHash string, fromChainID uint64, starcoinTxHash string) error {
	return fmt.Errorf("NOT IMPLEMENTED ERROR")
}

func (w *BoltDB) GetFirstFailedPolyTx() (*PolyTx, error) {
	return nil, fmt.Errorf("NOT IMPLEMENTED ERROR")
}

func (w *BoltDB) GetFirstTimedOutPolyTx() (*PolyTx, error) {
	return nil, fmt.Errorf("NOT IMPLEMENTED ERROR")
}

func (w *BoltDB) GetFirstPolyTxToBeRemoved() (*PolyTx, error) {
	return nil, fmt.Errorf("NOT IMPLEMENTED ERROR")
}

func (w *BoltDB) GetFirstRemovedPolyTxToBePushedBack() (*RemovedPolyTx, error) {
	return nil, fmt.Errorf("NOT IMPLEMENTED ERROR")
}

func (w *BoltDB) GetTimedOutOrFailedPolyTxList() ([]*PolyTx, error) {
	return nil, fmt.Errorf("NOT IMPLEMENTED ERROR")
}

func (w *BoltDB) Close() {
	w.rwlock.Lock()
	w.db.Close()
	w.rwlock.Unlock()
}
