# Overview

The compiled binary `bin/taiko-client` is the main entrypoint which includes three sub-commands:

- `driver`: keep the L2 execution engine's chain in sync with the `TaikoL1` contract, by directing the L2 [execution engine](https://ethereum.org/en/glossary/#execution-client).
- `proposer`: propose new transactions from the L2 execution engine's transaction pool to the `TaikoL1` contract.
- `prover`: request ZK proofs from the zkEVM, and send transactions to prove the proposed blocks are valid or invalid.

## Driver

### Engine API

Driver directs a L2 execution engine to insert new blocks or reorg the local chain through the [Engine API](https://github.com/ethereum/execution-apis/blob/main/src/engine/specification.md).

### Chain synchronization process

> NOTE: The Taiko protocol allows a block's timestamp to be equal to its parent block's timestamp, which differs from the original Ethereum protocol. So it's fine that there are two `TaikoL1.proposeBlock` transactions included in one L1 block.

Driver will inform the L2 execution engine Taiko protocol contract's latest verified L2 head, and try to let it catch up the latest verified L2 block through P2P at first. Driver will monitor the execution engine's sync progress, if it is not able to make any new sync progress in a period of time, driver will switch to insert the verified blocks to its local chain through the Engine API one by one.

After the L2 execution engine catches up the latest verified L2 head, driver will subscribe to `TaikoL1.BlockProposed` events, and when a new pending block is proposed:

1. Get the corresponding `TaikoL1.proposeBlock` L1 transaction.
2. Decode the txList and block metadata from the transaction's calldata.
3. Check whether the txList is valid based on the rules defined in Taiko protocol.

If the txList is **valid**:

4. Assemble a deterministic `TaikoL2.anchor` transaction based on the rules defined in the protocol, and put it as the first transaction in the proposed txList.
5. Use this txList and the decoded block metadata to assemble a deterministic L2 block.
6. Direct L2 execution engine to insert this assembled block and set it as the current chain's head via the Engine API.

If the txList is **invalid**:

4. Create a `TaikoL2.invalidateBlock` transaction and then assemble a L2 block only including this transaction.
5. Direct the L2 execution engine to insert this block, but does not set it as the chain's head via the Engine API.

> NOTE: For more detailed information about: block metadata, please see `5.2.2 Block Metadata` in the white paper.

> NOTE: For more detailed information about txList validation rules, please see `5.3.1 Validation` in the white paper.

> NOTE: For more detailed information about the `TaikoL2.anchor` transaction and proposed block's determination, please see `5.4.1 Construction of Anchor Transactions` in the white paper.

## Proposer

### Proposing strategy

Since tokenomics have not been fully implemented in the Taiko protocol, the current proposing strategy is simply based on time interval.

### Proposing process

Proposing a block involves a few steps:

1. Fetch the pending transactions from the L2 execution engine through the `txpool_content` RPC method.
2. If there are too many pending transactions in the L2 execution engine, split them into several smaller txLists. This is because the Taiko protocol restricts the max size of each proposed txList.
3. Commit hashes of the txLists by sending `TaikoL1.commitBlock` transactions to L1.
4. Wait for `LibConstants.TAIKO_COMMIT_DELAY_CONFIRMATIONS` (currently `4`) L1 blocks confirmations.
5. Propose all splitted txLists by sending `TaikoL1.proposeBlock` transactions.

## Prover

### Proving strategy

Since tokenomics have not been fully implemented in the Taiko protocol, the prover software currently proves all proposed blocks.

### Proving process

When a new block is proposed:

1. Get the `TaikoL1.proposeBlock` L1 transaction calldata, decode it, and validate the txList. Just like what the `driver` software does.
2. Wait until the corresponding block is inserted by the L2 execution engine's `driver` software.
3. Generate a ZK proof for that block asynchronously.

If the proposed block has a valid txList:

4. Generate the merkel proof of the block's `TaikoL2.anchor` transaction to prove its existence in the `block.txRoot`'s [MPT](https://ethereum.org/en/developers/docs/data-structures-and-encoding/patricia-merkle-trie/), and also this transaction receipt's merkel proof in the `block.receiptRoot`'s MPT from the L2 execution engine.
5. Submit the `TaikoL2.anchor` transaction's RLP encoded bytes, its receipt's RLP encoded bytes, generated merkel proofs, and ZK proof to prove this block **valid**, by sending a `TaikoL1.proveBlock` transaction.

If the proposed block has an invalid txList:

4. Generate the merkel proof of the block's `TaikoL2.invalidateBlock` transaction receipt to prove its existence in the `block.receiptRoot`'s MPT from the L2 execution engine.
5. Submit the `TaikoL2.invalidateBlock` transaction receipt's RLP encoded bytes, generated merkel proof, and ZK proof to prove this block **invalid**, by sending a `TaikoL1.proveBlockInvalid` transaction.

> NOTE: For more information about why we need these merkel proofs when proving, please see `5.5 Proving Blocks` in the white paper.
