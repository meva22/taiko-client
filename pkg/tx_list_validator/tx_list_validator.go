package tx_list_validator

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/taikoxyz/taiko-client/bindings/encoding"
)

// InvalidTxListReason represents a reason why a transactions list is invalid,
// must match the definitions in LibInvalidTxList.sol:
//
//	enum Reason {
//		OK,
//		BINARY_TOO_LARGE,
//		BINARY_NOT_DECODABLE,
//		BLOCK_TOO_MANY_TXS,
//		BLOCK_GAS_LIMIT_TOO_LARGE,
//		TX_INVALID_SIG,
//		TX_GAS_LIMIT_TOO_SMALL
//	}
type InvalidTxListReason uint8

// All invalid transactions list reasons.
const (
	HintOK InvalidTxListReason = iota
	HintBinaryTooLarge
	HintBinaryNotDecodable
	HintBlockTooManyTxs
	HintBlockGasLimitTooLarge
	HintTxInvalidSig
	HintTxGasLimitTooSmall
)

type TxListValidator struct {
	blockMaxGasLimit uint64
	blockMaxTxs      uint64
	txListMaxBytes   uint64
	txMinGasLimit    uint64
	chainID          *big.Int
}

// NewTxListValidator creates a new TxListValidator instance based on giving configurations.
func NewTxListValidator(
	blockMaxGasLimit uint64,
	blockMaxTxs uint64,
	txListMaxBytes uint64,
	txMinGasLimit uint64,
	chainID *big.Int,
) *TxListValidator {
	return &TxListValidator{
		blockMaxGasLimit: blockMaxGasLimit,
		blockMaxTxs:      blockMaxTxs,
		txListMaxBytes:   txListMaxBytes,
		txMinGasLimit:    txMinGasLimit,
		chainID:          chainID,
	}
}

// ValidateTxList checks whether the transactions list in the TaikoL1.proposeBlock transaction's
// input data is valid.
func (v *TxListValidator) ValidateTxList(
	blockID *big.Int,
	proposeBlockTxInput []byte,
) (txListBytes []byte, hint InvalidTxListReason, txIdx int, err error) {
	if txListBytes, err = encoding.UnpackTxListBytes(proposeBlockTxInput); err != nil {
		return nil, HintBinaryNotDecodable, 0, err
	}

	if len(txListBytes) == 0 {
		return txListBytes, HintOK, 0, nil
	}

	hint, txIdx = v.isTxListValid(blockID, txListBytes)

	return txListBytes, hint, txIdx, nil
}

// isTxListValid checks whether the transaction list is valid, must match
// the validation rule defined in LibInvalidTxList.sol.
// ref: https://github.com/taikoxyz/taiko-mono/blob/main/packages/bindings/contracts/libs/LibInvalidTxList.sol
func (v *TxListValidator) isTxListValid(blockID *big.Int, txListBytes []byte) (hint InvalidTxListReason, txIdx int) {
	if len(txListBytes) > int(v.txListMaxBytes) {
		log.Info("Transactions list binary too large", "length", len(txListBytes), "blockID", blockID)
		return HintBinaryTooLarge, 0
	}

	var txs types.Transactions
	if err := rlp.DecodeBytes(txListBytes, &txs); err != nil {
		log.Info("Failed to decode transactions list bytes", "blockID", blockID, "error", err)
		return HintBinaryNotDecodable, 0
	}

	log.Debug("Transactions list decoded", "blockID", blockID, "length", len(txs))

	if txs.Len() > int(v.blockMaxTxs) {
		log.Info("Too many transactions", "blockID", blockID, "count", txs.Len())
		return HintBlockTooManyTxs, 0
	}

	sumGasLimit := uint64(0)
	for _, tx := range txs {
		sumGasLimit += tx.Gas()
	}

	if sumGasLimit > v.blockMaxGasLimit {
		log.Info("Accumulate gas limit too large", "blockID", blockID, "sumGasLimit", sumGasLimit)
		return HintBlockGasLimitTooLarge, 0
	}

	signer := types.LatestSignerForChainID(v.chainID)

	for i, tx := range txs {
		sender, err := types.Sender(signer, tx)
		if err != nil || sender == (common.Address{}) {
			log.Info("Invalid transaction signature", "error", err)
			return HintTxInvalidSig, i
		}

		if tx.Gas() < v.txMinGasLimit {
			log.Info("Transaction gas limit too small", "gasLimit", tx.Gas())
			return HintTxGasLimitTooSmall, i
		}
	}

	log.Info("Transaction list is valid", "blockID", blockID)
	return HintOK, 0
}
