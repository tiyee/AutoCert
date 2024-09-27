package applicant

import (
	"crypto"
	"github.com/go-acme/lego/v4/registration"
)

type User struct {
	Email        string
	Registration *registration.Resource
	key          crypto.PrivateKey
}

func (u *User) GetEmail() string {
	return u.Email
}

func (u *User) GetRegistration() *registration.Resource {
	return u.Registration
}

func (u *User) GetPrivateKey() crypto.PrivateKey {
	return u.key
}

func (u *User) SetEmail(email string) {
	u.Email = email
}

func (u *User) SetPrivateKey(privateKey crypto.PrivateKey) {
	u.key = privateKey
}

func (u *User) SetRegistration(registration *registration.Resource) {
	u.Registration = registration
}
