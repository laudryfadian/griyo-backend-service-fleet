package platform

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"testing"
)

func TestDecodeRejectsUnknownFields(t *testing.T) {
	var input struct {
		Name string `json:"name"`
	}
	for _, body := range []string{`{"name":"agent","password":"secret"}`, `{"name":"agent"} {}`} {
		if e := Decode([]byte(body), &input); status.Code(e) != codes.InvalidArgument {
			t.Fatalf("expected invalid argument, got %v", e)
		}
	}
}
func TestHashAndEqual(t *testing.T) {
	if !Equal(Hash("token"), Hash("token")) {
		t.Fatal("equal tokens rejected")
	}
	if Equal(Hash("token"), Hash("other")) {
		t.Fatal("different tokens accepted")
	}
}
