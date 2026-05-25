package beads

import (
	"reflect"
	"sort"
	"testing"
)

func TestClientPublicMethodSurface(t *testing.T) {
	t.Parallel()

	clientType := reflect.TypeOf((*Client)(nil))
	got := make([]string, 0, clientType.NumMethod())
	for i := range clientType.NumMethod() {
		got = append(got, clientType.Method(i).Name)
	}
	sort.Strings(got)

	want := []string{
		"Close",
		"Comment",
		"List",
		"Ready",
		"Show",
		"Transition",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Client public methods = %v, want %v", got, want)
	}
}

func TestClientDoesNotExposeRawEscapeHatch(t *testing.T) {
	t.Parallel()

	clientType := reflect.TypeOf((*Client)(nil))
	for _, name := range []string{"Raw", "Run", "Exec", "Command", "Do"} {
		if _, ok := clientType.MethodByName(name); ok {
			t.Fatalf("Client exposes %s; raw CLI-shaped escape hatches are forbidden", name)
		}
	}
}
