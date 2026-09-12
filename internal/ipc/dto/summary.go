package dto

type MotdRequest struct{}

type MotdResponse struct {
	ActiveIncidents    int
	CriticalContracts  []string
	StaleContracts     []string
	NbWarningContracts int
	NbHealthyContracts int
	NbRunningJobs      int
}
