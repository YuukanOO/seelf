package domain

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"

	"github.com/YuukanOO/seelf/pkg/monad"
	"github.com/YuukanOO/seelf/pkg/storage"
)

type (
	// Custom types to hold Service array which implements the Scanner and Valuer
	// interface to store it as a json string in the database (no need to create another table for it).
	Services []Service

	// Hold data related to services deployed and runned upon a deployment
	// success.
	Service struct {
		name          string
		qualifiedName string
		image         string
		url           monad.Maybe[Url]
	}
)

func newService(conf Config, name, image string, url monad.Maybe[Url]) (s Service) {
	s.name = name

	// Empty image name, let's build it based on the given configuration
	if image == "" {
		s.image = fmt.Sprintf("%s/%s:%s", conf.AppName(), name, conf.Environment())
	} else {
		s.image = image
	}

	s.url = url
	s.qualifiedName = fmt.Sprintf("%s-%s", conf.ProjectName(), name)
	return s
}

func (s Service) Name() string          { return s.name }
func (s Service) Image() string         { return s.image }
func (s Service) Url() monad.Maybe[Url] { return s.url }
func (s Service) IsExposed() bool       { return s.url.HasValue() }
func (s Service) QualifiedName() string { return s.qualifiedName } // Returns the service qualified name which identifies it uniquely

// Append a new service (not exposed to the outside world) to the current services array.
func (s Services) Internal(conf Config, name, image string) (Services, Service) {
	service := newService(conf, name, image, monad.None[Url]())
	return append(s, service), service
}

// Append a new exposed service to the current array.
// Given a base url and a deployment config, it will generate the correct URL for the provided
// service name.
func (s Services) Public(baseUrl Url, conf Config, name, image string) (Services, Service) {
	subdomain := conf.SubDomain()

	// If the default domain has already been taken by another app, build a
	// unique subdomain with the service name being exposed.
	if s.hasExposedServices() {
		subdomain = fmt.Sprintf("%s.%s", name, subdomain)
	}

	service := newService(conf, name, image, monad.Value(baseUrl.SubDomain(subdomain)))

	return append(s, service), service
}

func (s Services) hasExposedServices() bool {
	for _, service := range s {
		if service.IsExposed() {
			return true
		}
	}

	return false
}

func (s Services) Value() (driver.Value, error) { return storage.ValueJSON(s) }
func (s *Services) Scan(value any) error        { return storage.ScanJSON(value, s) }

// Type needed to marshal an unexposed Service data.
type marshalledService struct {
	Name          string           `json:"name"`
	QualifiedName string           `json:"qualified_name"`
	Image         string           `json:"image"`
	Url           monad.Maybe[Url] `json:"url"`
}

func (s Service) MarshalJSON() ([]byte, error) {
	serv := marshalledService{
		Name:          s.name,
		QualifiedName: s.qualifiedName,
		Image:         s.image,
		Url:           s.url,
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
	s.url = m.Url

	return nil
}
