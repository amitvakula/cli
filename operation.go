package main

import (
	. "fmt"
	"github.com/inconshreveable/log15"
	"os"
	"path/filepath"
	"strings"

	"flywheel.io/deja/job"
	"flywheel.io/deja/provider"
)

func (p *Project) Use(args []string) {
	if len(args) != 1 {
		Println("use only takes 1 argument: the name of the gear base.")
		os.Exit(1)
	}

	base := args[0]
	p.Base = base

	switch base {
	case "python3":
		Println("Downloading python3....")
		p.Provision("http", "https://storage.googleapis.com/flywheel/etc/deja-flak/anaconda-4.0.0.tar.gz", "vu0:sha384:PtQAKtjrhhH2FBawPhzDxyAYAM6R7mW935Nd9O0reRnfVEuI8F_HkWFwYZbBVax3")
	default:
		Println("Unrecognized gear base.")
		os.Exit(1)
	}
}

func (p *Project) Provision(itype, uri, vu string) {
	i := &job.Input{
		Type:     itype,
		Location: "/",
		URI:      uri,
		VuID:     vu,
	}
	f := &job.Formula{
		Inputs: []*job.Input{
			i,
		},
		Target: job.Target{
			Command: []string{"echo", "Gear base downloaded & tested."},
			Env:     map[string]string{},
			Dir:     "/",
		},
		Outputs: []*job.Output{},
	}

	log := log15.New()
	log.SetHandler(log15.LvlFilterHandler(log15.LvlError, log15.StderrHandler))
	result := provider.Run(f, provider.Logger(log))

	p.Input = i
	p.Input.VuID = result.Formula.Inputs[0].VuID // Workaround until deja doesn't modify result from caching
	p.Save()

	os.Exit(result.Result.ExitCode)
}

func (p *Project) Import(args []string) {
	if len(args) != 1 {
		Println("use only takes 1 argument: the filepath or HTTP url of the custom gear base.")
		os.Exit(1)
	}

	base := args[0]
	itype := "file"

	if strings.HasPrefix(base, "http") {
		itype = "http"
	}

	p.Base = "custom"
	p.Provision(itype, base, "")
}

func (p *Project) Run(args []string) {
	dir := filepath.Dir(p.Path)

	f := &job.Formula{
		Inputs: []*job.Input{
			p.Input,
			&job.Input{
				Type:     "bind",
				URI:      dir,
				Location: dir,
			},
		},
		Target: job.Target{
			Command: args,
			Env:     p.Env,
			Dir:     dir,
		},
		Outputs: []*job.Output{},
	}

	log := log15.New()
	log.SetHandler(log15.LvlFilterHandler(log15.LvlError, log15.StderrHandler))
	result := provider.Run(f, provider.Logger(log))
	os.Exit(result.Result.ExitCode)
}

func (p *Project) Frun(args []string) {
	if len(args) != len(p.Inputs) {
		Println("There are", len(p.Inputs), "inputs specified in flywheel.json but you gave", len(args))
		os.Exit(1)
	}

	// Could use go's templating system instead
	input := 0
	for i, x := range p.Args {
		if strings.HasPrefix(x, "{{") {
			p.Args[i] = args[input]
			input++
		}
	}

	p.Run(p.Args)
}
