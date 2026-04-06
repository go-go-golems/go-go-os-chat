package profilechat

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	gepprofiles "github.com/go-go-golems/geppetto/pkg/engineprofiles"
	infruntime "github.com/go-go-golems/pinocchio/pkg/inference/runtime"
	webhttp "github.com/go-go-golems/pinocchio/pkg/webchat/http"
	"github.com/stretchr/testify/require"
)

func TestStrictRequestResolver_WSRequiresConvID(t *testing.T) {
	r := NewStrictRequestResolver("inventory")
	req := httptest.NewRequest(http.MethodGet, "/ws", nil)

	_, err := r.Resolve(req)
	require.Error(t, err)

	var re *webhttp.RequestResolutionError
	require.ErrorAs(t, err, &re)
	require.Equal(t, http.StatusBadRequest, re.Status)
}

func TestStrictRequestResolver_ChatUsesTextFallback(t *testing.T) {
	r := NewStrictRequestResolver("inventory")
	req := httptest.NewRequest(http.MethodPost, "/chat", strings.NewReader(`{"text":"hello"}`))

	plan, err := r.Resolve(req)
	require.NoError(t, err)
	require.Equal(t, "hello", plan.Prompt)
	require.NotEmpty(t, plan.ConvID)
	require.Equal(t, "inventory", plan.RuntimeKey)
}

func TestStrictRequestResolver_ChatUsesProfileSelection(t *testing.T) {
	r := newResolverWithProfiles(t)
	req := httptest.NewRequest(http.MethodPost, "/chat", strings.NewReader(`{"text":"hello","profile":"analyst"}`))

	plan, err := r.Resolve(req)
	require.NoError(t, err)
	require.Equal(t, "hello", plan.Prompt)
	require.Equal(t, "analyst", plan.RuntimeKey)
	require.Equal(t, uint64(7), plan.ProfileVersion)
	require.NotNil(t, plan.ResolvedRuntime)
	require.Equal(t, "Analyst system", plan.ResolvedRuntime.SystemPrompt)
}

func TestStrictRequestResolver_UsesConfiguredDefaultProfileSelectionWhenRequestOmitsProfile(t *testing.T) {
	r := newResolverWithProfiles(t).WithDefaultProfileSelection(gepprofiles.MustEngineProfileSlug("analyst"))
	req := httptest.NewRequest(http.MethodPost, "/chat", strings.NewReader(`{"text":"hello"}`))

	plan, err := r.Resolve(req)
	require.NoError(t, err)
	require.Equal(t, "analyst", plan.RuntimeKey)
	require.Equal(t, uint64(7), plan.ProfileVersion)
	require.NotNil(t, plan.ResolvedRuntime)
	require.Equal(t, "Analyst system", plan.ResolvedRuntime.SystemPrompt)
}

func TestStrictRequestResolver_WSUsesProfileQuerySelection(t *testing.T) {
	r := newResolverWithProfiles(t)
	req := httptest.NewRequest(http.MethodGet, "/ws?conv_id=conv-1&profile=analyst", nil)

	plan, err := r.Resolve(req)
	require.NoError(t, err)
	require.Equal(t, "analyst", plan.RuntimeKey)
	require.Equal(t, uint64(7), plan.ProfileVersion)
}

func TestStrictRequestResolver_WSIgnoresLegacyCookieProfileSelection(t *testing.T) {
	r := newResolverWithProfiles(t)
	req := httptest.NewRequest(http.MethodGet, "/ws?conv_id=conv-1", nil)
	req.AddCookie(&http.Cookie{Name: "chat_profile", Value: "analyst"})

	plan, err := r.Resolve(req)
	require.NoError(t, err)
	require.Equal(t, "inventory", plan.RuntimeKey)
	require.Equal(t, uint64(3), plan.ProfileVersion)
}

func TestStrictRequestResolver_UnknownProfileReturnsNotFound(t *testing.T) {
	r := newResolverWithProfiles(t)
	req := httptest.NewRequest(http.MethodPost, "/chat", strings.NewReader(`{"prompt":"hi","profile":"missing"}`))

	_, err := r.Resolve(req)
	require.Error(t, err)
	var re *webhttp.RequestResolutionError
	require.ErrorAs(t, err, &re)
	require.Equal(t, http.StatusNotFound, re.Status)
}

