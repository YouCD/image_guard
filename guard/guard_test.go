package guard

import (
	"fmt"
	"testing"

	"github.com/google/go-containerregistry/pkg/name"
)

type reference struct {
	name.Reference
}

func Test_newRef(t *testing.T) {
	ref, err := newRef("http://0.0.0.0:50000", "whyour/qinglong:latest")
	if err != nil {
		t.Error(err)
		return
	}

	t.Log(ref)
	t.Log(ref.Name())
	t.Log(ref.Identifier())
	t.Log(ref.Context().Registry)
	t.Log(ref.Context().RepositoryStr(), ref.Identifier())
	t.Log(ref.Context().RegistryStr())
	t.Log(ref.Context())
	t.Log(ref.Context().Scheme())
	t.Log(ref.Context().Scheme())

	reference, err := name.ParseReference("lscr.io/linuxserver/transmission:latest")
	if err != nil {
		t.Error(err)
	}
	fmt.Println()
	fmt.Println()
	fmt.Println()
	t.Log(reference)
	t.Log(reference.Name())
	t.Log(reference.Identifier())
	t.Log(reference.Context().Registry)
	t.Log(reference.Context().RepositoryStr(), ref.Identifier())
	t.Log(reference.Context().RegistryStr())
	t.Log(reference.Context())
}
