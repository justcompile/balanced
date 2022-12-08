package k8s

import (
	"fmt"
)

type IgnoreService struct {
	service string
	reason  string
}

func (i *IgnoreService) Error() string {
	return fmt.Sprintf("service %s ignored due to: %s", i.service, i.reason)
}
