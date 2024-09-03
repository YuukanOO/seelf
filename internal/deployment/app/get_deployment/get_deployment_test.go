package get_deployment_test

import (
	"testing"

	"github.com/YuukanOO/seelf/internal/deployment/app/get_deployment"
	"github.com/YuukanOO/seelf/pkg/assert"
	"github.com/YuukanOO/seelf/pkg/monad"
)

func Test_Deployment(t *testing.T) {
	t.Run("should skip the url resolve if the target has been deleted", func(t *testing.T) {
		d := get_deployment.Deployment{
			Target: get_deployment.TargetSummary{
				ID: "target-id",
			},
			State: get_deployment.State{
				Services: monad.Value(get_deployment.Services{
					{
						Name:  "app",
						Image: "app-image",
						Entrypoints: []get_deployment.Entrypoint{
							{
								Name:      "http1",
								Router:    "http",
								IsCustom:  false,
								Subdomain: monad.Value("app"),
								Port:      8081,
							},
							{
								Name:      "http2",
								Router:    "http",
								IsCustom:  true,
								Subdomain: monad.Value("app"),
								Port:      8081,
							},
						},
					},
					{
						Name:  "db",
						Image: "db-image",
						Entrypoints: []get_deployment.Entrypoint{
							{
								Name:     "tcp1",
								Router:   "tcp",
								IsCustom: true,
								Port:     5432,
							},
						},
					},
				}),
			},
		}

		d.ResolveServicesUrls()

		assert.DeepEqual(t, get_deployment.Services{
			{
				Name:  "app",
				Image: "app-image",
				Entrypoints: []get_deployment.Entrypoint{
					{
						Name:      "http1",
						Router:    "http",
						IsCustom:  false,
						Subdomain: monad.Value("app"),
						Port:      8081,
					},
					{
						Name:      "http2",
						Router:    "http",
						IsCustom:  true,
						Subdomain: monad.Value("app"),
						Port:      8081,
					},
				},
			},
			{
				Name:  "db",
				Image: "db-image",
				Entrypoints: []get_deployment.Entrypoint{
					{
						Name:     "tcp1",
						Router:   "tcp",
						IsCustom: true,
						Port:     5432,
					},
				},
			},
		}, d.State.Services.Get(get_deployment.Services{}))
	})

	t.Run("should handle deployment data before the v2.0.0 release", func(t *testing.T) {
		d := get_deployment.Deployment{
			AppID:       "app-id",
			Environment: "production",
			Target: get_deployment.TargetSummary{
				ID:          "target-id",
				Url:         monad.Value("https://docker.localhost"),
				Entrypoints: monad.Value(get_deployment.Entrypoints{}),
			},
			State: get_deployment.State{
				Services: monad.Value(get_deployment.Services{
					{
						Name:  "app",
						Image: "app-image",
						Url:   monad.Value("https://app.docker.localhost"),
					},
				}),
			},
		}

		d.ResolveServicesUrls()

		assert.DeepEqual(t, get_deployment.Services{
			{
				Name:  "app",
				Image: "app-image",
				Url:   monad.Value("https://app.docker.localhost"),
				Entrypoints: []get_deployment.Entrypoint{
					{
						Name:     "default",
						Router:   "http",
						IsCustom: false,
						Port:     80,
						Url:      monad.Value("https://app.docker.localhost"),
					},
				},
			},
		}, d.State.Services.Get(get_deployment.Services{}))
	})

	t.Run("should handle deployment data after the v2.0.0 and before the v2.2.0 release", func(t *testing.T) {
		d := get_deployment.Deployment{
			AppID:       "app-id",
			Environment: "production",
			Target: get_deployment.TargetSummary{
				ID:          "target-id",
				Url:         monad.Value("https://docker.localhost"),
				Entrypoints: monad.Value(get_deployment.Entrypoints{}),
			},
			State: get_deployment.State{
				Services: monad.Value(get_deployment.Services{
					{
						Name:      "app",
						Image:     "app-image",
						Subdomain: monad.Value("app"),
					},
				}),
			},
		}

		d.ResolveServicesUrls()

		assert.DeepEqual(t, get_deployment.Services{
			{
				Name:      "app",
				Image:     "app-image",
				Subdomain: monad.Value("app"),
				Entrypoints: []get_deployment.Entrypoint{
					{
						Name:      "default",
						Router:    "http",
						IsCustom:  false,
						Port:      80,
						Subdomain: monad.Value("app"),
						Url:       monad.Value("https://app.docker.localhost"),
					},
				},
			},
		}, d.State.Services.Get(get_deployment.Services{}))
	})

	t.Run("should correctly handle new deployments", func(t *testing.T) {
		d := get_deployment.Deployment{
			AppID:       "app-id",
			Environment: "production",
			Target: get_deployment.TargetSummary{
				ID:  "target-id",
				Url: monad.Value("https://docker.localhost"),
				Entrypoints: monad.Value(get_deployment.Entrypoints{
					"app-id": {
						"production": {
							"http2": monad.Value[uint](80810),
							"tcp1":  monad.Value[uint](54320),
						},
					},
				}),
			},
			State: get_deployment.State{
				Services: monad.Value(get_deployment.Services{
					{
						Name:  "app",
						Image: "app-image",
						Entrypoints: []get_deployment.Entrypoint{
							{
								Name:      "http1",
								Router:    "http",
								IsCustom:  false,
								Subdomain: monad.Value("app"),
								Port:      8081,
							},
							{
								Name:      "http2",
								Router:    "http",
								IsCustom:  true,
								Subdomain: monad.Value("app"),
								Port:      8081,
							},
						},
					},
					{
						Name:  "db",
						Image: "db-image",
						Entrypoints: []get_deployment.Entrypoint{
							{
								Name:     "tcp1",
								Router:   "tcp",
								IsCustom: true,
								Port:     5432,
							},
							{
								Name:     "tcp2",
								Router:   "tcp",
								IsCustom: true,
								Port:     5433,
							},
						},
					},
				}),
			},
		}

		d.ResolveServicesUrls()

		assert.DeepEqual(t, get_deployment.Services{
			{
				Name:  "app",
				Image: "app-image",
				Entrypoints: []get_deployment.Entrypoint{
					{
						Name:      "http1",
						Router:    "http",
						IsCustom:  false,
						Subdomain: monad.Value("app"),
						Port:      8081,
						Url:       monad.Value("https://app.docker.localhost"),
					},
					{
						Name:          "http2",
						Router:        "http",
						IsCustom:      true,
						Subdomain:     monad.Value("app"),
						Port:          8081,
						Url:           monad.Value("http://app.docker.localhost:80810"),
						PublishedPort: monad.Value[uint](80810),
					},
				},
			},
			{
				Name:  "db",
				Image: "db-image",
				Entrypoints: []get_deployment.Entrypoint{
					{
						Name:          "tcp1",
						Router:        "tcp",
						IsCustom:      true,
						Port:          5432,
						Url:           monad.Value("tcp://docker.localhost:54320"),
						PublishedPort: monad.Value[uint](54320),
					},
					{
						Name:     "tcp2",
						Router:   "tcp",
						IsCustom: true,
						Port:     5433,
					},
				},
			},
		}, d.State.Services.Get(get_deployment.Services{}))
	})
}
