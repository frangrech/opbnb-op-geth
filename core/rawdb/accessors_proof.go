package rawdb

import (
	"path/filepath"

	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
)

const (
	blockNumberLength = 8 // uint64 is 8 bytes.
	proposeProofTable = "propose.proof"
)

var proofFreezerTableConfigs = map[string]freezerTableConfig{
	proposeProofTable: {noSnappy: true},
}

// NewProofFreezer initializes the freezer for propose withdraw proof.
func NewProofFreezer(ancientDir string, readOnly bool) (*ResettableFreezer, error) {
	return NewResettableFreezer(filepath.Join(ancientDir, ProofFreezerName), "eth/db/proof", readOnly, stateHistoryTableSize, proofFreezerTableConfigs)
}

// IterateKeeperMeta returns keep meta iterator.
func IterateKeeperMeta(db ethdb.Iteratee) ethdb.Iterator {
	return NewKeyLengthIterator(db.NewIterator(proofKeeperMetaPrefix, nil), len(proofKeeperMetaPrefix)+blockNumberLength)
}

// DeleteKeeperMeta removes the specified keeper meta.
func DeleteKeeperMeta(db ethdb.KeyValueWriter, blockID uint64) {
	if err := db.Delete(proofKeeperMetaKey(blockID)); err != nil {
		log.Crit("Failed to delete keeper meta", "err", err)
	}
}

// PutKeeperMeta adds a new keeper meta.
func PutKeeperMeta(db ethdb.KeyValueWriter, blockID uint64, meta []byte) {
	if err := db.Put(proofKeeperMetaKey(blockID), meta); err != nil {
		log.Crit("Failed to store keeper meta", "err", err)
	}
}

// GetLatestProofData returns the latest proof data from the freezer.
func GetLatestProofData(f *ResettableFreezer) []byte {
	n, err := f.Ancients()
	if err != nil || n == 0 {
		return nil
	}
	blob, err := f.Ancient(proposeProofTable, n-1)
	if err != nil {
		log.Error("Failed to get latest proof data", "latest_proof_id", n-1, "error", err)
		return nil
	}
	return blob
}

// GetProofData returns the specified proof data.
func GetProofData(f *ResettableFreezer, proofID uint64) []byte {
	blob, err := f.Ancient(proposeProofTable, proofID)
	if err != nil {
		return nil
	}
	return blob
}

// TruncateProofDataHead discards proof data above the provided threshold.
func TruncateProofDataHead(f *ResettableFreezer, proofID uint64) {
	if _, err := f.TruncateHead(proofID); err != nil {
		log.Error("Failed to truncate proof data head", "proofID", proofID, "error", err)
	}
}

// TruncateProofDataTail discards proof data below the provided threshold.
func TruncateProofDataTail(f *ResettableFreezer, proofID uint64) {
	if _, err := f.TruncateTail(proofID); err != nil {
		log.Error("Failed to truncate proof data tail", "proofID", proofID, "error", err)
	}
}

// PutProofData appends a new proof to the ancient proof db.
func PutProofData(db ethdb.AncientWriter, proofID uint64, proof []byte) error {
	_, err := db.ModifyAncients(func(op ethdb.AncientWriteOp) error {
		return op.AppendRaw(proposeProofTable, proofID, proof)
	})
	return err
}
