package domain

// Contains stuff related to how the deployment has been triggered.
// The inner data depends on the Trigger which has been requested, this is
// why I use primitive types here.
type Meta struct {
	kind Kind
	data string
}

// Builds a new deployment meta from a kind and any additional data.
// You should never call it outside a Trigger implementation since
// it will be picked by a job to actually initiate the deployment.
func NewMeta(kind Kind, data string) Meta {
	return Meta{kind, data}
}

func (m Meta) Kind() Kind   { return m.kind }
func (m Meta) Data() string { return m.data }

// Specific type to represents a trigger kind to add helper methods on it.
type Kind string

// Represents a trigger which will fetch source code from Git. It is defined here
// because it has special meanings (see below).
const KindGit Kind = "git"

// Gets wether this kind of trigger is a version controlled one.
// For now it only checks for git but in the future, maybe there will be other
// VCS to deal with.
func (k Kind) IsVCS() bool {
	return k == KindGit
}
