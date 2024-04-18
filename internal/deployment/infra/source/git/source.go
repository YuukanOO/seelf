package git

import (
	"context"
	"errors"

	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/go-git/go-git/v5/config"

	"github.com/YuukanOO/seelf/internal/deployment/infra/source"
	"github.com/YuukanOO/seelf/pkg/apperr"
	"github.com/YuukanOO/seelf/pkg/monad"
	"github.com/YuukanOO/seelf/pkg/types"
	"github.com/YuukanOO/seelf/pkg/validate"
	"github.com/YuukanOO/seelf/pkg/validate/strings"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
)

var (
	ErrGitRemoteNotReachable = apperr.New("git_remote_not_reachable")
	ErrGitBranchNotFound     = apperr.New("git_branch_not_found")
	ErrAppRetrievedFailed    = errors.New("app_retrieved_failed")
	ErrGitCloneFailed        = errors.New("git_clone_failed")
	ErrGitResolveFailed      = errors.New("git_resolve_failed")
	ErrGitCheckoutFailed     = errors.New("git_checkout_failed")
)

const basicAuthUser = "seelf"

type (
	// Public request to trigger a git deployment
	Body struct {
		Branch string              `json:"branch"`
		Hash   monad.Maybe[string] `json:"hash"`
	}

	service struct {
		reader domain.AppsReader
	}
)

// Builds a new trigger to process git deployments
func New(reader domain.AppsReader) source.Source {
	return &service{reader}
}

func (*service) CanPrepare(payload any) bool          { return types.Is[Body](payload) }
func (*service) CanFetch(meta domain.SourceData) bool { return types.Is[Data](meta) }

func (s *service) Prepare(ctx context.Context, app domain.App, payload any) (domain.SourceData, error) {
	req, ok := payload.(Body)

	if !ok {
		return nil, domain.ErrInvalidSourcePayload
	}

	if err := validate.Struct(validate.Of{
		"git.branch": validate.Field(req.Branch, strings.Required),
		"git.hash": validate.Maybe(req.Hash, func(hash string) error {
			return validate.Field(hash, strings.Required)
		}),
	}); err != nil {
		return nil, err
	}

	vcs, hasVCS := app.VersionControl().TryGet()

	if !hasVCS {
		return nil, domain.ErrVersionControlNotConfigured
	}

	// Retrieve the latest commit to make sure the branch exists
	latestCommit, err := getLatestBranchCommit(ctx, vcs, req.Branch)

	if err != nil {
		return nil, validate.Wrap(err, "git.branch")
	}

	return Data{req.Branch, req.Hash.Get(latestCommit)}, nil
}

func (s *service) Fetch(ctx context.Context, deploymentCtx domain.DeploymentContext, depl domain.Deployment) error {
	logger := deploymentCtx.Logger()

	// Retrieve git url and token from the app
	app, err := s.reader.GetByID(ctx, depl.ID().AppID())

	if err != nil {
		logger.Error(err)
		return ErrAppRetrievedFailed
	}

	vcs, hasVCS := app.VersionControl().TryGet()

	// Could happen if the app vcs config has been removed since the deployment has been queued
	if !hasVCS {
		return domain.ErrVersionControlNotConfigured
	}

	data, ok := depl.Source().(Data)

	if !ok {
		return domain.ErrInvalidSourcePayload
	}

	logger.Stepf("cloning branch %s at %s from %s using token: %t", data.Branch, data.Hash, vcs.Url(), vcs.Token().HasValue())

	r, err := git.PlainCloneContext(ctx, deploymentCtx.BuildDirectory(), false, &git.CloneOptions{
		Auth:          getAuthMethod(vcs),
		SingleBranch:  true,
		ReferenceName: plumbing.NewBranchReferenceName(data.Branch),
		URL:           vcs.Url().String(),
		Progress:      logger,
	})

	if err != nil {
		logger.Error(err)
		return ErrGitCloneFailed
	}

	// Resolve short hash names if needed
	rev, err := r.ResolveRevision(plumbing.Revision(data.Hash))

	if err != nil {
		logger.Error(err)
		return ErrGitResolveFailed
	}

	w, err := r.Worktree()

	if err != nil {
		logger.Error(err)
		return ErrGitResolveFailed
	}

	if err = w.Checkout(&git.CheckoutOptions{
		Hash: *rev,
	}); err != nil {
		logger.Error(err)
		return ErrGitCheckoutFailed
	}

	return nil
}

func getAuthMethod(vcs domain.VersionControl) transport.AuthMethod {
	if token, isSet := vcs.Token().TryGet(); isSet {
		return &http.BasicAuth{
			Username: basicAuthUser,
			Password: token,
		}
	}

	return nil
}

func getLatestBranchCommit(ctx context.Context, vcs domain.VersionControl, branch string) (string, error) {
	branchRef := plumbing.NewBranchReferenceName(branch)
	refs, err := git.NewRemote(nil, &config.RemoteConfig{
		Name: "origin",
		URLs: []string{vcs.Url().String()},
	}).ListContext(ctx, &git.ListOptions{
		Auth: getAuthMethod(vcs),
	})

	if err != nil {
		return "", ErrGitRemoteNotReachable
	}

	for _, ref := range refs {
		if ref.Name() == branchRef {
			return ref.Hash().String(), nil
		}
	}

	return "", ErrGitBranchNotFound
}
