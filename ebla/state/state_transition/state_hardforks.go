package state_transition

import (
	dpos "github.com/EBLA-network/ebla-evm/ebla/state/contracts/dpos/precompiled"
	dpos_sol "github.com/EBLA-network/ebla-evm/ebla/state/contracts/dpos/solidity"
	"github.com/EBLA-network/ebla-evm/ebla/state/state_transition/op_stack"
)

func (st *StateTransition) applyHFChanges() {
	blk_n := st.BlockNumber()

	if st.dpos_contract != nil {
		st.dpos_contract.Register(st.evm.RegisterPrecompiledContract)
		if st.chain_config.Hardforks.IsOnAspenHardforkPartOne(blk_n) {
			acc := st.state.GetAccount(dpos.ContractAddress())
			if acc.GetCodeSize() == 0 {
				acc.SetCode(dpos_sol.AspenDposImplBytecode)
			}
		}
	}

	if st.slashing_contract != nil && st.chain_config.Hardforks.IsOnMagnoliaHardfork(blk_n) {
		st.slashing_contract.Register(st.evm.RegisterPrecompiledContract)
	}

}
