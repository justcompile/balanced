package cloud

const (
	SecurityGroupTag = "balanced:managed"
)

type CloudProvider interface {
	GetAddresses() ([]string, error)
	UpsertRecordSet([]string) error
}

type SecurityGroup struct {
	Id    string
	Ports []int32
}
