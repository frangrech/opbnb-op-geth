// Copyright 2024 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package rawdb

import (
	"encoding/binary"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
)

// WriteTrienodeHistory stores the before-value of a trie node as of the given
// stateID into db. blob is the node's serialised form before the state transition
// at stateID; a nil/empty blob means the node was created by that transition.
func WriteTrienodeHistory(db ethdb.KeyValueWriter, owner common.Hash, path []byte, stateID uint64, blob []byte) {
	key := trienodeHistoryEntryKey(owner, path, stateID)
	if err := db.Put(key, blob); err != nil {
		log.Crit("Failed to write trienode history", "owner", owner, "stateID", stateID, "err", err)
	}
}

// LookupTrienodeFirstChangeAfter returns the smallest stateID S > target for
// which a before-value was recorded for the trie node identified by (owner, path).
// The returned blob is the node's serialised form immediately before S; nil means
// the node did not exist before S.  found is false when no change after target is
// recorded (i.e. the current disk value is authoritative for stateID target).
func LookupTrienodeFirstChangeAfter(db ethdb.Database, owner common.Hash, path []byte, target uint64) (id uint64, blob []byte, found bool) {
	prefix := trienodeHistoryEntryPrefixKey(owner, path)

	var startSuffix [8]byte
	binary.BigEndian.PutUint64(startSuffix[:], target+1)

	it := db.NewIterator(prefix, startSuffix[:])
	defer it.Release()

	if !it.Next() {
		return 0, nil, false
	}
	key := it.Key()
	if len(key) != len(prefix)+8 {
		return 0, nil, false
	}
	id = binary.BigEndian.Uint64(key[len(key)-8:])
	return id, common.CopyBytes(it.Value()), true
}
