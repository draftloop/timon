package enums

type ContractType string

const (
	ContractTypeProbe ContractType = "probe"
	ContractTypeJob   ContractType = "job"
)

type ContractRule string

const (
	ContractRuleStale               ContractRule = "stale"
	ContractRuleStaleIncident       ContractRule = "stale-incident"
	ContractRuleJobOvertimeIncident ContractRule = "overtime-incident"
	ContractRuleJobOverlapIncident  ContractRule = "overlap-incident"
)
