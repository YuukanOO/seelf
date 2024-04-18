package domain

import (
	"database/sql/driver"
	"encoding/json"

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
		subdomain     monad.Maybe[string]
	}
)

func (s Service) Name() string                   { return s.name }
func (s Service) Image() string                  { return s.image }
func (s Service) Subdomain() monad.Maybe[string] { return s.subdomain }
func (s Service) QualifiedName() string          { return s.qualifiedName } // Returns the service qualified name which identifies it uniquely

// Append a new service to the current array using the provided configuration.
// If the service is exposed, a subdomain will be generated for it.
func (s Services) Append(conf DeploymentConfig, name, image string, exposed bool) (Services, Service) {
	var service Service

	service.name = name

	// Empty image name, let's build it based on the given configuration
	if image == "" {
		service.image = conf.ImageName(name)
	} else {
		service.image = image
	}

	if exposed {
		service.subdomain.Set(conf.SubDomain(name, s.hasExposedServices()))
	}

	service.qualifiedName = conf.QualifiedName(name)

	return append(s, service), service
}

func (s Services) hasExposedServices() bool {
	for _, service := range s {
		if service.subdomain.HasValue() {
			return true
		}
	}

	return false
}

func (s Services) Value() (driver.Value, error) { return storage.ValueJSON(s) }
func (s *Services) Scan(value any) error        { return storage.ScanJSON(value, s) }

// Type needed to marshal an unexposed Service data.
type marshalledService struct {
	Name          string              `json:"name"`
	QualifiedName string              `json:"qualified_name"`
	Image         string              `json:"image"`
	Subdomain     monad.Maybe[string] `json:"subdomain"`
}

func (s Service) MarshalJSON() ([]byte, error) {
	serv := marshalledService{
		Name:          s.name,
		QualifiedName: s.qualifiedName,
		Image:         s.image,
		Subdomain:     s.subdomain,
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
	s.subdomain = m.Subdomain

	return nil
}
