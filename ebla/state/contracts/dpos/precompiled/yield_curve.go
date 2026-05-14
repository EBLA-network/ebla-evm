package dpos

import (
	chain_config "github.com/EBLA-network/ebla-evm/ebla/state/chain_config"
	"github.com/EBLA-network/ebla-evm/ebla/util/asserts"
	"github.com/EBLA-network/ebla-evm/ebla/util/bigutil"
	"github.com/holiman/uint256"
)

// Yield is calculated with 6 decimal precision (1e6)
// Example: 7% = 70000, 1% = 10000
var YieldFractionDecimalPrecision = uint256.NewInt(1e+6)

// EBLA epoch-based yield decay constants
const (
	// MinYieldValue: the 1% yield floor (1e6 precision).
	// Applies from epoch 38 onward, where natural decay falls below 1.0000%.
	MinYieldValue uint64 = 10000
	// EpochLength: number of blocks per epoch (10 million ≈ 14 months at ~3.2s/block).
	EpochLength uint64 = 10_000_000

	MaxEpoch uint64 = 37
)

// EblaYieldTable encodes the per-epoch yield rate (1e6 precision).
//
// SPEC: Initial 7%, decays by 5% per epoch (10M blocks ≈ 14 months) until
// the natural decay value falls below the 1% floor at epoch 38.
//
// Each value is computed as floor(70000 × 95^epoch / 100^epoch), with all
// arithmetic performed at full precision and floor-divided exactly once at
// the end. This avoids accumulated truncation that would result from
// iterative compounding.
//
// At epoch 37 the natural value (10492 = 1.0492%) is still above the floor
// (10000 = 1.0000%), so it is encoded explicitly. Epoch 38+ falls below
// the floor naturally and is served by the floor branch in calculateCurrentYield.
//
// CONSENSUS-CRITICAL: This table is part of EBLA's economic consensus.
// Modifying any entry changes the chain's emission schedule. Any change
// requires a coordinated protocol upgrade. Verified against the canonical formula
// by TestEblaYieldTableMatchesFormula in dpos_test.go (CI-gated).
//
// Generation reference (Python):
//
//	for e in range(38): print((70000 * 95**e) // (100**e))
var EblaYieldTable = [38]uint64{
	70000, // epoch  0:  7.0000%   (initial)
	66500, // epoch  1:  6.6500%
	63175, // epoch  2:  6.3175%
	60016, // epoch  3:  6.0016%
	57015, // epoch  4:  5.7015%
	54164, // epoch  5:  5.4164%
	51456, // epoch  6:  5.1456%
	48883, // epoch  7:  4.8883%
	46439, // epoch  8:  4.6439%
	44117, // epoch  9:  4.4117%
	41911, // epoch 10:  4.1911%
	39816, // epoch 11:  3.9816%
	37825, // epoch 12:  3.7825%
	35933, // epoch 13:  3.5933%
	34137, // epoch 14:  3.4137%
	32430, // epoch 15:  3.2430%
	30808, // epoch 16:  3.0808%
	29268, // epoch 17:  2.9268%
	27805, // epoch 18:  2.7805%
	26414, // epoch 19:  2.6414%
	25094, // epoch 20:  2.5094%
	23839, // epoch 21:  2.3839%
	22647, // epoch 22:  2.2647%
	21514, // epoch 23:  2.1514%
	20439, // epoch 24:  2.0439%
	19417, // epoch 25:  1.9417%
	18446, // epoch 26:  1.8446%
	17524, // epoch 27:  1.7524%
	16647, // epoch 28:  1.6647%
	15815, // epoch 29:  1.5815%
	15024, // epoch 30:  1.5024%
	14273, // epoch 31:  1.4273%
	13559, // epoch 32:  1.3559%
	12881, // epoch 33:  1.2881%
	12237, // epoch 34:  1.2237%
	11625, // epoch 35:  1.1625%
	11044, // epoch 36:  1.1044%
	10492, // epoch 37:  1.0492%   (last natural-decay value above floor)
	// Epoch 38+: served by MinYieldValue (10000 = 1.0000%) in calculateCurrentYield
}

type YieldCurve struct {
	cfg        chain_config.ChainConfig
	max_supply *uint256.Int
}

func (self *YieldCurve) Init(cfg chain_config.ChainConfig) {
	self.cfg = cfg

	max_supply, overflow := uint256.FromBig(self.cfg.Protocol.Supply.MaxSupply)
	asserts.Holds(overflow == false, "YieldCurve max supply overflow")
	self.max_supply = max_supply

	// BlocksPerYear must not be zero — would cause division by zero
	asserts.Holds(cfg.DPOS.BlocksPerYear > 0, "BlocksPerYear must be > 0")
}

// GetMaxSupply returns a copy of max_supply for external cap checks
func (self *YieldCurve) GetMaxSupply() *uint256.Int {
	return new(uint256.Int).Set(self.max_supply)
}

// calculateCurrentYield returns the yield rate for the given block.
//
// O(1) lookup against EblaYieldTable. Pure function: no storage, no mutation.
// For epoch > MaxEpoch, returns the MinYieldValue floor (1.0000%).
//
// Determinism: identical answer on all honest validators given identical
// block_num input. No floating point, no system calls, no shared state.
func (self *YieldCurve) calculateCurrentYield(block_num uint64) *uint256.Int {
	epoch := block_num / EpochLength

	if epoch > MaxEpoch {
		return uint256.NewInt(MinYieldValue)
	}
	return uint256.NewInt(EblaYieldTable[epoch])
}

// CalculateBlockReward computes the per-block reward.
//
// Formula: block_reward = yield × total_delegation / (precision × blocks_per_year)
//
// Parameters:
//   - current_total_delegation: total staked across all validators
//   - current_total_ebla_supply: current total supply (unused in new model, kept for interface compat)
//   - block_num: current block number (determines epoch)
func (self *YieldCurve) CalculateBlockReward(current_total_delegation *uint256.Int, current_total_ebla_supply *uint256.Int, block_num uint64) (block_reward *uint256.Int, yield *uint256.Int) {
	yield = self.calculateCurrentYield(block_num)

	block_reward = new(uint256.Int).Mul(current_total_delegation, yield)
	blocks_per_year := uint256.NewInt(uint64(self.cfg.DPOS.BlocksPerYear))
	block_reward.Div(block_reward, new(uint256.Int).Mul(YieldFractionDecimalPrecision, blocks_per_year))

	return
}

// CalculateTotalSupply computes total supply from genesis balances + minted tokens + generated rewards.
// Initializes total_supply from genesis balances + legacy minted_tokens counter + generated rewards.
func (self *YieldCurve) CalculateTotalSupply(minted_tokens *uint256.Int) *uint256.Int {
	total_supply := bigutil.Add(self.cfg.GenesisBalancesSum(), minted_tokens.ToBig())
	total_supply.Add(total_supply, self.cfg.Protocol.Supply.GeneratedRewards)

	total_supply_uint256, overflow := uint256.FromBig(total_supply)
	asserts.Holds(overflow == false, "CalculateTotalSupply: Genesis balances sum overflow")

	return total_supply_uint256
}
