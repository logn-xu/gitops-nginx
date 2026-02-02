package main

import (
	"embed"

	"github.com/logn-xu/gitops-nginx/cmd/gitops-nginx/cmd"
)

//go:embed dist/*
var dist embed.FS

func main() {
	cmd.SetDist(dist)
	cmd.Execute()
}
