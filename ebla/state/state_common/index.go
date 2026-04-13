package state_common

import (
	"github.com/EBLA-network/ebla-evm/common"
	"github.com/EBLA-network/ebla-evm/consensus/ethash"
	"github.com/EBLA-network/ebla-evm/crypto"
	"github.com/EBLA-network/ebla-evm/rlp"
)

type UncleBlock = ethash.BlockNumAndCoinbase

var EmptyRLPListHash = func() common.Hash {
	return crypto.Keccak256Hash(rlp.MustEncodeToBytes([]byte(nil)))
}()

func IsEmptyStateRoot(state_root *common.Hash) bool {
	return state_root == nil || *state_root == EmptyRLPListHash || *state_root == common.ZeroHash
}