func TestStrictRequestResolver_InvalidProfileReturnsBadRequest(t *testing.T) {
	r := newResolverWithProfiles(t)
	req := httptest.NewRequest(http.MethodPost, "/chat", strings.NewReader(`{"prompt":"hi","profile":"invalid runtime key!"}`))

	_, err := r.Resolve(req)
	require.Error(t, err)
	var re *webhttp.RequestResolutionError
	require.ErrorAs(t, err, &re)
	require.Equal(t, http.StatusBadRequest, re.Status)
}

func TestStrictRequestResolver_UnknownRegistryQueryReturnsNotFound(t *testing.T) {
	r := newResolverWithProfiles(t)
	req := httptest.NewRequest(http.MethodPost, "/chat?registry=missing", strings.NewReader(`{"prompt":"hi"}`))

	_, err := r.Resolve(req)
	require.Error(t, err)
	var re *webhttp.RequestResolutionError
	require.ErrorAs(t, err, &re)
	require.Equal(t, http.StatusNotFound, re.Status)
}

func TestStrictRequestResolver_InvalidRegistryInBodyReturnsBadRequest(t *testing.T) {
	r := newResolverWithProfiles(t)
	req := httptest.NewRequest(http.MethodPost, "/chat", strings.NewReader(`{"prompt":"hi","registry":"invalid registry!","profile":"analyst"}`))

	_, err := r.Resolve(req)
	require.Error(t, err)
	var re *webhttp.RequestResolutionError
	require.ErrorAs(t, err, &re)
	require.Equal(t, http.StatusBadRequest, re.Status)
}

func TestStrictRequestResolver_LegacyRegistrySlugQueryReturnsBadRequest(t *testing.T) {
	r := newResolverWithProfiles(t)
	req := httptest.NewRequest(http.MethodPost, "/chat?registry_slug=missing", strings.NewReader(`{"prompt":"hi"}`))

	_, err := r.Resolve(req)
	require.Error(t, err)
	var re *webhttp.RequestResolutionError
	require.ErrorAs(t, err, &re)
	require.Equal(t, http.StatusBadRequest, re.Status)
}

func TestStrictRequestResolver_LegacyRegistrySlugBodyReturnsBadRequest(t *testing.T) {
	r := newResolverWithProfiles(t)
	req := httptest.NewRequest(http.MethodPost, "/chat", strings.NewReader(`{"prompt":"hi","registry_slug":"invalid registry!","profile":"analyst"}`))

	_, err := r.Resolve(req)
	require.Error(t, err)
	var re *webhttp.RequestResolutionError
	require.ErrorAs(t, err, &re)
	require.Equal(t, http.StatusBadRequest, re.Status)
}

func TestStrictRequestResolver_LegacyRuntimeKeyBodyReturnsBadRequest(t *testing.T) {
	r := newResolverWithProfiles(t)
	req := httptest.NewRequest(http.MethodPost, "/chat", strings.NewReader(`{"prompt":"hi","runtime_key":"analyst"}`))

	_, err := r.Resolve(req)
	require.Error(t, err)
	var re *webhttp.RequestResolutionError
	require.ErrorAs(t, err, &re)
	require.Equal(t, http.StatusBadRequest, re.Status)
}

func TestStrictRequestResolver_RequestOverridesAreValidatedByPolicy(t *testing.T) {
	r := newResolverWithProfiles(t)
	req := httptest.NewRequest(http.MethodPost, "/chat", strings.NewReader(`{"prompt":"hi","profile":"inventory","request_overrides":{"system_prompt":"override"}}`))

	_, err := r.Resolve(req)
	require.Error(t, err)
	var re *webhttp.RequestResolutionError
	require.ErrorAs(t, err, &re)
	require.Equal(t, http.StatusBadRequest, re.Status)
}

func TestStrictRequestResolver_ResolveRuntimePlanMapsValidationErrorsToBadRequest(t *testing.T) {
	r := NewStrictRequestResolver("inventory").WithProfileRegistry(runtimePlanErrorRegistry{
		getErr: &gepprofiles.ValidationError{
			Field:  "profile.extensions[pinocchio.runtime]",
			Reason: "invalid middleware payload",
		},
	}, gepprofiles.MustRegistrySlug("default"))

	_, err := r.resolveRuntimePlan(context.Background(), &gepprofiles.ResolvedEngineProfile{
		RegistrySlug:      gepprofiles.MustRegistrySlug("default"),
		EngineProfileSlug: gepprofiles.MustEngineProfileSlug("inventory"),
		StackLineage: []gepprofiles.ResolvedProfileStackEntry{
			{
				RegistrySlug:      gepprofiles.MustRegistrySlug("default"),
				EngineProfileSlug: gepprofiles.MustEngineProfileSlug("inventory"),
			},
		},
	})
	require.Error(t, err)

	var re *webhttp.RequestResolutionError
	require.ErrorAs(t, err, &re)
	require.Equal(t, http.StatusBadRequest, re.Status)
	require.Equal(t, "invalid pinocchio runtime extension", re.ClientMsg)
}

