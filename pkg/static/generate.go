// +build ignore

package main

import (
	"github.com/linkerd/linkerd-smi/pkg/static"
	"github.com/shurcooL/vfsgen"
	log "github.com/sirupsen/logrus"
)

func main() {
	err := vfsgen.Generate(static.Templates, vfsgen.Options{
		Filename:     "generated_smi_templates.gogen.go",
		PackageName:  "static",
		BuildTags:    "production",
		VariableName: "Templates",
	})
	if err != nil {
		log.Fatalln(err)
	}
}
