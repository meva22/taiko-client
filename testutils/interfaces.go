package testutils

import (
	"context"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/taikoxyz/taiko-client/bindings"
	"github.com/taikoxyz/taiko-client/cmd/utils"
)

type L2ChainSyncer interface {
	ProcessL1Blocks(ctx context.Context, l1End *types.Header) error
}

type Proposer interface {
	utils.SubcommandApplication
	ProposeOp(ctx context.Context) error
	CommitTxList(ctx context.Context, txListBytes []byte, gasLimit uint64, splittedIdx int) (
		*bindings.LibDataBlockMetadata,
		*types.Transaction,
		error,
	)
	ProposeTxList(
		ctx context.Context,
		meta *bindings.LibDataBlockMetadata,
		commitTx *types.Transaction,
		txListBytes []byte,
		txNum uint,
	) error
}
