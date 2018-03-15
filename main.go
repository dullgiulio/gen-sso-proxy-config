package main

import (
	"encoding/json"
	"flag"
	"log"
	"os"
	"strings"
	"text/template"

	"github.com/dullgiulio/gen-sso-proxy-config/jsoncomments"
)

type server struct {
	Domains []string
	Port    int
	Name    string   `json:"-"`
	Aliases []string `json:"-"`
}

type proxy struct {
	Proto  string
	Domain string
	Port   int
	Path   string
}

type service struct {
	Name   string            `json:"-"`
	Mellon map[string]string // TODO
	Server server
	Proxy  map[string]proxy
	Public []string
}

type tmpldata struct {
	Mellon map[string]string
}

type confmap map[string]service

func process(c confmap) ([]service, error) {
	var i int
	srvs := make([]service, len(c))
	for k := range c {
		s := c[k]
		// Split first domain and others as ServerName and ServerAlias
		s.Server.Name = s.Server.Domains[0]
		s.Server.Aliases = s.Server.Domains[1:]
		// defaults for proxy
		for pk, p := range s.Proxy {
			if p.Proto == "" {
				p.Proto = "http"
			}
			if p.Port == 0 {
				p.Port = 80
			}
			if p.Path == "" {
				p.Path = "/"
			}
			s.Proxy[pk] = p
		}
		// Make service name available
		s.Name = k
		srvs[i] = s
		i++
	}
	return srvs, nil
}

type envmap map[string]string

func makeEnvmap(env []string) envmap {
	m := make(map[string]string)
	for i := range env {
		vals := strings.SplitN(env[i], "=", 2)
		m[vals[0]] = vals[1]
	}
	return m
}

func (e envmap) Get(k, defv string) string {
	if _, ok := e[k]; !ok {
		return defv
	}
	return e[k]
}

func main() {
	flag.Parse()
	cfile := flag.Arg(0)
	if cfile == "" {
		log.Fatal("usage: gen-sso-proxy-conf <config-file.json>")
	}
	fh, err := os.Open(cfile)
	if err != nil {
		log.Fatalf("cannot open configuration file: %v", err)
	}
	var conf confmap
	jcr := jsoncomments.NewReader(fh)
	dec := json.NewDecoder(jcr)
	if err := dec.Decode(&conf); err != nil {
		log.Fatalf("cannot read JSON configuration: %v", err)
	}
	servs, err := process(conf)
	if err != nil {
		log.Fatalf("cannot process configuration: %v", err)
	}
	tmpl, err := template.ParseFiles(flag.Arg(1))
	if err != nil {
		log.Fatalf("cannot parse template: %v", err)
	}
	env := makeEnvmap(os.Environ())
	vars := make(map[string]interface{})
	vars["Services"] = servs
	vars["Env"] = env
	if err := tmpl.Execute(os.Stdout, vars); err != nil {
		log.Fatalf("cannot render template: %v", err)
	}
}
