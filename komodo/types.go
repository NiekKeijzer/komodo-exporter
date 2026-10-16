package komodo

// Server is a ServerListItem.
type Server struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Info struct {
		// State is "Ok", "NotOk" or "Disabled".
		State string `json:"state"`
		// Version is the Periphery version, empty when unknown.
		Version string `json:"version"`
	} `json:"info"`
}

// Server states.
const (
	ServerOk       = "Ok"
	ServerDisabled = "Disabled"
)

// SystemStats are the system stats of a server. Komodo reports sizes in GiB.
type SystemStats struct {
	CPUPerc    float64 `json:"cpu_perc"`
	MemUsedGB  float64 `json:"mem_used_gb"`
	MemTotalGB float64 `json:"mem_total_gb"`
	Disks      []struct {
		Mount   string  `json:"mount"`
		UsedGB  float64 `json:"used_gb"`
		TotalGB float64 `json:"total_gb"`
	} `json:"disks"`
}

// Stack is a StackListItem.
type Stack struct {
	Name string `json:"name"`
	Info struct {
		// State is a snake_case StackState, such as "running" or "down".
		State          string `json:"state"`
		ServerName     string `json:"server_name"`
		ProjectMissing bool   `json:"project_missing"`
	} `json:"info"`
}

// Container is a ContainerListItem.
type Container struct {
	Name       string `json:"name"`
	ServerName string `json:"server_name"`
	// State is a lowercase Docker state, such as "running" or "exited".
	State string `json:"state"`
	// Status is Docker's status text, such as "Up 2 hours (healthy)".
	Status string `json:"status"`
	// Stats are the strings `docker stats` prints, nil when unavailable.
	Stats *struct {
		CPUPerc  string `json:"cpu_perc"`
		MemUsage string `json:"mem_usage"`
	} `json:"stats"`
}

// Resource is a list item of a resource whose info has a state: a resource
// sync, a procedure or a repo. The state is a variant name such as "Ok" or
// "Failed".
type Resource struct {
	Name string `json:"name"`
	Info struct {
		State string `json:"state"`
	} `json:"info"`
}
