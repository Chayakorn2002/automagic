package enum

// RunType represents the different execution modes for automagic
type RunType string

const (
	// RunTypeNormal executes Claude with full functionality
	RunTypeNormal RunType = "normal"
	
	// RunTypeDryRun shows what would happen without making any changes
	RunTypeDryRun RunType = "dry-run"
	
	// RunTypeSemiDryRun clones repository and shows prompt without executing Claude
	RunTypeSemiDryRun RunType = "semi-dry-run"
)

// IsValid checks if the run type is valid
func (rt RunType) IsValid() bool {
	switch rt {
	case RunTypeNormal, RunTypeDryRun, RunTypeSemiDryRun:
		return true
	default:
		return false
	}
}

// IsDryRun returns true if this is any type of dry run
func (rt RunType) IsDryRun() bool {
	return rt == RunTypeDryRun || rt == RunTypeSemiDryRun
}

// RequiresRepositoryClone returns true if repository should be cloned/verified
func (rt RunType) RequiresRepositoryClone() bool {
	return rt == RunTypeNormal || rt == RunTypeSemiDryRun
}

// ShouldExecuteClaude returns true if Claude should be executed
func (rt RunType) ShouldExecuteClaude() bool {
	return rt == RunTypeNormal
}

// GetValidValues returns all valid run type values
func GetValidValues() []RunType {
	return []RunType{RunTypeNormal, RunTypeDryRun, RunTypeSemiDryRun}
}

// String returns the string representation of the run type
func (rt RunType) String() string {
	return string(rt)
}

// ParseRunType parses a string into a RunType
func ParseRunType(s string) (RunType, bool) {
	rt := RunType(s)
	return rt, rt.IsValid()
}