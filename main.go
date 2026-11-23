package main

import (
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"strings"

	fzf "github.com/junegunn/fzf/src"
	"github.com/junegunn/fzf/src/protector"
)

var version = "0.1.0"
var revision = "devel"

const upstreamVersion = "0.75"

//go:embed shell/key-bindings.bash
var bashKeyBindings []byte

//go:embed shell/completion.bash
var bashCompletion []byte

//go:embed shell/key-bindings.zsh
var zshKeyBindings []byte

//go:embed shell/completion.zsh
var zshCompletion []byte

//go:embed shell/key-bindings.fish
var fishKeyBindings []byte

//go:embed shell/key-bindings.nu
var nushellKeyBindings []byte

//go:embed shell/completion.nu
var nushellCompletion []byte

//go:embed shell/completion.fish
var fishCompletion []byte

//go:embed man/man1/fzfp.1
var manPage []byte

func printScript(label string, content []byte) {
	fmt.Println("### " + label + " ###")
	fmt.Println(strings.TrimSpace(string(content)))
	fmt.Println("### end: " + label + " ###")
}

func exit(code int, err error) {
	if code == fzf.ExitError && err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
	}
	os.Exit(code)
}

func main() {
	protector.Protect()

	options, err := fzf.ParseOptions(true, os.Args[1:])
	if err != nil {
		exit(fzf.ExitError, err)
		return
	}
	if options.Bash {
		printScript("key-bindings.bash", bashKeyBindings)
		printScript("completion.bash", bashCompletion)
		return
	}
	if options.Zsh {
		printScript("key-bindings.zsh", zshKeyBindings)
		printScript("completion.zsh", zshCompletion)
		return
	}
	if options.Fish {
		printScript("key-bindings.fish", fishKeyBindings)
		printScript("completion.fish", fishCompletion)
		return
	}
	if options.Nushell {
		printScript("key-bindings.nu", nushellKeyBindings)
		printScript("completion.nu", nushellCompletion)
		return
	}
	if options.Help {
		usage := strings.Replace(fzf.Usage, "fzf is an", "fzfp is an", 1)
		usage = strings.Replace(usage, "Usage: fzf [options]", "Usage: fzfp [options]", 1)
		usage = strings.Replace(usage, "Project URL: https://github.com/junegunn/fzf", "Project URL: https://github.com/zhiyozhao/fzf-pinyin\nUpstream: https://github.com/junegunn/fzf", 1)
		fmt.Print(strings.Replace(usage, "fzf --man", "fzfp --man", 1))
		return
	}
	if options.Version {
		label := fmt.Sprintf("fzfp %s (based on fzf %s)", version, upstreamVersion)
		if len(revision) > 0 {
			fmt.Printf("%s (%s)\n", label, revision)
		} else {
			fmt.Println(label)
		}
		return
	}
	if options.Man {
		file := fzf.WriteTemporaryFile([]string{string(manPage)}, "\n")
		if len(file) == 0 {
			fmt.Print(string(manPage))
			return
		}
		defer os.Remove(file)
		cmd := exec.Command("man", file)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		if err := cmd.Run(); err != nil {
			fmt.Print(string(manPage))
		}
		return
	}

	code, err := fzf.Run(options)
	exit(code, err)
}
