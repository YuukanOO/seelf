package domain_test

import (
	"testing"

	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/pkg/testutil"
)

func Test_Meta(t *testing.T) {
	t.Run("could be created", func(t *testing.T) {
		var (
			kind    domain.Kind = "meta kind"
			payload             = "meta payload"
		)

		meta := domain.NewMeta(kind, payload)

		testutil.Equals(t, kind, meta.Kind())
		testutil.Equals(t, payload, meta.Data())
	})
}

func Test_Kind(t *testing.T) {
	t.Run("could check wether it represents a VCS managed source", func(t *testing.T) {
		var rawKind domain.Kind = "raw"

		testutil.IsFalse(t, rawKind.IsVCS())
		testutil.IsTrue(t, domain.KindGit.IsVCS())
	})
}
