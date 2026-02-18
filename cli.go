package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	appcore "github.com/jackhorton/veil/internal/app"
	"github.com/jackhorton/veil/internal/tui"
)

func runCLI(args []string) error {
	application, err := appcore.NewApp()
	if err != nil {
		return err
	}
	if len(args) == 0 {
		return tui.RunTUI(application)
	}
	switch args[0] {
	case "help", "-h", "--help":
		printHelp()
		return nil
	case "init":
		return cmdInit(application, args[1:])
	case "set":
		return cmdSet(application, args[1:])
	case "get":
		return cmdGet(application, args[1:])
	case "import":
		return cmdImport(application, args[1:])
	case "export":
		return cmdExport(application, args[1:])
	case "run":
		return cmdRun(application, args[1:])
	case "sync":
		return cmdSync(application, args[1:])
	case "list":
		return cmdList(application, args[1:])
	case "ls":
		return cmdLS(application, args[1:])
	case "rm":
		return cmdRM(application, args[1:])
	case "link":
		return cmdLink(application, args[1:])
	default:
		return fmt.Errorf("unknown command %q (run `veil --help`)", args[0])
	}
}

func cmdInit(app *appcore.App, args []string) error {
	args = reorderFlags(args, map[string]bool{"--key-storage": true, "--machine-name": true, "--link": false})
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	keyStorage := fs.String("key-storage", "file", "key storage backend: file or keychain")
	machineName := fs.String("machine-name", "", "machine name")
	link := fs.Bool("link", false, "create/link GitHub gist after init")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if err := app.Init(*keyStorage, *machineName); err != nil {
		return err
	}
	if *link {
		if err := app.Link("", ""); err != nil {
			return err
		}
	}
	fmt.Printf("Initialized Veil at %s\n", app.HomeDir)
	return nil
}

