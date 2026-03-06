package chatservice

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-go-golems/geppetto/pkg/inference/middlewarecfg"
	gepprofiles "github.com/go-go-golems/geppetto/pkg/profiles"
	webchat "github.com/go-go-golems/pinocchio/pkg/webchat"
	webhttp "github.com/go-go-golems/pinocchio/pkg/webchat/http"
	plzconfirmbackend "github.com/go-go-golems/plz-confirm/pkg/backend"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

const defaultConfirmMountPath = "/confirm"

type ProfileAPIOptions struct {
	Registry                        gepprofiles.Registry
	DefaultRegistrySlug             gepprofiles.RegistrySlug
	MiddlewareDefinitions           middlewarecfg.DefinitionRegistry
	WriteActor                      string
	WriteSource                     string
	EnableCurrentProfileCookieRoute bool
}

type Options struct {
	Server           *webchat.Server
	RequestResolver  webhttp.ConversationRequestResolver
	ProfileAPI       *ProfileAPIOptions
	ConfirmMountPath string
}

type Component struct {
	server           *webchat.Server
	requestResolver  webhttp.ConversationRequestResolver
	profileAPI       *ProfileAPIOptions
	confirmMountPath string
}

func New(opts Options) *Component {
	confirmMountPath := opts.ConfirmMountPath
	if confirmMountPath == "" {
		confirmMountPath = defaultConfirmMountPath
	}
	var profileAPI *ProfileAPIOptions
	if opts.ProfileAPI != nil {
		copyValue := *opts.ProfileAPI
		profileAPI = &copyValue
	}
	return &Component{
		server:           opts.Server,
		requestResolver:  opts.RequestResolver,
		profileAPI:       profileAPI,
		confirmMountPath: confirmMountPath,
	}
}

func (c *Component) MountRoutes(mux *http.ServeMux) error {
	if mux == nil {
		return fmt.Errorf("chat service mount mux is nil")
	}
	if c.server == nil {
		return fmt.Errorf("chat service server is nil")
	}
	if c.requestResolver == nil {
		return fmt.Errorf("chat service request resolver is nil")
	}

	chatHandler := webhttp.NewChatHandler(c.server.ChatService(), c.requestResolver)
	wsHandler := webhttp.NewWSHandler(
		c.server.StreamHub(),
		c.requestResolver,
		websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }},
	)
	timelineHandler := webhttp.NewTimelineHandler(
		c.server.TimelineService(),
		log.With().Str("component", "chat-service").Logger(),
	)

	mux.HandleFunc("/chat", chatHandler)
	mux.HandleFunc("/chat/", chatHandler)
	mux.HandleFunc("/ws", wsHandler)
	mux.HandleFunc("/api/timeline", timelineHandler)
	mux.HandleFunc("/api/timeline/", timelineHandler)
	mux.Handle("/api/", c.server.APIHandler())

	if c.profileAPI != nil {
		if c.profileAPI.Registry == nil {
			return fmt.Errorf("chat service profile registry is nil")
		}
		webhttp.RegisterProfileAPIHandlers(mux, c.profileAPI.Registry, webhttp.ProfileAPIHandlerOptions{
			DefaultRegistrySlug:             c.profileAPI.DefaultRegistrySlug,
			EnableCurrentProfileCookieRoute: c.profileAPI.EnableCurrentProfileCookieRoute,
			WriteActor:                      c.profileAPI.WriteActor,
			WriteSource:                     c.profileAPI.WriteSource,
			MiddlewareDefinitions:           c.profileAPI.MiddlewareDefinitions,
		})
	}

	if c.confirmMountPath != "" {
		plzconfirmbackend.NewServer().Mount(mux, c.confirmMountPath)
	}

	return nil
}

func (c *Component) Init(context.Context) error {
	if c == nil || c.server == nil {
		return fmt.Errorf("chat service is not initialized")
	}
	return nil
}

func (c *Component) Start(context.Context) error {
	if c == nil || c.server == nil {
		return fmt.Errorf("chat service is not initialized")
	}
	return nil
}

func (c *Component) Stop(context.Context) error {
	return nil
}

func (c *Component) Health(context.Context) error {
	if c == nil || c.server == nil {
		return fmt.Errorf("chat service server is not initialized")
	}
	return nil
}
