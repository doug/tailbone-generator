package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

var (
	reApplication = regexp.MustCompile("^application:[ \t]*([a-zA-Z0-9-]+)[ \t]*")
	reVersion     = regexp.MustCompile("^version:[ \t]*([a-zA-Z0-9-]+)[ \t]*")
)

func pipeCmd(command string, args ...string) error {
	cmd := exec.Command(command, args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	err = cmd.Start()
	if err != nil {
		return err
	}
	go io.Copy(os.Stdout, stdout)
	go io.Copy(os.Stderr, stderr)
	cmd.Wait()
	return nil
}

func run(action string) (err error) {
	switch action {
	case "init":
		if _, err = os.Stat("tailbone"); err == nil {
			return errors.New("Tailbone is already initialized.")
		}
		if _, err = os.Stat(".git"); os.IsNotExist(err) {
			return errors.New("Your current directory must be a git repo.")
		}
		args := strings.Split("git submodule add -b reorg https://github.com/doug/tailbone.git", " ")
		err = pipeCmd(args[0], args[1:]...)
		if err != nil {
			return errors.New("Problem checking out the tailbone git submodule.")
		}
		if apptemplate, err := os.Open("tailbone/app.template.yaml"); err == nil {
			if appyaml, err := os.Create("app.yaml"); err == nil {
				_, err = io.Copy(appyaml, apptemplate)
			}
		}
		if _, err = os.Stat("app/index.html"); os.IsNotExist(err) {
			os.Mkdir("app", 0644)
			if index, err := os.Create("app/index.html"); err == nil {
				index.WriteString(INDEX_TEMPLATE)
			}
		}
	case "serve":
		if _, err := os.Stat("tailbone"); os.IsNotExist(err) {
			return errors.New("Must run 'tailbone init' first.")
		}
		if _, err := exec.LookPath("dev_appserver.py"); err != nil {
			return errors.New("You must have the cloud sdk for python installed on your system. Download it at https://developers.google.com/cloud/sdk/")
		}
		args := flag.Args()[1:]
		args = append(args, "tailbone")
		err = pipeCmd("dev_appserver.py", args...)
	case "deploy":
		if _, err := os.Stat("tailbone"); os.IsNotExist(err) {
			return errors.New("Must run 'tailbone init' first.")
		}
		if _, err := os.Stat("app.yaml"); os.IsNotExist(err) {
			return errors.New("Must have app.yaml, this can be copied from tailbone/app.template.yaml.")
		}
		if _, err := exec.LookPath("appcfg.py"); err != nil {
			return errors.New("You must have the cloud sdk for python installed on your system. Download it at https://developers.google.com/cloud/sdk/")
		}
		version := flag.Arg(1)
		if version == "" {
			return errors.New("Must provide a version name. Example: tailbone deploy master")
		}
		if appbytes, err := ioutil.ReadFile("app.yaml"); err == nil {
			appyaml := string(appbytes)
			matches := reApplication.FindStringSubmatch(appyaml)
			if len(matches) != 2 {
				return errors.New("Incorrectly formated app.yaml could not find application.")
			}
			application := matches[1]
			if application == DEFAULT_APPLICATION {
				fmt.Printf("You must enter an application id. If you haven't already created a project, do so now at http://cloud.google.com\n")
				fmt.Scanf("%s", &application)
				appyaml = reApplication.ReplaceAllStringFunc(appyaml, func(s string) string { return "application: " + application })
			}
			appyaml = reVersion.ReplaceAllStringFunc(appyaml, func(s string) string { return "version: " + version })
			ioutil.WriteFile("app.yaml", []byte(appyaml), 0644)
			fmt.Println(version)
			err = pipeCmd("appcfg.py", "update", "--oauth2", "tailbone")
		}
	case "update":
		if _, err := os.Stat("tailbone"); os.IsNotExist(err) {
			return errors.New("Must run 'tailbone init' first.")
		}
		if _, err := exec.LookPath("git"); err != nil {
			return errors.New("You must have git installed.")
		}
		log.Printf("Update doesn't work yet do this manually by updating the git submodule tailbone.")
	default:
		flag.Usage()
	}
	return
}

func main() {
	flag.Parse()

	if flag.NArg() < 1 {
		flag.Usage()
		os.Exit(1)
	}

	flag.Usage = func() {
		fmt.Printf(USAGE_TEMPLATE)
	}

	cmdname := flag.Arg(0)

	if err := run(cmdname); err != nil {
		log.Println(err)
		os.Exit(1)
	}
}

const (
	DEFAULT_APPLICATION = "your-application-id"
	INDEX_TEMPLATE      = `<!doctype html>
<html>
<head></head>
<body>
hello tailbone
</body>
</html>
`
	USAGE_TEMPLATE = `
tailbone init:
  Initialize tailbone
tailbone serve:
  Serve tailbone locally
tailbone deploy {version_name}:
  Deploy tailbone to AppEngine
tailbone update:
  Update the version of tailbone to latest
tailbone version:
  Get the version of tailbone

`
)
