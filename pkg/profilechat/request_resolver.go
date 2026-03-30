package profilechat

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	gepprofiles "github.com/go-go-golems/geppetto/pkg/engineprofiles"
	aisettings "github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	infruntime "github.com/go-go-golems/pinocchio/pkg/inference/runtime"
	webhttp "github.com/go-go-golems/pinocchio/pkg/webchat/http"
	"github.com/google/uuid"
)

type StrictRequestResolver struct {
	runtimeKey           string
	profileRegistry      gepprofiles.Registry
	defaultRegistrySlug  gepprofiles.RegistrySlug
	defaultProfileSlug   gepprofiles.EngineProfileSlug
	baseInferenceSetting *aisettings.InferenceSettings
}

type strictChatRequestBody struct {
	Prompt           string         `json:"prompt"`
	Text             string         `json:"text,omitempty"`
	ConvID           string         `json:"conv_id"`
	Profile          string         `json:"profile,omitempty"`
	Registry         string         `json:"registry,omitempty"`
	LegacyRuntimeKey string         `json:"runtime_key,omitempty"`
	LegacyRegistry   string         `json:"registry_slug,omitempty"`
	RequestOverrides map[string]any `json:"request_overrides"`
	IdempotencyKey   string         `json:"idempotency_key,omitempty"`
}

type resolvedConversationPlan struct {
	ConvID         string
	Prompt         string
	IdempotencyKey string
	Runtime        *resolvedConversationRuntime
}

type resolvedConversationRuntime struct {
	SystemPrompt       string
	Middlewares        []infruntime.MiddlewareUse
	ToolNames          []string
	RuntimeKey         string
	RuntimeFingerprint string
	ProfileVersion     uint64
	InferenceSettings  *aisettings.InferenceSettings
	ProfileMetadata    map[string]any
}

func NewStrictRequestResolver(runtimeKey string) *StrictRequestResolver {
	key := strings.TrimSpace(runtimeKey)
	if key == "" {
		key = "default"
	}
	return &StrictRequestResolver{
		runtimeKey:          key,
		defaultRegistrySlug: gepprofiles.MustRegistrySlug("default"),
	}
}

func (r *StrictRequestResolver) WithProfileRegistry(profileRegistry gepprofiles.Registry, registrySlug gepprofiles.RegistrySlug) *StrictRequestResolver {
	if r == nil {
		return nil
	}
	r.profileRegistry = profileRegistry
	if !registrySlug.IsZero() {
		r.defaultRegistrySlug = registrySlug
	}
	return r
}

func (r *StrictRequestResolver) WithBaseInferenceSettings(base *aisettings.InferenceSettings) *StrictRequestResolver {
	if r == nil {
		return nil
	}
	if base == nil {
		r.baseInferenceSetting = nil
		return r
	}
	r.baseInferenceSetting = base.Clone()
	return r
}

func (r *StrictRequestResolver) WithDefaultProfileSelection(profileSlug gepprofiles.EngineProfileSlug) *StrictRequestResolver {
	if r == nil {
		return nil
	}
	r.defaultProfileSlug = profileSlug
	return r
}

func (r *StrictRequestResolver) Resolve(req *http.Request) (webhttp.ResolvedConversationRequest, error) {
	if req == nil {
		return webhttp.ResolvedConversationRequest{}, &webhttp.RequestResolutionError{
			Status:    http.StatusBadRequest,
			ClientMsg: "bad request",
		}
	}

	switch req.Method {
	case http.MethodGet:
		return r.resolveWS(req)
	case http.MethodPost:
		return r.resolveChat(req)
	default:
		return webhttp.ResolvedConversationRequest{}, &webhttp.RequestResolutionError{
			Status:    http.StatusMethodNotAllowed,
			ClientMsg: "method not allowed",
		}
	}
}

