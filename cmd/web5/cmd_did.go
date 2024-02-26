package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/tbd54566975/web5-go/dids"
)

type didResolveCmd struct {
	URI string `arg:"" name:"uri" help:"The URI to resolve."`
}

func (c *didResolveCmd) Run(_ context.Context) error {
	result, err := dids.Resolve(c.URI)
	if err != nil {
		return err
	}

	jsonDID, err := json.MarshalIndent(result.Document, "", "  ")
	if err != nil {
		return err
	}

	fmt.Println(string(jsonDID))

	return nil
}

type didCmd struct {
	Resolve didResolveCmd `cmd:"" help:"Resolve a DID."`
}
