package getters

type Generator interface {
	Name() string
	Generate()
}
