package chain_config

import (
	"math/big"

	"github.com/EBLA-network/ebla-evm/common"
	"github.com/EBLA-network/ebla-evm/core"
	"github.com/EBLA-network/ebla-evm/core/types"
	"github.com/EBLA-network/ebla-evm/core/vm"
	"github.com/EBLA-network/ebla-evm/params"
)

// SlashingConfig contains slashing / jailing parameters. Active from block 0.
type SlashingConfig struct {
	JailTime uint64 // number of blocks a double-voter stays jailed
}

// SupplyConfig holds permanent supply-cap parameters for EBLA. Active from block 0.
//   - MaxSupply: 12B EBLA hard cap, enforced every block by processBlockReward.
//   - GeneratedRewards: counter of rewards minted post-genesis, used by the
//     supply-cap invariant.
type SupplyConfig struct {
	MaxSupply        *big.Int // 12 billion EBLA hard cap
	GeneratedRewards *big.Int // genesis-seed amount counted against MaxSupply
}

// ProtocolConfig holds the permanent EBLA protocol parameters set at genesis
// and unchanged for the lifetime of the chain.
//
// RLP wire format: field declaration order is load-bearing.
// The C++-side mirror (libraries/config/include/config/protocol_config.hpp
// :: ProtocolConfig) MUST declare its fields in the same order.
// Do not reorder.
type ProtocolConfig struct {
	RewardsDistributionFrequency map[uint64]uint32
	Slashing                     SlashingConfig
	Supply                       SupplyConfig
}

func isForked(fork_start, block_num types.BlockNum) bool {
	if fork_start == types.BlockNumberNIL || block_num == types.BlockNumberNIL {
		return false
	}
	return fork_start <= block_num
}

// Rules returns the per-block EVM rules. Empty since Phase 14.3; reserved
// for future protocol upgrades that may need per-block flag gating.
func (c *ProtocolConfig) Rules(num types.BlockNum) vm.Rules {
	return vm.Rules{}
}

type GenesisValidator struct {
	Address     common.Address
	Owner       common.Address
	VrfKey      []byte
	Commission  uint16
	Endpoint    string
	Description string
	Delegations core.BalanceMap
}

type DPOSConfig = struct {
	EligibilityBalanceThreshold *big.Int
	VoteEligibilityBalanceStep  *big.Int
	ValidatorMaximumStake       *big.Int
	MinimumDeposit              *big.Int
	MaxBlockAuthorReward        uint16
	DagProposersReward          uint16
	CommissionChangeDelta       uint16
	CommissionChangeFrequency   uint32 // [number of blocks]
	DelegationDelay             uint32 // [number of blocks]
	DelegationLockingPeriod     uint32 // [number of blocks]
	BlocksPerYear               uint32 // [count]
	TrxMinGasPrice              uint64 // [wei] 1 Gwei minimum
	TrxMaxGasLimit              uint64 // max gas per transaction
	InitialValidators           []GenesisValidator
}

// ChainConfig field order must match C++'s ebla::state_api::Config.
// Position 4 (Protocol) must align with C++ position 4 (protocol).
type ChainConfig struct {
	EVMChainConfig  params.ChainConfig
	GenesisBalances core.BalanceMap
	DPOS            DPOSConfig
	Protocol        ProtocolConfig
}

func (self *ChainConfig) GenesisBalancesSum() *big.Int {
	sum := big.NewInt(0)
	for _, balance := range self.GenesisBalances {
		sum.Add(sum, balance)
	}

	return sum
}
