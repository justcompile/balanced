package types

type Processor interface {
	Start(changes chan *Change)
	OnExit() error
}