func (r *StrictRequestResolver) resolveWS(req *http.Request) (webhttp.ResolvedConversationRequest, error) {
	convID := strings.TrimSpace(req.URL.Query().Get("conv_id"))
	if convID == "" {
		return webhttp.ResolvedConversationRequest{}, &webhttp.RequestResolutionError{
			Status:    http.StatusBadRequest,
			ClientMsg: "missing conv_id",
		}
	}

	if err := rejectLegacySelectionFields(req, "", ""); err != nil {
		return webhttp.ResolvedConversationRequest{}, err
	}

	if r.profileRegistry == nil {
		return webhttp.ResolvedConversationRequest{
			ConvID:     convID,
			RuntimeKey: r.runtimeKey,
		}, nil
	}

	profileSlug, err := r.resolveProfileSelection(req, "")
	if err != nil {
		return webhttp.ResolvedConversationRequest{}, err
	}
	registrySlug, err := r.resolveRegistrySelection(req, "")
	if err != nil {
		return webhttp.ResolvedConversationRequest{}, err
	}
	resolvedProfile, err := r.resolveEffectiveProfile(context.Background(), registrySlug, profileSlug)
	if err != nil {
		return webhttp.ResolvedConversationRequest{}, err
	}
	plan, err := r.buildConversationPlan(context.Background(), convID, "", "", resolvedProfile)
	if err != nil {
		return webhttp.ResolvedConversationRequest{}, err
	}
	return toResolvedConversationRequest(plan), nil
}

func (r *StrictRequestResolver) resolveChat(req *http.Request) (webhttp.ResolvedConversationRequest, error) {
	var body strictChatRequestBody
	if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
		return webhttp.ResolvedConversationRequest{}, &webhttp.RequestResolutionError{
			Status:    http.StatusBadRequest,
			ClientMsg: "bad request",
			Err:       err,
		}
	}
	if body.Prompt == "" && body.Text != "" {
		body.Prompt = body.Text
	}
	if err := rejectLegacySelectionFields(req, body.LegacyRuntimeKey, body.LegacyRegistry); err != nil {
		return webhttp.ResolvedConversationRequest{}, err
	}
	if len(body.RequestOverrides) > 0 {
		return webhttp.ResolvedConversationRequest{}, &webhttp.RequestResolutionError{
			Status:    http.StatusBadRequest,
			ClientMsg: "unsupported request_overrides",
		}
	}

	convID := strings.TrimSpace(body.ConvID)
	if convID == "" {
		convID = uuid.NewString()
	}

	if r.profileRegistry == nil {
		return webhttp.ResolvedConversationRequest{
			ConvID:         convID,
			RuntimeKey:     r.runtimeKey,
			Prompt:         body.Prompt,
			IdempotencyKey: strings.TrimSpace(body.IdempotencyKey),
		}, nil
	}

	profileSlug, err := r.resolveProfileSelection(req, body.Profile)
	if err != nil {
		return webhttp.ResolvedConversationRequest{}, err
	}
	registrySlug, err := r.resolveRegistrySelection(req, body.Registry)
	if err != nil {
		return webhttp.ResolvedConversationRequest{}, err
	}
	resolvedProfile, err := r.resolveEffectiveProfile(context.Background(), registrySlug, profileSlug)
	if err != nil {
		return webhttp.ResolvedConversationRequest{}, err
	}
	plan, err := r.buildConversationPlan(context.Background(), convID, body.Prompt, strings.TrimSpace(body.IdempotencyKey), resolvedProfile)
	if err != nil {
		return webhttp.ResolvedConversationRequest{}, err
	}
	return toResolvedConversationRequest(plan), nil
}

func (r *StrictRequestResolver) resolveProfileSelection(req *http.Request, bodyProfileRaw string) (gepprofiles.EngineProfileSlug, error) {
	if r == nil || r.profileRegistry == nil {
		return "", &webhttp.RequestResolutionError{
			Status:    http.StatusInternalServerError,
			ClientMsg: "profile resolver is not configured",
		}
	}

	slugRaw := strings.TrimSpace(bodyProfileRaw)
	if slugRaw == "" && req != nil {
		slugRaw = strings.TrimSpace(req.URL.Query().Get("profile"))
	}
	if slugRaw == "" {
		return r.defaultProfileSlug, nil
	}

	slug, err := gepprofiles.ParseEngineProfileSlug(slugRaw)
	if err != nil {
		return "", &webhttp.RequestResolutionError{
			Status:    http.StatusBadRequest,
			ClientMsg: "invalid profile: " + slugRaw,
			Err:       err,
		}
	}
	return slug, nil
}

