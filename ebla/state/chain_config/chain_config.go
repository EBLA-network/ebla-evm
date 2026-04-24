package chain_config

import (
	"math/big"

	"github.com/EBLA-network/ebla-evm/common"
	"github.com/EBLA-network/ebla-evm/core"
	"github.com/EBLA-network/ebla-evm/core/types"
	"github.com/EBLA-network/ebla-evm/core/vm"
	"github.com/EBLA-network/ebla-evm/params"
)

// SlashingConfig contains slashing/jailing parameters.
// Originally introduced by Taraxa's Magnolia hardfork; features are permanent in EBLA
// from block 0, so the hardfork gate has been removed. Only the runtime JailTime
// parameter remains.
type SlashingConfig struct {
	JailTime uint64 // number of blocks a double-voter stays jailed
}

// AspenHfConfig holds permanent supply-cap / yield-curve parameters in EBLA.
// The original Taraxa "Aspen hardfork" block-number gates (part 1 = minted
// tokens DB, part 2 = dynamic yield curve) were removed in Phase 14.3 —
// both behaviors are now unconditional from block 0. Only the supply
// invariants remain.
type AspenHfConfig struct {
	MaxSupply        *big.Int // 12 billion EBLA hard cap
	GeneratedRewards *big.Int // genesis-seed amount counted against MaxSupply
}

// Leaving it here for next HF
// type BambooRedelegation struct {
// 	Validator common.Address
// 	Amount    *big.Int
// }

type HardforksConfig struct {
	RewardsDistributionFrequency map[uint64]uint32
	Slashing                     SlashingConfig
	AspenHf                      AspenHfConfig
}

func isForked(fork_start, block_num types.BlockNum) bool {
	if fork_start == types.BlockNumberNIL || block_num == types.BlockNumberNIL {
		return false
	}
	return fork_start <= block_num
}

func (c *HardforksConfig) Rules(num types.BlockNum) vm.Rules {
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
	YieldPercentage             uint16 // [%]
	TrxMinGasPrice              uint64 // [wei] 1 Gwei minimum
	TrxMaxGasLimit              uint64 // max gas per transaction
	InitialValidators           []GenesisValidator
}

type ChainConfig struct {
	EVMChainConfig  params.ChainConfig
	GenesisBalances core.BalanceMap
	DPOS            DPOSConfig
	Hardforks       HardforksConfig
}

func (self *ChainConfig) RewardsEnabled() bool {
	return self.DPOS.YieldPercentage > 0
}

func (self *ChainConfig) GenesisBalancesSum() *big.Int {
	sum := big.NewInt(0)
	for _, balance := range self.GenesisBalances {
		sum.Add(sum, balance)
	}

	return sum
}