func cmdSet(app *appcore.App, args []string) error {
	args = reorderFlags(args, map[string]bool{"-p": true, "--group": true})
	fs := flag.NewFlagSet("set", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	projectFlag := fs.String("p", "", "project override")
	group := fs.String("group", "", "group label override")
	if err := fs.Parse(args); err != nil {
		return err
	}
	remaining := fs.Args()
	if len(remaining) < 2 {
		return errors.New("usage: veil set KEY VALUE [-p project] [--group group]")
	}
	project, path, err := app.ResolveProject(*projectFlag)
	if err != nil {
		return err
	}
	bundle, err := app.LoadProject(project, path)
	if err != nil {
		return err
	}
	key := remaining[0]
	value := strings.Join(remaining[1:], " ")
	created := appcore.UpsertSecret(bundle, key, value, *group)
	if err := app.SaveProject(bundle); err != nil {
		return err
	}
	if created {
		fmt.Printf("Added %s to %s\n", key, project)
	} else {
		fmt.Printf("Updated %s in %s\n", key, project)
	}
	return nil
}

func cmdGet(app *appcore.App, args []string) error {
	args = reorderFlags(args, map[string]bool{"-p": true})
	fs := flag.NewFlagSet("get", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	projectFlag := fs.String("p", "", "project override")
	if err := fs.Parse(args); err != nil {
		return err
	}
	remaining := fs.Args()
	if len(remaining) < 1 {
		return errors.New("usage: veil get KEY [-p project]")
	}
	project, path, err := app.ResolveProject(*projectFlag)
	if err != nil {
		return err
	}
	bundle, err := app.LoadProject(project, path)
	if err != nil {
		return err
	}
	secret, ok := appcore.GetSecret(bundle, remaining[0])
	if !ok {
		return fmt.Errorf("key %q not found in project %q", remaining[0], project)
	}
	fmt.Println(secret.Value)
	return nil
}

func cmdImport(app *appcore.App, args []string) error {
	args = reorderFlags(args, map[string]bool{"-p": true, "--skip-existing": false})
	fs := flag.NewFlagSet("import", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	projectFlag := fs.String("p", "", "project override")
	skipExisting := fs.Bool("skip-existing", false, "skip duplicate keys")
	if err := fs.Parse(args); err != nil {
		return err
	}
	remaining := fs.Args()
	if len(remaining) < 1 {
		return errors.New("usage: veil import FILE|- [-p project] [--skip-existing]")
	}
	var raw []byte
	var err error
	if remaining[0] == "-" {
		raw, err = io.ReadAll(os.Stdin)
	} else {
		raw, err = os.ReadFile(remaining[0])
	}
	if err != nil {
		return err
	}
	pairs, err := appcore.ParseEnvContent(string(raw))
	if err != nil {
		return err
	}
	project, path, err := app.ResolveProject(*projectFlag)
	if err != nil {
		return err
	}
	bundle, err := app.LoadProject(project, path)
	if err != nil {
		return err
	}
	added := 0
	updated := 0
	skipped := 0
	for _, pair := range pairs {
		if _, exists := appcore.GetSecret(bundle, pair.Key); exists && *skipExisting {
			skipped++
			continue
		}
		created := appcore.UpsertSecret(bundle, pair.Key, pair.Value, "")
		if created {
			added++
		} else {
			updated++
		}
	}
	if err := app.SaveProject(bundle); err != nil {
		return err
	}
	fmt.Printf("Imported %d keys (%d added, %d updated, %d skipped) into %s\n", len(pairs), added, updated, skipped, project)
	return nil
}

func cmdExport(app *appcore.App, args []string) error {
	args = reorderFlags(args, map[string]bool{"--format": true, "--out": true, "-p": true})
	fs := flag.NewFlagSet("export", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	format := fs.String("format", "", "export format: env|json")
	outPath := fs.String("out", "", "output path (stdout when omitted)")
	projectFlag := fs.String("p", "", "project override")
	if err := fs.Parse(args); err != nil {
		return err
	}
	remaining := fs.Args()
	project := ""
	if len(remaining) > 0 {
		project = remaining[0]
	}
	if project == "" {
		project = *projectFlag
	}
	resolvedName, path, err := app.ResolveProject(project)
	if err != nil {
		return err
	}
	bundle, err := app.LoadProject(resolvedName, path)
	if err != nil {
		return err
	}
	selectedFormat := strings.ToLower(strings.TrimSpace(*format))
	if selectedFormat == "" {
		selectedFormat = app.ExportFormatPreference()
	}
	if selectedFormat == "" {
		selectedFormat = "env"
	}
	var rendered string
	switch selectedFormat {
	case "env":
		rendered = appcore.RenderEnv(bundle)
	case "json":
		rendered, err = appcore.RenderProjectJSON(bundle)
		if err != nil {
			return err
		}
	default:
		return errors.New("invalid format (use env or json)")
	}
	if *outPath == "" {
		fmt.Print(rendered)
		return nil
	}
	abs, err := filepath.Abs(*outPath)
	if err != nil {
		return err
	}
	abs = filepath.Clean(abs)
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(abs, []byte(rendered), 0o600); err != nil {
		return err
	}
	fmt.Printf("Exported %s (%s) to %s\n", resolvedName, selectedFormat, abs)
	return nil
}

func cmdRun(app *appcore.App, args []string) error {
	idx := -1
	for i, arg := range args {
		if arg == "--" {
			idx = i
			break
		}
	}
	if idx == -1 {
		return errors.New("usage: veil run [-p project] -- COMMAND")
	}
	left := reorderFlags(args[:idx], map[string]bool{"-p": true})
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	projectFlag := fs.String("p", "", "project override")
	if err := fs.Parse(left); err != nil {
		return err
	}
	commandArgs := args[idx+1:]
	if len(commandArgs) == 0 {
		return errors.New("usage: veil run [-p project] -- COMMAND")
	}
	project, path, err := app.ResolveProject(*projectFlag)
	if err != nil {
		return err
	}
	bundle, err := app.LoadProject(project, path)
	if err != nil {
		return err
	}
	cmd := exec.Command(commandArgs[0], commandArgs[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()
	for _, secret := range bundle.Secrets {
		cmd.Env = append(cmd.Env, secret.Key+"="+secret.Value)
	}
	return cmd.Run()
}

func cmdSync(app *appcore.App, args []string) error {
	args = reorderFlags(args, map[string]bool{"--token": true})
	fs := flag.NewFlagSet("sync", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	token := fs.String("token", "", "GitHub token override")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if err := app.Sync(*token); err != nil {
		return err
	}
	fmt.Println("Sync complete")
	return nil
}

func cmdList(app *appcore.App, args []string) error {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	if err := fs.Parse(args); err != nil {
		return err
	}
	projects, err := app.ListProjects()
	if err != nil {
		return err
	}
	if len(projects) == 0 {
		fmt.Println("No projects yet")
		return nil
	}
	fmt.Println("PROJECT\tSECRETS\tPATH")
	for _, project := range projects {
		fmt.Printf("%s\t%d\t%s\n", project.Name, project.Count, project.Path)
	}
	return nil
}

func cmdLS(app *appcore.App, args []string) error {
	args = reorderFlags(args, map[string]bool{"-p": true})
	fs := flag.NewFlagSet("ls", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	projectFlag := fs.String("p", "", "project override")
	if err := fs.Parse(args); err != nil {
		return err
	}
	remaining := fs.Args()
	project := *projectFlag
	if len(remaining) > 0 {
		project = remaining[0]
	}
	resolvedName, path, err := app.ResolveProject(project)
	if err != nil {
		return err
	}
	bundle, err := app.LoadProject(resolvedName, path)
	if err != nil {
		return err
	}
	if len(bundle.Secrets) == 0 {
		fmt.Printf("No secrets in %s\n", resolvedName)
		return nil
	}
	sorted := append([]appcore.Secret(nil), bundle.Secrets...)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Group == sorted[j].Group {
			return sorted[i].Key < sorted[j].Key
		}
		return sorted[i].Group < sorted[j].Group
	})
	currentGroup := ""
	for _, secret := range sorted {
		if secret.Group != currentGroup {
			currentGroup = secret.Group
			fmt.Printf("[%s]\n", currentGroup)
		}
		fmt.Printf("  %s=%s\n", secret.Key, appcore.MaskValue(secret.Value))
	}
	return nil
}

func cmdRM(app *appcore.App, args []string) error {
	args = reorderFlags(args, map[string]bool{"-p": true, "-y": false})
	fs := flag.NewFlagSet("rm", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	projectFlag := fs.String("p", "", "project override")
	yes := fs.Bool("y", false, "skip confirmation")
	if err := fs.Parse(args); err != nil {
		return err
	}
	remaining := fs.Args()
	if len(remaining) < 1 {
		return errors.New("usage: veil rm KEY [-p project] [-y]")
	}
	key := remaining[0]
	project, path, err := app.ResolveProject(*projectFlag)
	if err != nil {
		return err
	}
	bundle, err := app.LoadProject(project, path)
	if err != nil {
		return err
	}
	if _, exists := appcore.GetSecret(bundle, key); !exists {
		return fmt.Errorf("key %q not found in %q", key, project)
	}
	if !*yes {
		fmt.Printf("Delete %s from %s? [y/N]: ", key, project)
		in := bufio.NewScanner(os.Stdin)
		if !in.Scan() || strings.ToLower(strings.TrimSpace(in.Text())) != "y" {
			fmt.Println("Cancelled")
			return nil
		}
	}
	if !appcore.RemoveSecret(bundle, key) {
		return fmt.Errorf("key %q not found in %q", key, project)
	}
	if err := app.SaveProject(bundle); err != nil {
		return err
	}
	fmt.Printf("Deleted %s from %s\n", key, project)
	return nil
}

func cmdLink(app *appcore.App, args []string) error {
	args = reorderFlags(args, map[string]bool{"--token": true, "--gist": true})
	fs := flag.NewFlagSet("link", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	token := fs.String("token", "", "GitHub token")
	gistID := fs.String("gist", "", "existing gist id")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*token) != "" {
		_ = app.StoreGitHubToken(*token)
	}
	if err := app.Link(*token, *gistID); err != nil {
		return err
	}
	fmt.Printf("Linked gist %s\n", app.LinkedGistID())
	return nil
}

func printHelp() {
	fmt.Println("Veil - TUI-first encrypted secret manager")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  veil")
	fmt.Println("  veil <command> [args]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  init                First-time setup wizard")
	fmt.Println("  set KEY VALUE       Add or update a secret")
	fmt.Println("  get KEY             Retrieve a secret value")
	fmt.Println("  import FILE|-       Batch import from .env file")
	fmt.Println("  export PROJECT      Export project secrets")
	fmt.Println("  run -- COMMAND      Inject secrets into subprocess")
	fmt.Println("  sync                Push/pull encrypted secrets")
	fmt.Println("  list                Show projects with secret counts")
	fmt.Println("  ls PROJECT          Show keys in a project")
	fmt.Println("  rm KEY              Delete a secret")
	fmt.Println("  link                Connect to GitHub gist")
	fmt.Println()
	fmt.Println("Project detection:")
	fmt.Println("  Defaults to current directory and known markers")
	fmt.Println("  Use -p <project> to override")
}

func reorderFlags(args []string, known map[string]bool) []string {
	flags := make([]string, 0, len(args))
	rest := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		arg := args[i]
		handled := false
		for name, takesValue := range known {
			if arg == name {
				flags = append(flags, arg)
				if takesValue && i+1 < len(args) {
					flags = append(flags, args[i+1])
					i++
				}
				handled = true
				break
			}
			prefix := name + "="
			if strings.HasPrefix(arg, prefix) {
				flags = append(flags, arg)
				handled = true
				break
			}
		}
		if !handled {
			rest = append(rest, arg)
		}
	}
	return append(flags, rest...)
}
