/*
gohack $module...

	fetch module if needed into $GOHACK/$module
	checkout correct commit
	"is not clean" failure if it's been changed but at right commit
	"is not clean; will not update" failure if changed and at wrong commit
	add replace statement to go.mod

gohack -u $module

	remove replace statement from go.mod
*/
package main

import (
	"flag"
	"fmt"
	"os"

	"gopkg.in/errgo.v2/fmt/errors"
)

/*
As the amount of functionality grows, it seems like we should consider having subcommands.

A possible set of commands:

 gohack get [-vcs] [-u] [-f] [module...]
Get gets the modules at the current version and adds replace statements to the go.mod file if they're not already replaced.
If the -u flag is provided, the source code will also be updated to the current version if it's clean.
If the -f flag is provided with -u, the source code will be updated even if it's not clean.
If the -vcs flag is provided, it also checks out VCS information for the modules. If the modules were already gohacked in non-VCS mode, gohack switches them to VCS mode, preserving any changes made (this might result in the directory moving).

With no module arguments and the -u flag, it will try to update all currently gohacked modules.

gohack status [

Status prints
gohack diff module
Diff prints (in git style) changes that have been made to the module since it was checked out.

gohack rm [-f] module...
Rm removes the gohack directory if it is clean and then runs gohack undo. If the -f flag is provided, the directory is removed even if it's not clean.

 gohack undo [module...]
Undo removes the replace statements for the modules. If no modules are provided, it will undo all gohack replace statements. The gohack module directories are unaffected.

gohack dir [-vcs] [module...]
Dir prints the gohack module directory names for the given modules. If no modules are given, all the currently gohacked module directories are printed. If the -vcs flag is provided, the directory to be used in VCS mode is printed. Unlike the other subcommands, the modules don't need to be referenced by the current module.
*/

var (
	printCommands = flag.Bool("x", false, "show executed commands")
	dryRun        = flag.Bool("n", false, "print but do not execute update commands")
)

var (
	exitCode = 0
	cwd      = "."
)

var commands = []*Command{
	getCommand,
	undoCommand,
	statusCommand,
}

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: gohack module...\n")
		flag.PrintDefaults()
		fmt.Print(`
The gohack command checks out Go module dependencies
into a directory where they can be edited.
By default it extracts module source code into $HOME/gohack/<module>.
By default, it also tries to check out the version control information into that
directory and update it to the expected version. If the directory
already exists, it will be updated in place.

With no arguments, gohack prints all modules
that are currently replaced by local directories.

The -u flag can be used to revert to the non-gohacked
module versions. It only removes the relevant replace
statements from the go.mod file - it does not change any
of the directories referred to. With the -u flag and no
arguments, all replace statements that refer to directories will
be removed.
`)
		os.Exit(2)
	}
	flag.Parse()
	main1()
	os.Exit(exitCode)
}

func main1() {
	if dir, err := os.Getwd(); err == nil {
		cwd = dir
	} else {
		errorf("cannot get current working directory: %v", err)
	}
	flag.Usage = func() {
		mainUsage(os.Stderr)
		os.Exit(2)
	}
	flag.Parse()
	if flag.NArg() == 0 {
		mainUsage(os.Stderr)
		os.Exit(2)
	}
	cmdName := flag.Arg(0)
	args := flag.Args()[1:]
	if cmdName == "help" {
		runHelp(args)
		os.Exit(0)
	}
	for _, cmd := range commands {
		if cmd.Name() != cmdName {
			continue
		}
		cmd.Flag.Usage = func() { cmd.Usage() }
		cmd.Flag.Parse(args)
		cmd.Run(cmd, cmd.Flag.Args())
		os.Exit(exitCode)
	}
	errorf("gohack %s: unknown command\nRun 'gohack help' for usage\n", cmdName)
	os.Exit(2)
}

const debug = false

func errorf(f string, a ...interface{}) {
	fmt.Fprintln(os.Stderr, fmt.Sprintf(f, a...))
	if debug {
		for _, arg := range a {
			if err, ok := arg.(error); ok {
				fmt.Fprintf(os.Stderr, "error: %s\n", errors.Details(err))
			}
		}
	}
	exitCode = 1
}
