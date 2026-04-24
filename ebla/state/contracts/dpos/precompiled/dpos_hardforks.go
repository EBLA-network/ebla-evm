package dpos

import (
	"github.com/holiman/uint256"
)

func (self *Contract) processBlockReward(block_num uint64) *uint256.Int {
	if self.total_supply == nil {
		self.total_supply = self.yield_curve.CalculateTotalSupply(self.minted_tokens)
		self.saveTotalSupplyDb()
		self.eraseMintedTokensDb()
	}

	// Calculate block reward using epoch-based yield decay
	blockReward, yield := self.yield_curve.CalculateBlockReward(self.amount_delegated, self.total_supply, block_num)

	// CRITICAL: Enforce max_supply cap.
	// The epoch decay model has a 1% yield floor — rewards never naturally stop.
	// This cap ensures total_supply never exceeds max_supply (12B EBLA).
	max_supply := self.yield_curve.GetMaxSupply()
	if self.total_supply.Cmp(max_supply) >= 0 {
		self.saveYieldDb(uint64(0))
		return uint256.NewInt(0)
	}
	remaining := new(uint256.Int).Sub(max_supply, self.total_supply)
	if blockReward.Cmp(remaining) > 0 {
		blockReward = remaining.Clone()
	}

	self.saveYieldDb(yield.Uint64())
	return blockReward
}
