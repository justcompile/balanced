package dns

type Registrar interface {
	Add(string) error
	Remove(string) error
	RemoveAll() error
}
