package guard

import (
	"fmt"
	"github.com/google/go-containerregistry/pkg/name"
	"testing"
)

func Test_newRef(t *testing.T) {
	ref, err := newRef("https://127.0.0.1:5000", "lscr.io/linuxserver/transmission:latest")
	if err != nil {
		t.Error(err)
	}
	t.Log(ref)
	t.Log(ref.Name())
	t.Log(ref.Identifier())
	t.Log(ref.Context().Registry)
	t.Log(ref.Context().RepositoryStr(), ref.Identifier())
	t.Log(ref.Context().RegistryStr())
	t.Log(ref.Context().String())

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
