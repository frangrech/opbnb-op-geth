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

package pathdb

import (
	"fmt"
	"io"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/trie/trienode"
	"github.com/ethereum/go-ethereum/trie/triestate"
)

// historicalDiskLayer is a read-only layer that serves trie nodes for a
// specific historical state.  It reads current values from disk and uses the
// KV trienode history index to retrieve before-values for nodes that have been
// overwritten since the target stateID.
type historicalDiskLayer struct {
	root common.Hash
	id   uint64
	db   *Database
}

func newHistoricalDiskLayer(root common.Hash, id uint64, db *Database) *historicalDiskLayer {
	return &historicalDiskLayer{root: root, id: id, db: db}
}

// Node implements the layer interface.
func (hl *historicalDiskLayer) Node(owner common.Hash, path []byte, hash common.Hash) ([]byte, error) {
	// Read the current (latest) disk value.
	var (
		nBlob []byte
		nHash common.Hash
	)
	if owner == (common.Hash{}) {
		nBlob, nHash = rawdb.ReadAccountTrieNode(hl.db.diskdb, path)
	} else {
		nBlob, nHash = rawdb.ReadStorageTrieNode(hl.db.diskdb, owner, path)
	}
	// If current disk value has the expected hash, no reversion needed.
	if nHash == hash {
		return nBlob, nil
	}
	// Look up the first recorded change after our target stateID.
	// The stored blob is the before-value: what the node was immediately before
	// that state transition, which is also its value at hl.id.
	_, blob, found := rawdb.LookupTrienodeFirstChangeAfter(hl.db.diskdb, owner, path, hl.id)
	if !found {
		// No change recorded after hl.id; disk value should be authoritative.
		// A hash mismatch here indicates missing history (pre-archive import)
		// or data corruption.
		if len(nBlob) == 0 {
			return nil, nil
		}
		return nil, fmt.Errorf("trie node not found in archive history (owner=%x path=%x expected=%x disk=%x)", owner, path, hash, nHash)
	}
	// A nil/empty blob means the node was created at the found stateID, so it
	// did not exist at hl.id.
	if len(blob) == 0 {
		return nil, nil
	}
	// Integrity check.
	if got := crypto.Keccak256Hash(blob); got != hash {
		return nil, fmt.Errorf("archive trie node hash mismatch (owner=%x path=%x expected=%x got=%x)", owner, path, hash, got)
	}
	return blob, nil
}

// rootHash implements the layer interface.
func (hl *historicalDiskLayer) rootHash() common.Hash { return hl.root }

// stateID implements the layer interface.
func (hl *historicalDiskLayer) stateID() uint64 { return hl.id }

// parentLayer implements the layer interface; historical layers have no parent.
func (hl *historicalDiskLayer) parentLayer() layer { return nil }

// update is not supported on a historical read-only layer.
func (hl *historicalDiskLayer) update(_ common.Hash, _ uint64, _ uint64, _ map[common.Hash]map[string]*trienode.Node, _ *triestate.Set) *diffLayer {
	panic("update called on historical disk layer")
}

// journal is not supported on a historical read-only layer.
func (hl *historicalDiskLayer) journal(_ io.Writer, _ JournalType) error {
	return fmt.Errorf("journal not supported for historical disk layer")
}
