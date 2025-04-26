package guard

import (
	"testing"
)

func Test_newRef(t *testing.T) {
	ref, err := newRef("https://127.0.0.1:5000", "aaa/nginx")
	if err != nil {
		t.Error(err)
	}
	t.Log(ref)
	t.Log(ref.Name())
	t.Log(ref.Identifier())
	t.Log(ref.Context().Registry)
	t.Log(ref.Context().RepositoryStr(), ref.Identifier())
	t.Log(ref.Context().RegistryStr())
	t.Log(ref.Context())
}
