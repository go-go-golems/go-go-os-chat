package chatservice

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLifecycleRequiresServer(t *testing.T) {
	component := New(Options{})

	require.Error(t, component.Init(context.Background()))
	require.Error(t, component.Start(context.Background()))
	require.Error(t, component.Health(context.Background()))
	require.NoError(t, component.Stop(context.Background()))
}

func TestMountRoutesValidatesRequiredDependencies(t *testing.T) {
	component := New(Options{})

	require.Error(t, component.MountRoutes(nil))
	require.Error(t, component.MountRoutes(http.NewServeMux()))
}
