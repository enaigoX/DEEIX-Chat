package imagestudio

// Module aggregates Image Studio HTTP handlers.
type Module struct {
	Handler *Handler
}

// NewModule creates an Image Studio HTTP module.
func NewModule(handler *Handler) *Module {
	return &Module{Handler: handler}
}
