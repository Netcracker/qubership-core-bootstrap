package getters

import (
	"testing"
)

type mockGenerator struct {
	name      string
	generated bool
}

func (m *mockGenerator) Name() string {
	return m.name
}

func (m *mockGenerator) Generate() {
	m.generated = true
}

func TestGeneratorManager_RegisterAndRun(t *testing.T) {
	gm := &GeneratorManager{generators: make(map[string]Generator)}
	gen1 := &mockGenerator{name: "gen1"}
	gen2 := &mockGenerator{name: "gen2"}

	gm.register(gen1)
	gm.register(gen2)

	if len(gm.generators) != 2 {
		t.Errorf("expected 2 generators, got %d", len(gm.generators))
	}

	gm.run()

	if !gen1.generated {
		t.Errorf("gen1 was not generated")
	}
	if !gen2.generated {
		t.Errorf("gen2 was not generated")
	}
}
