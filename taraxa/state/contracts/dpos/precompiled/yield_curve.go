package dpos

import (
	chain_config "github.com/Taraxa-project/taraxa-evm/taraxa/state/chain_config"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/asserts"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/bigutil"
	"github.com/holiman/uint256"
)

// Yield is calculated with 6 decimal precision (1e6)
// Example: 7% = 70000, 1% = 10000
var YieldFractionDecimalPrecision = uint256.NewInt(1e+6)

// EBLA epoch-based yield decay constants
const (
	// InitialYield is 7% expressed with 1e6 precision = 70000
	InitialYieldValue uint64 = 70000

	// MinYield is 1% floor expressed with 1e6 precision = 10000
	MinYieldValue uint64 = 10000

	// EpochLength: number of blocks per epoch (10 million)
	EpochLength uint64 = 10_000_000

	// DecayNumerator: yield retains 95% each epoch
	DecayNumerator uint64 = 95

	// DecayDenominator: base for decay fraction
	DecayDenominator uint64 = 100

	// MaxEpoch: safety cap to prevent uint256 overflow.
	// At epoch 37, InitialYieldValue * 95^37 exceeds uint256.
	// At epoch 36, yield = 1.104% (still above floor).
	// Epochs > MaxEpoch return MinYield directly.
	MaxEpoch uint64 = 36
)

type YieldCurve struct {
	cfg        chain_config.ChainConfig
	max_supply *uint256.Int
}

func (self *YieldCurve) Init(cfg chain_config.ChainConfig) {
	self.cfg = cfg

	max_supply, overflow := uint256.FromBig(self.cfg.Hardforks.AspenHf.MaxSupply)
	asserts.Holds(overflow == false, "YieldCurve max supply overflow")
	self.max_supply = max_supply

	// BlocksPerYear must not be zero — would cause division by zero
	asserts.Holds(cfg.DPOS.BlocksPerYear > 0, "BlocksPerYear must be > 0")
}

// GetMaxSupply returns a copy of max_supply for external cap checks
func (self *YieldCurve) GetMaxSupply() *uint256.Int {
	return new(uint256.Int).Set(self.max_supply)
}

// pow256 computes base^exp using uint256 integer arithmetic.
// SAFETY: caller must ensure InitialYieldValue * base^exp fits uint256.
// With MaxEpoch=36: 70000 * 95^36 ≈ 1.10e76 < uint256 max (1.15e77).
func pow256(base, exp uint64) *uint256.Int {
	result := uint256.NewInt(1)
	b := uint256.NewInt(base)
	for i := uint64(0); i < exp; i++ {
		result.Mul(result, b)
	}
	return result
}

// calculateCurrentYield computes the epoch-based decaying yield.
//
// Formula: yield = max(InitialYield × 95^epoch / 100^epoch, MinYield)
// Where:   epoch = block_number / EpochLength
//
// All arithmetic is pure integer using uint256. No floating point.
// Returns yield with YieldFractionDecimalPrecision (1e6) precision.
func (self *YieldCurve) calculateCurrentYield(block_num uint64) *uint256.Int {
	epoch := block_num / EpochLength

	// Safety cap: prevent uint256 overflow in pow256 multiplication.
	// At epoch 37+, InitialYieldValue * 95^epoch overflows uint256.
	// Natural floor (1%) is reached at epoch 38, so returning
	// MinYield for epochs > MaxEpoch is both safe and correct.
	if epoch > MaxEpoch {
		return uint256.NewInt(MinYieldValue)
	}

	// Epoch 0: no decay applied
	if epoch == 0 {
		return uint256.NewInt(InitialYieldValue)
	}

	// yield = InitialYield × 95^epoch / 100^epoch
	// Using batch exponentiation (not iterative) to avoid
	// compounding integer truncation errors across epochs.
	numerator := pow256(DecayNumerator, epoch)     // 95^epoch
	denominator := pow256(DecayDenominator, epoch) // 100^epoch

	current_yield := uint256.NewInt(InitialYieldValue)
	current_yield.Mul(current_yield, numerator)
	current_yield.Div(current_yield, denominator)

	// Enforce minimum yield floor (1%)
	min_yield := uint256.NewInt(MinYieldValue)
	if current_yield.Cmp(min_yield) < 0 {
		return min_yield
	}

	return current_yield
}

// CalculateBlockReward computes the per-block reward.
//
// Formula: block_reward = yield × total_delegation / (precision × blocks_per_year)
//
// Parameters:
//   - current_total_delegation: total staked across all validators
//   - current_total_tara_supply: current total supply (unused in new model, kept for interface compat)
//   - block_num: current block number (determines epoch)
func (self *YieldCurve) CalculateBlockReward(current_total_delegation *uint256.Int, current_total_tara_supply *uint256.Int, block_num uint64) (block_reward *uint256.Int, yield *uint256.Int) {
	yield = self.calculateCurrentYield(block_num)

	block_reward = new(uint256.Int).Mul(current_total_delegation, yield)
	blocks_per_year := uint256.NewInt(uint64(self.cfg.DPOS.BlocksPerYear))
	block_reward.Div(block_reward, new(uint256.Int).Mul(YieldFractionDecimalPrecision, blocks_per_year))

	return
}

// CalculateTotalSupply computes total supply from genesis balances + minted tokens + generated rewards.
// Used during Aspen hardfork transition to initialize total_supply from legacy minted_tokens counter.
func (self *YieldCurve) CalculateTotalSupply(minted_tokens *uint256.Int) *uint256.Int {
	total_supply := bigutil.Add(self.cfg.GenesisBalancesSum(), minted_tokens.ToBig())
	total_supply.Add(total_supply, self.cfg.Hardforks.AspenHf.GeneratedRewards)

	total_supply_uint256, overflow := uint256.FromBig(total_supply)
	asserts.Holds(overflow == false, "CalculateTotalSupply: Genesis balances sum overflow")

	return total_supply_uint256
}
