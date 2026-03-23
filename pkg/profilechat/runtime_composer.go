package profilechat

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	gepmw "github.com/go-go-golems/geppetto/pkg/inference/middleware"
	"github.com/go-go-golems/geppetto/pkg/inference/middlewarecfg"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	infruntime "github.com/go-go-golems/pinocchio/pkg/inference/runtime"
	"github.com/pkg/errors"
)

type RuntimeComposerOptions struct {
	RuntimeKey      string
	SystemPrompt    string
	AllowedTools    []string
	ContextProvider ConversationContextProvider
}

type ConversationContext struct {
	SystemPromptAddendum string
	Metadata             map[string]any
}

type ConversationContextProvider interface {
	Lookup(ctx context.Context, convID string) (*ConversationContext, error)
}

type RuntimeComposer struct {
	parsed             *values.Values
	options            RuntimeComposerOptions
	definitions        middlewarecfg.DefinitionRegistry
	buildDeps          middlewarecfg.BuildDeps
	defaultMiddlewares []infruntime.MiddlewareUse
}

func NewRuntimeComposer(
	parsed *values.Values,
	options RuntimeComposerOptions,
	definitions middlewarecfg.DefinitionRegistry,
	buildDeps middlewarecfg.BuildDeps,
	defaultMiddlewares []infruntime.MiddlewareUse,
) *RuntimeComposer {
	return &RuntimeComposer{
		parsed:             parsed,
		options:            options,
		definitions:        definitions,
		buildDeps:          buildDeps,
		defaultMiddlewares: cloneMiddlewareUses(defaultMiddlewares),
	}
}

func NewRuntimeComposerWithDefinitions(
	parsed *values.Values,
	options RuntimeComposerOptions,
	definitions middlewarecfg.DefinitionRegistry,
	buildDeps middlewarecfg.BuildDeps,
	defaultMiddlewares []infruntime.MiddlewareUse,
) *RuntimeComposer {
	return NewRuntimeComposer(parsed, options, definitions, buildDeps, defaultMiddlewares)
}

func (c *RuntimeComposer) MiddlewareDefinitions() middlewarecfg.DefinitionRegistry {
	if c == nil {
		return nil
	}
	return c.definitions
}

func (c *RuntimeComposer) Compose(ctx context.Context, req infruntime.ConversationRuntimeRequest) (infruntime.ComposedRuntime, error) {
	if c == nil || c.parsed == nil {
		return infruntime.ComposedRuntime{}, errors.New("runtime composer is not configured")
	}
	if ctx == nil {
		return infruntime.ComposedRuntime{}, errors.New("compose context is nil")
	}

	effectiveInferenceSettings, err := settings.NewInferenceSettingsFromParsedValues(c.parsed)
	if err != nil {
		return infruntime.ComposedRuntime{}, errors.Wrap(err, "parse inference settings")
	}
	if req.ResolvedInferenceSettings != nil {
		effectiveInferenceSettings = req.ResolvedInferenceSettings.Clone()
	}

	runtimeKey := strings.TrimSpace(req.ProfileKey)
	if runtimeKey == "" {
		runtimeKey = strings.TrimSpace(c.options.RuntimeKey)
	}
	if runtimeKey == "" {
		runtimeKey = "default"
	}

	profileRuntime := req.ResolvedProfileRuntime

	systemPrompt := strings.TrimSpace(c.options.SystemPrompt)
	if profileRuntime != nil && strings.TrimSpace(profileRuntime.SystemPrompt) != "" {
		systemPrompt = strings.TrimSpace(profileRuntime.SystemPrompt)
	}
	if systemPrompt == "" {
		systemPrompt = "You are a helpful assistant."
	}

	allowedTools := append([]string(nil), c.options.AllowedTools...)
	if profileRuntime != nil && len(profileRuntime.Tools) > 0 {
		allowedTools = runtimeToolsFromProfile(profileRuntime)
	}

	contextAddendum := ""
	if c.options.ContextProvider != nil && strings.TrimSpace(req.ConvID) != "" {
		convContext, err := c.options.ContextProvider.Lookup(ctx, strings.TrimSpace(req.ConvID))
		if err != nil {
			return infruntime.ComposedRuntime{}, errors.Wrap(err, "lookup conversation context")
		}
		if convContext != nil {
			contextAddendum = strings.TrimSpace(convContext.SystemPromptAddendum)
		}
	}
	if contextAddendum != "" {
		systemPrompt = joinPromptSections(systemPrompt, contextAddendum)
	}

	profileMiddlewares, err := runtimeMiddlewaresFromProfile(profileRuntime)
	if err != nil {
		return infruntime.ComposedRuntime{}, errors.Wrap(err, "normalize profile middlewares")
	}
	if profileRuntime == nil {
		profileMiddlewares = cloneMiddlewareUses(c.defaultMiddlewares)
	}

	middlewareInputs, err := runtimeMiddlewareInputsFromProfile(profileMiddlewares)
	if err != nil {
		return infruntime.ComposedRuntime{}, errors.Wrap(err, "normalize middleware inputs")
	}
	resolvedMiddlewares, resolvedUses, err := c.resolveMiddlewares(ctx, middlewareInputs)
	if err != nil {
		return infruntime.ComposedRuntime{}, errors.Wrap(err, "resolve middlewares")
	}

	engine_, err := infruntime.BuildEngineFromSettingsWithMiddlewares(
		ctx,
		effectiveInferenceSettings,
		systemPrompt,
		resolvedMiddlewares,
	)
	if err != nil {
		return infruntime.ComposedRuntime{}, errors.Wrap(err, "compose engine")
	}

	runtimeFingerprint := strings.TrimSpace(req.ResolvedProfileFingerprint)
	if runtimeFingerprint == "" {
		runtimeFingerprint = buildRuntimeFingerprint(runtimeKey, req.ProfileVersion, systemPrompt, resolvedUses, allowedTools, effectiveInferenceSettings)
	}

	return infruntime.ComposedRuntime{
		Engine:             engine_,
		RuntimeKey:         runtimeKey,
		RuntimeFingerprint: runtimeFingerprint,
		SeedSystemPrompt:   systemPrompt,
	}, nil
}

