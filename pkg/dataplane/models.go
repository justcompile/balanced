package dataplane

type Backend struct {
	Balance   Balance   `json:"balance"`
	Mode      string    `json:"mode"`
	Name      string    `json:"name"`
	HTTPCheck HTTPCheck `json:"http-check"`
}

type Balance struct {
	Algorithm string `json:"algorithm"`
}

type Header struct {
	Key   string `json:"name"`
	Value string `json:"fmt"`
}

type HTTPCheck struct {
	Headers []Header `json:"headers"`
	Index   int      `json:"index"`
	Method  string   `json:"method"`
	Type    string   `json:"type"`
	Uri     string   `json:"uri"`
}

type Server struct {
	Check      string `json:"check"`
	CheckSSL   string `json:"check-ssl"`
	Identifier string `json:"name"`
	Address    string `json:"address"`
	Port       int    `json:"port"`
}

type VersionedResponsed[T any] struct {
	Data    T   `json:"data"`
	Version int `json:"_version"`
}
