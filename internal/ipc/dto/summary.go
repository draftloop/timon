package dto

type SummaryRequest struct{}

type SummaryResponse struct {
	ActiveIncidents    int
	CriticalContracts  []string
	StaleContracts     []string
	NbWarningContracts int
	NbHealthyContracts int
	NbRunningJobs      int
}