type middlewareResolveInput struct {
	Use           infruntime.MiddlewareUse
	ProfileConfig map[string]any
}

func (c *RuntimeComposer) resolveMiddlewares(
	ctx context.Context,
	inputs []middlewareResolveInput,
) ([]gepmw.Middleware, []infruntime.MiddlewareUse, error) {
	if len(inputs) == 0 {
		return nil, nil, nil
	}
	if c == nil || c.definitions == nil {
		return nil, nil, errors.New("middleware definitions are not configured")
	}

	resolvedInstances := make([]middlewarecfg.ResolvedInstance, 0, len(inputs))
	resolvedUses := make([]infruntime.MiddlewareUse, 0, len(inputs))
	for i, input := range inputs {
		instanceKey := middlewarecfg.MiddlewareInstanceKey(middlewarecfg.Use{
			Name:    input.Use.Name,
			ID:      input.Use.ID,
			Enabled: cloneBoolPtr(input.Use.Enabled),
		}, i)
		def, ok := c.definitions.GetDefinition(input.Use.Name)
		if !ok {
			return nil, nil, fmt.Errorf("unknown middleware %s", instanceKey)
		}

		sources := make([]middlewarecfg.Source, 0, 1)
		if len(input.ProfileConfig) > 0 {
			sources = append(sources, fixedPayloadSource{
				name:    "profile",
				layer:   middlewarecfg.SourceLayerProfile,
				payload: input.ProfileConfig,
			})
		}

		resolver := middlewarecfg.NewResolver(sources...)
		resolvedCfg, err := resolver.Resolve(def, middlewarecfg.Use{
			Name:    input.Use.Name,
			ID:      input.Use.ID,
			Enabled: cloneBoolPtr(input.Use.Enabled),
		})
		if err != nil {
			return nil, nil, fmt.Errorf("resolve middleware %s: %w", instanceKey, err)
		}

		resolvedInstances = append(resolvedInstances, middlewarecfg.ResolvedInstance{
			Key: instanceKey,
			Use: middlewarecfg.Use{
				Name:    input.Use.Name,
				ID:      input.Use.ID,
				Enabled: cloneBoolPtr(input.Use.Enabled),
			},
			Resolved: resolvedCfg,
			Def:      def,
		})
		resolvedUses = append(resolvedUses, infruntime.MiddlewareUse{
			Name:    input.Use.Name,
			ID:      input.Use.ID,
			Enabled: cloneBoolPtr(input.Use.Enabled),
			Config:  cloneStringAnyMap(resolvedCfg.Config),
		})
	}

	chain, err := middlewarecfg.BuildChain(ctx, c.buildDeps, resolvedInstances)
	if err != nil {
		return nil, nil, err
	}
	return chain, resolvedUses, nil
}

type fixedPayloadSource struct {
	name    string
	layer   middlewarecfg.SourceLayer
	payload map[string]any
}

func (s fixedPayloadSource) Name() string {
	return s.name
}

func (s fixedPayloadSource) Layer() middlewarecfg.SourceLayer {
	return s.layer
}

func (s fixedPayloadSource) Payload(middlewarecfg.Definition, middlewarecfg.Use) (map[string]any, bool, error) {
	if len(s.payload) == 0 {
		return nil, false, nil
	}
	return cloneStringAnyMap(s.payload), true, nil
}

