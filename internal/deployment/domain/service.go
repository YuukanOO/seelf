package domain

import (
	"database/sql/driver"
	"encoding/json"
	"slices"
	"strconv"
	"strings"

	"github.com/YuukanOO/seelf/pkg/apperr"
	"github.com/YuukanOO/seelf/pkg/monad"
	"github.com/YuukanOO/seelf/pkg/storage"
)

const (
	RouterHttp Router = "http"
	RouterTcp  Router = "tcp"
	RouterUdp  Router = "udp"
)

var ErrInvalidPort = apperr.New("invalid_port")

type (
	Port           uint // Tiny type definition to provide helper methods when dealing with all those uints!
	Router         string
	EntrypointName string

	Entrypoint struct {
		name      EntrypointName // Unique name of the entrypoint
		isCustom  bool           // Wether or not this entrypoint should be manually configured by the target
		router    Router
		subdomain monad.Maybe[string]
		port      Port
	}

	// Custom types to hold Service array which implements the Scanner and Valuer
	// interface to store it as a json string in the database (no need to create another table for it).
	Services []Service

	// Hold data related to services deployed upon a deployment success.
	Service struct {
		name        string
		image       string
		entrypoints []Entrypoint
	}

	// Main structure used to build a services array. Manipulated by actual providers.
	ServicesBuilder struct {
		config                    ConfigSnapshot
		defaultSubdomainAvailable bool
		services                  []*ServiceBuilder
	}

	ServiceBuilder struct {
		parent        *ServicesBuilder
		qualifiedName string
		subdomain     monad.Maybe[string]
		Service
	}
)

// Returns a new builder used to ease the process of building the services array.
func (c ConfigSnapshot) ServicesBuilder() ServicesBuilder {
	return ServicesBuilder{
		config:                    c,
		defaultSubdomainAvailable: true,
	}
}

func (b *ServicesBuilder) AddService(name, image string) *ServiceBuilder {
	// Check if the service already exists
	for _, service := range b.services {
		if service.Service.name == name {
			return service
		}
	}

	builder := &ServiceBuilder{
		parent:        b,
		qualifiedName: b.config.qualifiedName(name),
		Service: Service{
			name:  name,
			image: image,
		},
	}

	if builder.Service.image == "" {
		builder.Service.image = b.config.imageName(name)
	}

	b.services = append(b.services, builder)

	return builder
}

func (b *ServiceBuilder) AddHttpEntrypoint(port Port, custom bool) Entrypoint {
	return b.addEntrypoint(RouterHttp, port, custom)
}

func (b *ServiceBuilder) AddTCPEntrypoint(port Port, custom bool) Entrypoint {
	return b.addEntrypoint(RouterTcp, port, custom)
}

func (b *ServiceBuilder) AddUDPEntrypoint(port Port, custom bool) Entrypoint {
	return b.addEntrypoint(RouterUdp, port, custom)
}

func (b *ServiceBuilder) addEntrypoint(router Router, port Port, custom bool) Entrypoint {
	// Check if the entrypoint already exists and returns early
	for _, entry := range b.Service.entrypoints {
		if entry.port == port && entry.router == router {
			return entry
		}
	}

	entrypoint := Entrypoint{
		name:     newEntrypointName(b.qualifiedName, router, port),
		isCustom: custom,
		router:   router,
		port:     port,
	}

	if router == RouterHttp {
		if !b.subdomain.HasValue() {
			b.subdomain.Set(b.parent.config.subDomain(b.Service.name, b.parent.defaultSubdomainAvailable))
			b.parent.defaultSubdomainAvailable = false
		}

		entrypoint.subdomain = b.subdomain
	}

	b.Service.entrypoints = append(b.Service.entrypoints, entrypoint)

	return entrypoint
}

func (b *ServicesBuilder) Services() Services {
	services := make(Services, len(b.services))

	for i, service := range b.services {
		services[i] = service.Service
	}

	return services
}

// Try to parse the given port from a raw string.
func ParsePort(raw string) (Port, error) {
	v, err := strconv.ParseUint(raw, 10, 0)

	if err != nil {
		return 0, ErrInvalidPort
	}

	return Port(v), nil
}

func (p Port) String() string { return strconv.FormatUint(uint64(p), 10) }
func (p Port) Uint32() uint32 { return uint32(p) }

func (s Service) Name() string  { return s.name }
func (s Service) Image() string { return s.image }

func (e Entrypoint) Name() EntrypointName           { return e.name }
func (e Entrypoint) Router() Router                 { return e.router }
func (e Entrypoint) Subdomain() monad.Maybe[string] { return e.subdomain }
func (e Entrypoint) Port() Port                     { return e.port }

// Check if this entrypoint should be manually configured by the target.
// This is needed because default HTTP entrypoints are mostly managed automatically by the proxy
// since ports are known (80 and 443).
func (e Entrypoint) IsCustom() bool { return e.isCustom }

func (e EntrypointName) Protocol() string {
	p := Router(e[strings.LastIndex(string(e), "-")+1:])

	if p == RouterHttp {
		return "tcp"
	}

	return string(p)
}

// Retrieve all entrypoints for every services.
func (s Services) Entrypoints() []Entrypoint {
	var result []Entrypoint

	for _, service := range s {
		result = append(result, service.entrypoints...)
	}

	return result
}

// Retrieve all custom entrypoints. Ones that are not natively
// managed by the target and requires a manual configuration.
func (s Services) CustomEntrypoints() []Entrypoint {
	return slices.DeleteFunc(s.Entrypoints(), isNotCustom)
}

func (s Services) Value() (driver.Value, error) { return storage.ValueJSON(s) }
func (s *Services) Scan(value any) error        { return storage.ScanJSON(value, s) }

func isNotCustom(entrypoint Entrypoint) bool {
	return !entrypoint.isCustom
}

func newEntrypointName(prefix string, router Router, port Port) EntrypointName {
	return EntrypointName(prefix + "-" + port.String() + "-" + string(router))
}

// Types needed to marshal an unexposed Service data.
type (
	marshalledEntrypoint struct {
		Name      string              `json:"name"`
		IsCustom  bool                `json:"is_custom"`
		Router    Router              `json:"router"`
		Subdomain monad.Maybe[string] `json:"subdomain"`
		Port      Port                `json:"port"`
	}

	marshalledService struct {
		Name        string                 `json:"name"`
		Image       string                 `json:"image"`
		Entrypoints []marshalledEntrypoint `json:"entrypoints"`
	}
)

func (s Service) MarshalJSON() ([]byte, error) {
	service := marshalledService{
		Name:        s.name,
		Image:       s.image,
		Entrypoints: make([]marshalledEntrypoint, len(s.entrypoints)),
	}

	for i, entry := range s.entrypoints {
		service.Entrypoints[i] = marshalledEntrypoint{
			Name:      string(entry.name),
			IsCustom:  entry.isCustom,
			Router:    entry.router,
			Subdomain: entry.subdomain,
			Port:      entry.port,
		}
	}

	return json.Marshal(service)
}

func (s *Service) UnmarshalJSON(b []byte) error {
	var m marshalledService

	if err := json.Unmarshal(b, &m); err != nil {
		return err
	}

	s.image = m.Image
	s.name = m.Name
	s.entrypoints = make([]Entrypoint, len(m.Entrypoints))

	for i, entry := range m.Entrypoints {
		s.entrypoints[i] = Entrypoint{
			name:      EntrypointName(entry.Name),
			isCustom:  entry.IsCustom,
			router:    entry.Router,
			subdomain: entry.Subdomain,
			port:      entry.Port,
		}
	}

	return nil
}
