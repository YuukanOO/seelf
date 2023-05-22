package git

import (
	"context"
	"errors"
	"fmt"
	"os"
	sstrings "strings"

	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/go-git/go-git/v5/config"

	"github.com/YuukanOO/seelf/internal/deployment/infra/trigger"
	"github.com/YuukanOO/seelf/pkg/apperr"
	"github.com/YuukanOO/seelf/pkg/log"
	"github.com/YuukanOO/seelf/pkg/monad"
	"github.com/YuukanOO/seelf/pkg/ostools"
	"github.com/YuukanOO/seelf/pkg/validation"
	"github.com/YuukanOO/seelf/pkg/validation/strings"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
)

var (
	ErrGitRemoteNotReachable = apperr.New("git_remote_not_reachable")
	ErrGitBranchNotFound     = apperr.New("git_branch_not_found")
	ErrGitCloneFailed        = errors.New("git_clone_failed")
	ErrGitResolveFailed      = errors.New("git_resolve_failed")
	ErrGitCheckoutFailed     = errors.New("git_checkout_failed")
)

const (
	basicAuthUser = "seelf"
	separator     = "@"
)

type (
	Options interface {
		AppsDir() string
		LogsDir() string
	}

	Payload struct {
		Branch string              `json:"branch"`
		Hash   monad.Maybe[string] `json:"hash"`
	}

	service struct {
		reader  domain.AppsReader
		options Options
	}
)

// Builds a new trigger to process git deployments
func New(options Options, reader domain.AppsReader) trigger.Trigger {
	return &service{
		options: options,
		reader:  reader,
	}
}

func (*service) CanPrepare(payload any) bool {
	_, ok := payload.(Payload)
	return ok
}

func (s *service) Prepare(app domain.App, payload any) (domain.Meta, error) {
	p, ok := payload.(Payload)

	if !ok {
		return domain.Meta{}, domain.ErrInvalidTriggerPayload
	}

	if err := validation.Check(validation.Of{
		"branch": validation.Is(p.Branch, strings.Required),
		"hash": validation.Maybe(p.Hash, func(hash string) error {
			return validation.Is(hash, strings.Required)
		}),
	}); err != nil {
		return domain.Meta{}, err
	}

	if !app.VCS().HasValue() {
		return domain.Meta{}, domain.ErrVCSNotConfigured
	}

	// Retrieve the latest commit to make sure the branch exists
	latestCommit, err := getLatestBranchCommit(app.VCS().MustGet(), p.Branch)

	if err != nil {
		return domain.Meta{}, validation.WrapIfAppErr(err, "branch")
	}

	if !p.Hash.HasValue() {
		p.Hash = p.Hash.WithValue(latestCommit)
	}

	metaPayload := fmt.Sprintf("%s%s%s", p.Branch, separator, p.Hash.MustGet())

	return domain.NewMeta(domain.KindGit, metaPayload), nil
}

func (*service) CanFetch(meta domain.Meta) bool {
	return meta.Kind() == domain.KindGit
}

func (s *service) Fetch(ctx context.Context, depl domain.Deployment) error {
	logfile, err := ostools.OpenAppend(depl.LogPath(s.options.LogsDir()))

	if err != nil {
		return err
	}

	defer logfile.Close()

	logger := log.NewStepLogger(logfile)

	// Retrieve git url and token from the app
	app, err := s.reader.GetByID(context.Background(), depl.ID().AppID())

	if err != nil {
		logger.Error(err)
		return domain.ErrTriggerFetchFailed
	}

	// Could happen if the app vcs config has been removed since the deployment has been queued
	if !app.VCS().HasValue() {
		return domain.ErrVCSNotConfigured
	}

	buildDir := depl.Path(s.options.AppsDir())

	if err := os.RemoveAll(buildDir); err != nil {
		logger.Error(err)
		return domain.ErrTriggerFetchFailed
	}

	config := app.VCS().MustGet()

	// Retrieve the branch and hash for deployment payload
	// We use the LastIndex because an @ is a vaid character in a branch name
	data := depl.Trigger().Data()
	lastSeparatorIdx := sstrings.LastIndex(data, separator)
	branch, hash := data[:lastSeparatorIdx], data[lastSeparatorIdx+1:]

	logger.Stepf("cloning branch %s at %s from %s using token: %t", branch, hash, config.Url(), config.Token().HasValue())

	r, err := git.PlainCloneContext(ctx, buildDir, false, &git.CloneOptions{
		Auth:          getAuthMethod(config),
		SingleBranch:  true,
		ReferenceName: plumbing.NewBranchReferenceName(branch),
		URL:           config.Url().String(),
		Progress:      logfile,
	})

	if err != nil {
		logger.Error(err)
		return ErrGitCloneFailed
	}

	// Resolve short hash names if needed
	rev, err := r.ResolveRevision(plumbing.Revision(hash))

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

func getAuthMethod(vcs domain.VCSConfig) transport.AuthMethod {
	if vcs.Token().HasValue() {
		return &http.BasicAuth{
			Username: basicAuthUser,
			Password: vcs.Token().MustGet(),
		}
	}

	return nil
}

func getLatestBranchCommit(vcs domain.VCSConfig, branch string) (string, error) {
	branchRef := plumbing.NewBranchReferenceName(branch)
	refs, err := git.NewRemote(nil, &config.RemoteConfig{
		Name: "origin",
		URLs: []string{vcs.Url().String()},
	}).List(&git.ListOptions{
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