func (r *StrictRequestResolver) resolveRegistrySelection(req *http.Request, bodyRegistryRaw string) (gepprofiles.RegistrySlug, error) {
	if r == nil || r.profileRegistry == nil {
		return "", &webhttp.RequestResolutionError{
			Status:    http.StatusInternalServerError,
			ClientMsg: "profile resolver is not configured",
		}
	}

	registryRaw := strings.TrimSpace(bodyRegistryRaw)
	if registryRaw == "" && req != nil {
		registryRaw = strings.TrimSpace(req.URL.Query().Get("registry"))
	}
	if registryRaw == "" {
		return r.defaultRegistrySlug, nil
	}

	registrySlug, err := gepprofiles.ParseRegistrySlug(registryRaw)
	if err != nil {
		return "", &webhttp.RequestResolutionError{
			Status:    http.StatusBadRequest,
			ClientMsg: "invalid registry: " + registryRaw,
			Err:       err,
		}
	}
	return registrySlug, nil
}

func (r *StrictRequestResolver) resolveEffectiveProfile(
	ctx context.Context,
	registrySlug gepprofiles.RegistrySlug,
	profileSlug gepprofiles.EngineProfileSlug,
) (*gepprofiles.ResolvedEngineProfile, error) {
	in := gepprofiles.ResolveInput{
		RegistrySlug:      registrySlug,
		EngineProfileSlug: profileSlug,
	}
	resolved, err := r.profileRegistry.ResolveEngineProfile(ctx, in)
	if err != nil {
		return nil, r.toRequestResolutionError(err, profileSlug.String())
	}
	return resolved, nil
}

func cloneResolvedInferenceSettings(in *aisettings.InferenceSettings) *aisettings.InferenceSettings {
	if in == nil {
		return nil
	}
	return in.Clone()
}

func (r *StrictRequestResolver) buildConversationPlan(
	ctx context.Context,
	convID string,
	prompt string,
	idempotencyKey string,
	resolvedProfile *gepprofiles.ResolvedEngineProfile,
) (*resolvedConversationPlan, error) {
	resolvedPlan, err := r.resolveRuntimePlan(ctx, resolvedProfile)
	if err != nil {
		return nil, err
	}
	runtimeKey := r.runtimeKey
	if resolvedProfile != nil && strings.TrimSpace(resolvedProfile.EngineProfileSlug.String()) != "" {
		runtimeKey = strings.TrimSpace(resolvedProfile.EngineProfileSlug.String())
	}
	if runtimeKey == "" {
		runtimeKey = "default"
	}

	runtime := &resolvedConversationRuntime{
		RuntimeKey:        runtimeKey,
		ProfileVersion:    0,
		InferenceSettings: nil,
		ProfileMetadata:   nil,
	}
	if resolvedPlan != nil {
		runtime.ProfileVersion = resolvedPlan.ProfileVersion
		runtime.InferenceSettings = cloneResolvedInferenceSettings(resolvedPlan.InferenceSettings)
		runtime.ProfileMetadata = copyMetadataMap(resolvedPlan.ProfileMetadata)
		if resolvedPlan.Runtime != nil {
			runtime.SystemPrompt = strings.TrimSpace(resolvedPlan.Runtime.SystemPrompt)
			runtime.Middlewares = append([]infruntime.MiddlewareUse(nil), resolvedPlan.Runtime.Middlewares...)
			runtime.ToolNames = append([]string(nil), resolvedPlan.Runtime.Tools...)
		}
	}
	runtime.RuntimeFingerprint = infruntime.BuildRuntimeFingerprintFromSettings(runtime.RuntimeKey, runtime.ProfileVersion, &infruntime.ProfileRuntime{
		SystemPrompt: runtime.SystemPrompt,
		Middlewares:  append([]infruntime.MiddlewareUse(nil), runtime.Middlewares...),
		Tools:        append([]string(nil), runtime.ToolNames...),
	}, runtime.InferenceSettings)

	return &resolvedConversationPlan{
		ConvID:         convID,
		Prompt:         prompt,
		IdempotencyKey: idempotencyKey,
		Runtime:        runtime,
	}, nil
}

