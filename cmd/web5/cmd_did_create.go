package main

import (
	"encoding/json"
	"fmt"

	"github.com/tbd54566975/web5-go/dids/didjwk"
	"github.com/tbd54566975/web5-go/dids/didweb"
)

type didCreateCMD struct {
	JWK didCreateJWKCMD `cmd:"" help:"Create a did:jwk."`
	Web didCreateWebCMD `cmd:"" help:"Create a did:web."`
}

type didCreateJWKCMD struct{}

func (c *didCreateJWKCMD) Run() error {
	did, err := didjwk.Create()
	if err != nil {
		return err
	}

	portableDID, err := did.ToPortableDID()
	if err != nil {
		return err
	}

	jsonDID, err := json.MarshalIndent(portableDID, "", "  ")
	if err != nil {
		return err
	}

	fmt.Println(string(jsonDID))

	return nil
}

type didCreateWebCMD struct {
	Domain string `arg:"" help:"The domain name for the DID." required:""`
}

func (c *didCreateWebCMD) Run() error {
	did, err := didweb.Create(c.Domain)
	if err != nil {
		return err
	}

	portableDID, err := did.ToPortableDID()
	if err != nil {
		return err
	}

	jsonDID, err := json.MarshalIndent(portableDID, "", "  ")
	if err != nil {
		return err
	}

	fmt.Println(string(jsonDID))

	return nil
}