func TestStrictRequestResolver_ResolveRuntimePlanPreservesNonValidationErrors(t *testing.T) {
	expectedErr := errors.New("registry unavailable")
	r := NewStrictRequestResolver("inventory").WithProfileRegistry(runtimePlanErrorRegistry{
		getErr: expectedErr,
	}, gepprofiles.MustRegistrySlug("default"))

	_, err := r.resolveRuntimePlan(context.Background(), &gepprofiles.ResolvedEngineProfile{
		RegistrySlug:      gepprofiles.MustRegistrySlug("default"),
		EngineProfileSlug: gepprofiles.MustEngineProfileSlug("inventory"),
		StackLineage: []gepprofiles.ResolvedProfileStackEntry{
			{
				RegistrySlug:      gepprofiles.MustRegistrySlug("default"),
				EngineProfileSlug: gepprofiles.MustEngineProfileSlug("inventory"),
			},
		},
	})
	require.Error(t, err)
	require.ErrorIs(t, err, expectedErr)

	var re *webhttp.RequestResolutionError
	require.False(t, errors.As(err, &re))
}

func newResolverWithProfiles(t *testing.T) *StrictRequestResolver {
	t.Helper()

	store := gepprofiles.NewInMemoryEngineProfileStore()
	registry := &gepprofiles.EngineProfileRegistry{
		Slug:                     gepprofiles.MustRegistrySlug("default"),
		DefaultEngineProfileSlug: gepprofiles.MustEngineProfileSlug("inventory"),
		Profiles: map[gepprofiles.EngineProfileSlug]*gepprofiles.EngineProfile{
			gepprofiles.MustEngineProfileSlug("inventory"): testEngineProfileWithRuntime(t, "inventory", 3, "Inventory system"),
			gepprofiles.MustEngineProfileSlug("analyst"):   testEngineProfileWithRuntime(t, "analyst", 7, "Analyst system"),
		},
	}
	require.NoError(t, gepprofiles.ValidateRegistry(registry))
	require.NoError(t, store.UpsertRegistry(context.Background(), registry, gepprofiles.SaveOptions{
		Actor:  "test",
		Source: "test",
	}))

	svc, err := gepprofiles.NewStoreRegistry(store, gepprofiles.MustRegistrySlug("default"))
	require.NoError(t, err)
	return NewStrictRequestResolver("inventory").WithProfileRegistry(svc, gepprofiles.MustRegistrySlug("default"))
}

func testEngineProfileWithRuntime(t *testing.T, slug string, version uint64, systemPrompt string) *gepprofiles.EngineProfile {
	t.Helper()

	profile := &gepprofiles.EngineProfile{
		Slug:     gepprofiles.MustEngineProfileSlug(slug),
		Metadata: gepprofiles.EngineProfileMetadata{Version: version},
	}
	require.NoError(t, infruntime.SetProfileRuntime(profile, &infruntime.ProfileRuntime{
		SystemPrompt: systemPrompt,
	}))
	return profile
}

type runtimePlanErrorRegistry struct {
	getErr error
}

func (r runtimePlanErrorRegistry) ListRegistries(context.Context) ([]gepprofiles.RegistrySummary, error) {
	return nil, nil
}

func (r runtimePlanErrorRegistry) GetRegistry(context.Context, gepprofiles.RegistrySlug) (*gepprofiles.EngineProfileRegistry, error) {
	return nil, nil
}

func (r runtimePlanErrorRegistry) ListEngineProfiles(context.Context, gepprofiles.RegistrySlug) ([]*gepprofiles.EngineProfile, error) {
	return nil, nil
}

func (r runtimePlanErrorRegistry) GetEngineProfile(context.Context, gepprofiles.RegistrySlug, gepprofiles.EngineProfileSlug) (*gepprofiles.EngineProfile, error) {
	return nil, r.getErr
}

func (r runtimePlanErrorRegistry) ResolveEngineProfile(context.Context, gepprofiles.ResolveInput) (*gepprofiles.ResolvedEngineProfile, error) {
	return nil, nil
}