func (r *StrictRequestResolver) resolveRuntimePlan(
	ctx context.Context,
	resolved *gepprofiles.ResolvedEngineProfile,
) (*infruntime.ResolvedRuntimePlan, error) {
	plan, err := infruntime.ResolveRuntimePlan(ctx, r.profileRegistry, resolved, infruntime.ResolveRuntimePlanOptions{
		BaseInferenceSettings: r.baseInferenceSetting,
	})
	if err == nil {
		return plan, nil
	}
	if errors.Is(err, gepprofiles.ErrProfileNotFound) {
		slug := ""
		if resolved != nil {
			slug = resolved.EngineProfileSlug.String()
		}
		return nil, r.toRequestResolutionError(err, slug)
	}
	if resolved != nil && r.profileRegistry != nil {
		return nil, &webhttp.RequestResolutionError{
			Status:    http.StatusBadRequest,
			ClientMsg: "invalid pinocchio runtime extension",
			Err:       err,
		}
	}
	return nil, err
}

func (r *StrictRequestResolver) resolveProfileRuntime(ctx context.Context, resolved *gepprofiles.ResolvedEngineProfile) (*infruntime.ProfileRuntime, error) {
	plan, err := r.resolveRuntimePlan(ctx, resolved)
	if err != nil {
		return nil, err
	}
	if plan == nil {
		return nil, nil
	}
	return plan.Runtime, nil
}

func toResolvedConversationRequest(plan *resolvedConversationPlan) webhttp.ResolvedConversationRequest {
	if plan == nil || plan.Runtime == nil {
		return webhttp.ResolvedConversationRequest{}
	}
	return webhttp.ResolvedConversationRequest{
		ConvID:                    plan.ConvID,
		RuntimeKey:                plan.Runtime.RuntimeKey,
		RuntimeFingerprint:        plan.Runtime.RuntimeFingerprint,
		ProfileVersion:            plan.Runtime.ProfileVersion,
		ResolvedInferenceSettings: cloneResolvedInferenceSettings(plan.Runtime.InferenceSettings),
		ResolvedRuntime: &infruntime.ProfileRuntime{
			SystemPrompt: strings.TrimSpace(plan.Runtime.SystemPrompt),
			Middlewares:  append([]infruntime.MiddlewareUse(nil), plan.Runtime.Middlewares...),
			Tools:        append([]string(nil), plan.Runtime.ToolNames...),
		},
		ProfileMetadata: copyMetadataMap(plan.Runtime.ProfileMetadata),
		Prompt:          plan.Prompt,
		IdempotencyKey:  plan.IdempotencyKey,
	}
}

func rejectLegacySelectionFields(req *http.Request, legacyRuntimeKey string, legacyRegistry string) error {
	if req != nil {
		query := req.URL.Query()
		if _, ok := query["runtime_key"]; ok {
			return &webhttp.RequestResolutionError{
				Status:    http.StatusBadRequest,
				ClientMsg: "unsupported legacy selector: runtime_key",
			}
		}
		if _, ok := query["registry_slug"]; ok {
			return &webhttp.RequestResolutionError{
				Status:    http.StatusBadRequest,
				ClientMsg: "unsupported legacy selector: registry_slug",
			}
		}
	}
	if strings.TrimSpace(legacyRuntimeKey) != "" {
		return &webhttp.RequestResolutionError{
			Status:    http.StatusBadRequest,
			ClientMsg: "unsupported legacy selector: runtime_key",
		}
	}
	if strings.TrimSpace(legacyRegistry) != "" {
		return &webhttp.RequestResolutionError{
			Status:    http.StatusBadRequest,
			ClientMsg: "unsupported legacy selector: registry_slug",
		}
	}
	return nil
}

func copyMetadataMap(in map[string]any) map[string]any {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func (r *StrictRequestResolver) toRequestResolutionError(err error, slug string) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, gepprofiles.ErrProfileNotFound) {
		msg := "profile not found"
		if strings.TrimSpace(slug) != "" {
			msg += ": " + slug
		}
		return &webhttp.RequestResolutionError{Status: http.StatusNotFound, ClientMsg: msg}
	}
	if errors.Is(err, gepprofiles.ErrRegistryNotFound) {
		return &webhttp.RequestResolutionError{
			Status:    http.StatusNotFound,
			ClientMsg: "registry not found",
			Err:       err,
		}
	}
	var validationErr *gepprofiles.ValidationError
	if errors.As(err, &validationErr) {
		return &webhttp.RequestResolutionError{
			Status:    http.StatusBadRequest,
			ClientMsg: validationErr.Error(),
			Err:       err,
		}
	}
	return &webhttp.RequestResolutionError{
		Status:    http.StatusInternalServerError,
		ClientMsg: "profile resolution failed",
		Err:       err,
	}
}
