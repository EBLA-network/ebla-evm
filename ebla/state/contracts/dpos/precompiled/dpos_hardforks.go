package dpos

import (
	"fmt"

	"github.com/EBLA-network/ebla-evm/core/types"
	"github.com/EBLA-network/ebla-evm/core/vm"
	dpos_sol "github.com/EBLA-network/ebla-evm/ebla/state/contracts/dpos/solidity"
	"github.com/EBLA-network/ebla-evm/ebla/util"
	"github.com/EBLA-network/ebla-evm/ebla/util/bigutil"
	"github.com/holiman/uint256"
)

// Pays off accumulated rewards back to delegator address from multiple validators at a time
// NOTE this is old PRE-ASPEN HF version
func (self *Contract) claimAllRewardsPreAspenHF(ctx vm.CallFrame, block types.BlockNum, args dpos_sol.ClaimAllRewardsArgs) (end bool, err error) {
	delegator_validators_addresses, end := self.delegations.GetDelegatorValidatorsAddresses(ctx.CallerAccount.Address(), args.Batch, ClaimAllRewardsMaxCount)
	var tmp_claim_rewards_args dpos_sol.ValidatorAddressArgs
	for _, validator_address := range delegator_validators_addresses {
		tmp_claim_rewards_args.Validator = validator_address

		tmp_err := self.claimRewards(ctx, block, tmp_claim_rewards_args)
		if tmp_err != nil {
			err = util.ErrorString(tmp_err.Error() + " -> validator: " + validator_address.String())
			return
		}
	}

	err = nil
	return
}

func (self *Contract) fixRedelegateBlockNumFunc(block_num uint64) {
	for _, redelegation := range self.cfg.Hardforks.Redelegations {
		delegation := self.delegations.GetDelegation(&redelegation.Delegator, &redelegation.Validator)

		val := self.validators.GetValidator(&redelegation.Validator)

		fmt.Println("Applying HF on validator", redelegation.Validator.String(), "delegator", redelegation.Delegator.String())

		state, _ := self.state_get(redelegation.Validator[:], BlockToBytes(delegation.LastUpdated))
		wrong_state, _ := self.state_get(redelegation.Validator[:], BlockToBytes(val.LastUpdated))
		if wrong_state != nil || state == nil {
			panic("HF on wrong account")
		}

		fmt.Println("Fixing block from", val.LastUpdated, "to", delegation.LastUpdated)

		// Corrected block num
		val.LastUpdated = delegation.LastUpdated
		val.TotalStake = bigutil.Sub(val.TotalStake, redelegation.Amount)
		self.validators.ModifyValidator(true, &redelegation.Validator, val)
	}
}

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

// func (self *Contract) bambooHFRedelegation(block_num uint64) {
// 	for _, redelegation := range self.cfg.Hardforks.BambooHf.Redelegations {
// 		val := self.validators.GetValidator(&redelegation.Validator)
// 		if val == nil {
// 			panic("Validator not found")
// 		}
// 		val_rewards := self.validators.GetValidatorRewards(&redelegation.Validator)
// 		if val_rewards == nil {
// 			panic("Validator rewards not found")
// 		}

// 		fmt.Println("Applying Bamboo HF on validator", redelegation.Validator.String(), "amount", redelegation.Amount.String())
// 		val.TotalStake = bigutil.Sub(val.TotalStake, redelegation.Amount)

// 		if val.TotalStake.Cmp(big.NewInt(0)) == 0 && val_rewards.CommissionRewardsPool.Cmp(big.NewInt(0)) == 0 {
// 			self.validators.DeleteValidator(&redelegation.Validator)
// 			fmt.Println("Deleted validator", redelegation.Validator.String())
// 			state, stake_k := self.state_get(redelegation.Validator[:], BlockToBytes(val.LastUpdated))
// 			if state != nil {
// 				self.state_put(&stake_k, nil)
// 			}
// 		} else {
// 			old_state := self.state_get_and_decrement(redelegation.Validator[:], BlockToBytes(val.LastUpdated))
// 			state, state_k := self.state_get(redelegation.Validator[:], BlockToBytes(block_num))
// 			if state == nil {
// 				state = new(State)
// 				if val.TotalStake.Cmp(big.NewInt(0)) > 0 {
// 					state.RewardsPer1Stake = bigutil.Add(old_state.RewardsPer1Stake, self.calculateRewardPer1Stake(val_rewards.RewardsPool, val.TotalStake))
// 				} else {
// 					state.RewardsPer1Stake = old_state.RewardsPer1Stake
// 				}

// 				val_rewards.RewardsPool = big.NewInt(0)
// 				val.LastUpdated = block_num
// 				state.Count++
// 			}
// 			self.state_put(&state_k, state)
// 			self.validators.ModifyValidator(self.isMagnoliaHardfork(block_num), &redelegation.Validator, val)
// 			self.validators.ModifyValidatorRewards(&redelegation.Validator, val_rewards)
// 			fmt.Println("Updated validator", redelegation.Validator.String())
// 		}
// 	}
// }