func runtimeMiddlewaresFromProfile(spec *infruntime.ProfileRuntime) ([]infruntime.MiddlewareUse, error) {
	if spec == nil || len(spec.Middlewares) == 0 {
		return nil, nil
	}

	middlewares := make([]infruntime.MiddlewareUse, 0, len(spec.Middlewares))
	for i, mw := range spec.Middlewares {
		name := strings.TrimSpace(mw.Name)
		if name == "" {
			continue
		}
		config, err := normalizeConfigObject(mw.Config, fmt.Sprintf("profile middleware %s config", middlewarecfg.MiddlewareInstanceKey(middlewarecfg.Use{Name: mw.Name, ID: mw.ID, Enabled: cloneBoolPtr(mw.Enabled)}, i)))
		if err != nil {
			return nil, err
		}
		middlewares = append(middlewares, infruntime.MiddlewareUse{
			Name:    name,
			ID:      strings.TrimSpace(mw.ID),
			Enabled: cloneBoolPtr(mw.Enabled),
			Config:  config,
		})
	}
	if len(middlewares) == 0 {
		return nil, nil
	}
	return middlewares, nil
}

func runtimeMiddlewareInputsFromProfile(profileMiddlewares []infruntime.MiddlewareUse) ([]middlewareResolveInput, error) {
	inputs := make([]middlewareResolveInput, 0, len(profileMiddlewares))

	for i, use := range profileMiddlewares {
		name := strings.TrimSpace(use.Name)
		if name == "" {
			continue
		}
		profileConfig, err := normalizeConfigObject(use.Config, fmt.Sprintf("profile middleware %s config", middlewarecfg.MiddlewareInstanceKey(middlewarecfg.Use{Name: use.Name, ID: use.ID, Enabled: cloneBoolPtr(use.Enabled)}, i)))
		if err != nil {
			return nil, err
		}
		inputs = append(inputs, middlewareResolveInput{
			Use: infruntime.MiddlewareUse{
				Name:    name,
				ID:      strings.TrimSpace(use.ID),
				Enabled: cloneBoolPtr(use.Enabled),
			},
			ProfileConfig: profileConfig,
		})
	}
	return inputs, nil
}

func runtimeToolsFromProfile(spec *infruntime.ProfileRuntime) []string {
	if spec == nil || len(spec.Tools) == 0 {
		return nil
	}
	tools := make([]string, 0, len(spec.Tools))
	for _, tool := range spec.Tools {
		name := strings.TrimSpace(tool)
		if name == "" {
			continue
		}
		tools = append(tools, name)
	}
	if len(tools) == 0 {
		return nil
	}
	return tools
}

func cloneMiddlewareUses(in []infruntime.MiddlewareUse) []infruntime.MiddlewareUse {
	if len(in) == 0 {
		return nil
	}
	out := make([]infruntime.MiddlewareUse, 0, len(in))
	for _, mw := range in {
		name := strings.TrimSpace(mw.Name)
		if name == "" {
			continue
		}
		config, _ := normalizeConfigObject(mw.Config, "")
		out = append(out, infruntime.MiddlewareUse{
			Name:    name,
			ID:      strings.TrimSpace(mw.ID),
			Enabled: cloneBoolPtr(mw.Enabled),
			Config:  config,
		})
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

type RuntimeFingerprintInput struct {
	ProfileVersion uint64                     `json:"profile_version,omitempty"`
	RuntimeKey     string                     `json:"runtime_key"`
	SystemPrompt   string                     `json:"system_prompt"`
	Middlewares    []infruntime.MiddlewareUse `json:"middlewares"`
	Tools          []string                   `json:"tools"`
	StepMetadata   map[string]any             `json:"step_metadata,omitempty"`
}

func buildRuntimeFingerprint(
	runtimeKey string,
	profileVersion uint64,
	systemPrompt string,
	middlewares []infruntime.MiddlewareUse,
	tools []string,
	inferenceSettings *settings.InferenceSettings,
) string {
	var metadata map[string]any
	if inferenceSettings != nil {
		metadata = inferenceSettings.GetMetadata()
	}
	payload := RuntimeFingerprintInput{
		ProfileVersion: profileVersion,
		RuntimeKey:     runtimeKey,
		SystemPrompt:   systemPrompt,
		Middlewares:    middlewares,
		Tools:          tools,
		StepMetadata:   metadata,
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return runtimeKey
	}
	return string(b)
}

func cloneBoolPtr(in *bool) *bool {
	if in == nil {
		return nil
	}
	v := *in
	return &v
}

func joinPromptSections(parts ...string) string {
	trimmed := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		trimmed = append(trimmed, part)
	}
	return strings.Join(trimmed, "\n\n")
}

func normalizeConfigObject(raw any, context string) (map[string]any, error) {
	if raw == nil {
		return nil, nil
	}
	if object, ok := raw.(map[string]any); ok {
		return cloneStringAnyMap(object), nil
	}

	b, err := json.Marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("%s must be JSON-serializable: %w", strings.TrimSpace(context), err)
	}
	var out map[string]any
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, fmt.Errorf("%s must be an object: %w", strings.TrimSpace(context), err)
	}
	if out == nil {
		return nil, nil
	}
	return out, nil
}

func cloneStringAnyMap(in map[string]any) map[string]any {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}
