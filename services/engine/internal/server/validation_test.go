package server

import (
	"strings"
	"testing"
)

func TestResourceNameValidation(t *testing.T) {
	for _, tc := range []struct {
		name  string
		valid bool
	}{{"Policy", true}, {"", false}, {strings.Repeat("a", 120), true}, {strings.Repeat("a", 121), false}, {strings.Repeat("é", 60), true}, {strings.Repeat("é", 61), false}, {"a\nb", false}} {
		if resourceNameValid(tc.name) != tc.valid {
			t.Errorf("name validation mismatch: %q", tc.name)
		}
	}
}
func TestInvitationEmailValidation(t *testing.T) {
	for _, tc := range []struct {
		email string
		valid bool
	}{{"teammate+demo@example.com", true}, {"person@optitrace.test", true}, {"a@b", false}, {"a@@example.com", false}, {"Name <a@example.com>", false}, {"a b@example.com", false}, {"a..b@example.com", false}, {"a@-example.com", false}, {strings.Repeat("a", 65) + "@example.com", false}, {"a@" + strings.Repeat("b", 64) + ".com", false}} {
		if invitationEmailValid(tc.email) != tc.valid {
			t.Errorf("email validation mismatch: %q", tc.email)
		}
	}
}
