package loadbalancer

import (
	"balanced/pkg/types"
	"io"
	"text/template"
)

type Renderer struct {
	t *template.Template
}

func (r *Renderer) ToWriter(w io.Writer, obj *types.LoadBalancerUpstreamDefinition) error {
	return r.t.Execute(w, obj)
}

func NewRenderer(templateText string) (*Renderer, error) {
	t, err := template.New("balanced").Parse(templateText)
	if err != nil {
		return nil, err
	}

	return &Renderer{t: t}, nil
}
