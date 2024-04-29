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

	HttpEntrypointOptions struct {
		// True if this entrypoint should take the default subdomain for an application.
		UseDefaultSubdomain bool
		// True if this entrypoint is natively managed by the target and does not require specific port exposure.
		Managed bool
	}

	// Custom types to hold Service array which implements the Scanner and Valuer
	// interface to store it as a json string in the database (no need to create another table for it).
	Services []Service

	// Hold data related to services deployed upon a deployment success.
	Service struct {
		name          string
		qualifiedName string
		image         string
		entrypoints   []Entrypoint
	}
)

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

func newEntrypointName(suffix string, router Router, port Port) EntrypointName {
	return EntrypointName(suffix + "-" + port.String() + "-" + string(router))
}

// Creates a new service. If the image is empty, a unique image name will be
// generated.
func (c DeploymentConfig) NewService(name, image string) (s Service) {
	s.name = name
	s.qualifiedName = c.QualifiedName(name)

	if image == "" {
		s.image = c.ImageName(name)
	} else {
		s.image = image
	}

	return s
}

// Adds an HTTP entrypoint to the service.
// HTTP entrypoints can be marked as automatically managed meaning they do not need a
// specific configuration and are natively handled by the target.
func (s *Service) AddHttpEntrypoint(conf DeploymentConfig, port Port, options HttpEntrypointOptions) Entrypoint {
	for _, entry := range s.entrypoints {
		// Already have an HTTP endpoint on this service, copy the subdomain and add it as a custom one.
		if entry.router == RouterHttp {
			return s.addEntrypoint(RouterHttp, !options.Managed, port, entry.subdomain.Get(""))
		}
	}

	return s.addEntrypoint(RouterHttp, !options.Managed, port, conf.SubDomain(s.name, options.UseDefaultSubdomain))
}

// Adds a custom TCP entrypoint.
func (s *Service) AddTCPEntrypoint(port Port) Entrypoint {
	return s.addEntrypoint(RouterTcp, true, port)
}

// Adds a custom UDP entrypoint.
func (s *Service) AddUDPEntrypoint(port Port) Entrypoint {
	return s.addEntrypoint(RouterUdp, true, port)
}

func (s *Service) addEntrypoint(router Router, isCustom bool, port Port, subdomain ...string) (e Entrypoint) {
	// Check if the entrypoint already exists
	for _, entry := range s.entrypoints {
		if entry.port == port && entry.router == router {
			return entry
		}
	}

	e.name = newEntrypointName(s.qualifiedName, router, port)
	e.isCustom = isCustom
	e.router = router
	e.port = port

	if len(subdomain) > 0 {
		e.subdomain.Set(subdomain[0])
	}

	s.entrypoints = append(s.entrypoints, e)

	return e
}

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

// Retrieve entrypoints for this service.
func (s Services) Entrypoints() []Entrypoint {
	var result []Entrypoint

	for _, service := range s {
		result = append(result, service.entrypoints...)
	}

	return result
}

// Retrieve custom entrypoints for this service. Ones that are not natively
// managed by the target and requires a manual configuration.
func (s Services) CustomEntrypoints() []Entrypoint {
	return slices.DeleteFunc(s.Entrypoints(), isNotCustom)
}

func (s Services) Value() (driver.Value, error) { return storage.ValueJSON(s) }
func (s *Services) Scan(value any) error        { return storage.ScanJSON(value, s) }

func isNotCustom(entrypoint Entrypoint) bool {
	return !entrypoint.isCustom
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
		Name          string                 `json:"name"`
		QualifiedName string                 `json:"qualified_name"`
		Image         string                 `json:"image"`
		Entrypoints   []marshalledEntrypoint `json:"entrypoints"`
	}
)

func (s Service) MarshalJSON() ([]byte, error) {
	serv := marshalledService{
		Name:          s.name,
		QualifiedName: s.qualifiedName,
		Image:         s.image,
		Entrypoints:   make([]marshalledEntrypoint, len(s.entrypoints)),
	}

	for i, entry := range s.entrypoints {
		serv.Entrypoints[i] = marshalledEntrypoint{
			Name:      string(entry.name),
			IsCustom:  entry.isCustom,
			Router:    entry.router,
			Subdomain: entry.subdomain,
			Port:      entry.port,
		}
	}

	return json.Marshal(serv)
}

func (s *Service) UnmarshalJSON(b []byte) error {
	var m marshalledService

	if err := json.Unmarshal(b, &m); err != nil {
		return err
	}

	s.image = m.Image
	s.name = m.Name
	s.qualifiedName = m.QualifiedName
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
