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
package tools

import (
	"context"
	"encoding/hex"
	"sort"
	"sync"
	"time"

	"github.com/elements-studio/poly-starcoin-relayer/log"
	stcclient "github.com/starcoinorg/starcoin-go/client"
	"github.com/starcoinorg/starcoin-go/types"
)

const clear_seqNum_interval = 10 * time.Minute

type SeqNumManager struct {
	addressSeqNum  map[types.AccountAddress]uint64
	returnedSeqNum map[types.AccountAddress]SortedSeqNumArr
	starcoinClient *stcclient.StarcoinClient
	lock           sync.Mutex
}

func NewSeqNumManager(starcoinClient *stcclient.StarcoinClient) *SeqNumManager {
	SeqNumManager := &SeqNumManager{
		addressSeqNum:  make(map[types.AccountAddress]uint64),
		starcoinClient: starcoinClient,
		returnedSeqNum: make(map[types.AccountAddress]SortedSeqNumArr),
	}
	//go SeqNumManager.clearSeqNum()
	return SeqNumManager
}

// return account seqNum, and than seqNum++
func (this *SeqNumManager) GetAddressSeqNum(address types.AccountAddress) uint64 {
	this.lock.Lock()
	defer this.lock.Unlock()

	if this.returnedSeqNum[address].Len() > 0 {
		seqNum := this.returnedSeqNum[address][0]
		this.returnedSeqNum[address] = this.returnedSeqNum[address][1:]
		return seqNum
	}

	// return a new point
	seqNum, ok := this.addressSeqNum[address]
	if !ok {
		// get seqNum from eth network
		uintSeqNum, err := this.starcoinClient.GetAccountSequenceNumber(context.Background(), "0x"+hex.EncodeToString(address[:]))
		if err != nil {
			log.Errorf("GetAddressSeqNum: cannot get account %s seqNum, err: %s, set it to nil!",
				address, err)
		}
		this.addressSeqNum[address] = uintSeqNum
		seqNum = uintSeqNum
	}
	// increase record
	this.addressSeqNum[address]++
	return seqNum
}

func (this *SeqNumManager) ReturnSeqNum(addr types.AccountAddress, seqNum uint64) {
	this.lock.Lock()
	defer this.lock.Unlock()

	arr, ok := this.returnedSeqNum[addr]
	if !ok {
		arr = make([]uint64, 0)
	}
	arr = append(arr, seqNum)
	sort.Sort(arr)
	this.returnedSeqNum[addr] = arr
}

func (this *SeqNumManager) DecreaseAddressSeqNum(address types.AccountAddress) {
	this.lock.Lock()
	defer this.lock.Unlock()

	seqNum, ok := this.addressSeqNum[address]
	if ok && seqNum > 0 {
		this.addressSeqNum[address]--
	}
}

// clear seqNum
func (this *SeqNumManager) clearSeqNum() {
	for {
		<-time.After(clear_seqNum_interval)
		this.lock.Lock()
		for addr, _ := range this.addressSeqNum {
			delete(this.addressSeqNum, addr)
		}
		this.lock.Unlock()
		//log.Infof("clearSeqNum: clear all cache seqNum")
	}
}

type SortedSeqNumArr []uint64

func (arr SortedSeqNumArr) Less(i, j int) bool {
	return arr[i] < arr[j]
}

func (arr SortedSeqNumArr) Len() int { return len(arr) }

func (arr SortedSeqNumArr) Swap(i, j int) { arr[i], arr[j] = arr[j], arr[i] }
