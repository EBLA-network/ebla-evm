package chain_config

import (
	"math/big"

	"github.com/EBLA-network/ebla-evm/common"
	"github.com/EBLA-network/ebla-evm/core"
	"github.com/EBLA-network/ebla-evm/core/types"
	"github.com/EBLA-network/ebla-evm/core/vm"
	"github.com/EBLA-network/ebla-evm/params"
)

type Redelegation struct {
	Validator common.Address
	Delegator common.Address
	Amount    *big.Int
}

// SlashingConfig contains slashing/jailing parameters.
// Originally introduced by Taraxa's Magnolia hardfork; features are permanent in EBLA
// from block 0, so the hardfork gate has been removed. Only the runtime JailTime
// parameter remains.
type SlashingConfig struct {
	JailTime uint64 // number of blocks a double-voter stays jailed
}

type AspenHfConfig struct {
	BlockNumPartOne  uint64 // part 1 just starts to save minted tokens (rewards) in db
	BlockNumPartTwo  uint64 // part 2 implements new dynamic yield curve
	MaxSupply        *big.Int
	GeneratedRewards *big.Int // Total number of generated rewards between block 0 and AspenHf BlockNum
}

type FicusHfConfig struct {
	BlockNum              uint64
	PillarBlocksInterval  uint64 // [number of blocks]
	BridgeContractAddress common.Address
}

// Leaving it here for next HF
// type BambooRedelegation struct {
// 	Validator common.Address
// 	Amount    *big.Int
// }
// type BambooHfConfig struct {
// 	BlockNum      uint64
// 	Redelegations []BambooRedelegation
// }

type HardforksConfig struct {
	FixRedelegateBlockNum        uint64
	Redelegations                []Redelegation
	RewardsDistributionFrequency map[uint64]uint32
	Slashing                     SlashingConfig
	PhalaenopsisHfBlockNum       uint64
	AspenHf                      AspenHfConfig
	FicusHf                      FicusHfConfig
	InactivityPenaltyBlock       uint64 // Block at which inactivity penalty activates (0 for EBLA genesis)
}

func (c *HardforksConfig) IsOnPhalaenopsisHardfork(block types.BlockNum) bool {
	return block >= c.PhalaenopsisHfBlockNum
}

func (c *HardforksConfig) IsOnAspenHardforkPartOne(block types.BlockNum) bool {
	return block >= c.AspenHf.BlockNumPartOne
}

func (c *HardforksConfig) IsOnAspenHardforkPartTwo(block types.BlockNum) bool {
	return block >= c.AspenHf.BlockNumPartTwo
}

func (c *HardforksConfig) IsOnFicusHardfork(block types.BlockNum) bool {
	return block >= c.FicusHf.BlockNum
}

func isForked(fork_start, block_num types.BlockNum) bool {
	if fork_start == types.BlockNumberNIL || block_num == types.BlockNumberNIL {
		return false
	}
	return fork_start <= block_num
}

func (c *HardforksConfig) Rules(num types.BlockNum) vm.Rules {
	return vm.Rules{
		IsAspenPartOne: isForked(c.AspenHf.BlockNumPartOne, num),
		IsAspenPartTwo: isForked(c.AspenHf.BlockNumPartTwo, num),
		IsFicus:        isForked(c.FicusHf.BlockNum, num),
	}
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

func (c *HardforksConfig) IsOnInactivityPenaltyHardfork(block types.BlockNum) bool {
	return block >= c.InactivityPenaltyBlock
}
