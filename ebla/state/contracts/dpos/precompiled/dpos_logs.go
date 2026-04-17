package dpos

import (
	"math/big"

	"github.com/EBLA-network/ebla-evm/accounts/abi"
	"github.com/EBLA-network/ebla-evm/common"
	"github.com/EBLA-network/ebla-evm/core/vm"
)

func checkError(log *vm.LogRecord, err error) *vm.LogRecord {
	if err != nil {
		panic("Update logs methods to correspond ABI: " + err.Error())
	}
	return log
}

type Logs struct {
	Events map[string]abi.Event
}

func (self *Logs) Init(events map[string]abi.Event) *Logs {
	self.Events = events

	return self
}

// All Make functions below are making log records for events.
// All hashes and data types should be the same as we have in solidity interface in ../solidity/dpos_contract_interface.sol
// If some event will be added or changed TestMakeLogsCheckTopics test should be modified

// event Delegated(address indexed delegator, address indexed validator, uint256 amount);
func (self *Logs) MakeDelegatedLog(delegator, validator *common.Address, amount *big.Int) vm.LogRecord {
	event := self.Events["Delegated"]

	return *checkError(event.MakeLog(dpos_contract_address, delegator, validator, amount))
}

// New EBLA - event UndelegatedV2(address indexed delegator, address indexed validator, uint64 undelegation_id, uint256 amount);
func (self *Logs) MakeUndelegatedLog(delegator, validator *common.Address, undelegation_id uint64, amount *big.Int) vm.LogRecord {
	event := self.Events["Undelegated"]

	return *checkError(event.MakeLog(dpos_contract_address, delegator, validator, undelegation_id, amount))
}
//New EBAL
// event UndelegateConfirmedV2(address indexed delegator, address indexed validator, uint64 undelegation_id, uint256 amount);
func (self *Logs) MakeUndelegateConfirmedLog(delegator, validator *common.Address, undelegation_id uint64, amount *big.Int) vm.LogRecord {
	event := self.Events["UndelegateConfirmed"]
	return *checkError(event.MakeLog(dpos_contract_address, delegator, validator, undelegation_id, amount))
}

// New EBLA - event UndelegateCanceledV2(address indexed delegator, address indexed validator, uint64 undelegation_id, uint256 amount);
func (self *Logs) MakeUndelegateCanceledLog(delegator, validator *common.Address, undelegation_id uint64, amount *big.Int) vm.LogRecord {
	event := self.Events["UndelegateCanceled"]
	return *checkError(event.MakeLog(dpos_contract_address, delegator, validator, undelegation_id, amount))
}

// event Redelegated(address indexed delegator, address indexed from, address indexed to, uint256 amount);
func (self *Logs) MakeRedelegatedLog(delegator, from, to *common.Address, amount *big.Int) vm.LogRecord {
	event := self.Events["Redelegated"]

	return *checkError(event.MakeLog(dpos_contract_address, delegator, from, to, amount))
}

// event RewardsClaimed(address indexed account, address indexed validator);
func (self *Logs) MakeRewardsClaimedLog(account, validator *common.Address, amount *big.Int) vm.LogRecord {
	event := self.Events["RewardsClaimed"]

	return *checkError(event.MakeLog(dpos_contract_address, account, validator, amount))
}

// event CommissionRewardsClaimed(address indexed account, address indexed validator);
func (self *Logs) MakeCommissionRewardsClaimedLog(account, validator *common.Address, amount *big.Int) vm.LogRecord {
	event := self.Events["CommissionRewardsClaimed"]

	return *checkError(event.MakeLog(dpos_contract_address, account, validator, amount))
}

// event CommissionSet(address indexed validator, uint16 commission);
func (self *Logs) MakeCommissionSetLog(account *common.Address, amount uint16) vm.LogRecord {
	event := self.Events["CommissionSet"]

	return *checkError(event.MakeLog(dpos_contract_address, account, amount))
}

// event ValidatorRegistered(address indexed validator);
func (self *Logs) MakeValidatorRegisteredLog(account *common.Address) vm.LogRecord {
	event := self.Events["ValidatorRegistered"]

	return *checkError(event.MakeLog(dpos_contract_address, account))
}

// event ValidatorInfoSet(address indexed validator);
func (self *Logs) MakeValidatorInfoSetLog(account *common.Address) vm.LogRecord {
	event := self.Events["ValidatorInfoSet"]

	return *checkError(event.MakeLog(dpos_contract_address, account))
}

func (self *Logs) MakeInactivityPenaltyLog(validator *common.Address, new_factor uint64) vm.LogRecord {
    event := self.Events["InactivityPenalty"]
    return *checkError(event.MakeLog(dpos_contract_address, validator, new_factor))
}

func (self *Logs) MakeValidatorEvictedLog(validator *common.Address) vm.LogRecord {
    event := self.Events["ValidatorEvicted"]
    return *checkError(event.MakeLog(dpos_contract_address, validator))
}

func (self *Logs) MakeVotingPowerRecoveredLog(validator *common.Address) vm.LogRecord {
    event := self.Events["VotingPowerRecovered"]
    return *checkError(event.MakeLog(dpos_contract_address, validator))
}
