package dbaas

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreateDatabaseIsSkippedWhenDbaasOperatorIsEnabled(t *testing.T) {
	configurer := &Configurer{dbaasOperatorEnabled: true}

	assert.NoError(t, configurer.CreateDatabase(context.Background(), "control-plane", "legacy-secret", nil))
}
