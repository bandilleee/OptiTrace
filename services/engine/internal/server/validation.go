package server

import (
	"net/mail"
	"regexp"
	"strings"
	"unicode"
)

func resourceNameValid(name string) bool {
	if name == "" || len(name) > 120 {
		return false
	}
	for _, r := range name {
		if unicode.IsControl(r) {
			return false
		}
	}
	return true
}

// Invitations accept conventional Google account addresses, never display names.
var invitationAddress = regexp.MustCompile("^[A-Za-z0-9!#$%&'*+/=?^_`{|}~-]+(?:\\.[A-Za-z0-9!#$%&'*+/=?^_`{|}~-]+)*@[A-Za-z0-9](?:[A-Za-z0-9-]*[A-Za-z0-9])?(?:\\.[A-Za-z0-9](?:[A-Za-z0-9-]*[A-Za-z0-9])?)+$")

func invitationEmailValid(email string) bool {
	if len(email) > 254 || !invitationAddress.MatchString(email) {
		return false
	}
	address, err := mail.ParseAddress(email)
	if err != nil || address.Address != email {
		return false
	}
	parts := strings.Split(email, "@")
	if len(parts[0]) > 64 {
		return false
	}
	for _, part := range strings.Split(parts[1], ".") {
		if len(part) > 63 {
			return false
		}
	}
	return true
}
