package k8s

import "fmt"

type ignoreService struct {
	service string
	reason  string
}

func (i *ignoreService) Error() string {
	return fmt.Sprintf("service %s ignored due to: %s", i.service, i.reason)
}
