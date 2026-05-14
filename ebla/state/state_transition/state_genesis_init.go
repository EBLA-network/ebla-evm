package state_transition

import (
	dpos "github.com/EBLA-network/ebla-evm/ebla/state/contracts/dpos/precompiled"
	dpos_sol "github.com/EBLA-network/ebla-evm/ebla/state/contracts/dpos/solidity"
)

// applyGenesisInit registers the DPoS and Slashing precompiles with the EVM and deploys
// the DPoS Solidity wrapper bytecode at the precompile address. Runs once at genesis
// (the only time SetBlock reports rules_changed=true since Phase 14.3 collapsed all
// per-block rule flags to an empty vm.Rules{}).
func (st *StateTransition) applyGenesisInit() {
	if st.dpos_contract != nil {
		st.dpos_contract.Register(st.evm.RegisterPrecompiledContract)
		acc := st.state.GetAccount(dpos.ContractAddress())
		if acc.GetCodeSize() == 0 {
			acc.SetCode(dpos_sol.DposImplBytecode)
		}
	}

	if st.slashing_contract != nil {
		st.slashing_contract.Register(st.evm.RegisterPrecompiledContract)
	}

}
