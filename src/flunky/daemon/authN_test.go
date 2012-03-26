package daemon

import (
	"testing"
)

func TestAuthentication(t *testing.T) {
	auth := NewAuthInfo("/dev/null")
	ud := UserNode{"bar", false}
	auth.Users["foo"] = ud
	uda := UserNode{"baz", true}
	auth.Users["bar"] = uda

	valid, admin := auth.Authenticate("foo", "bar")
	if valid != true || admin == true {
		t.Error("Positive authentication failure")
	}

	valid, admin = auth.Authenticate("bar", "bar")
	if valid == true || admin == false {
		t.Error("Negative authentication failure")
	}

	valid, admin = auth.Authenticate("nothere", "")
	if valid == true || admin == true {
		t.Error("Nonexistent user failure")
	}
}
