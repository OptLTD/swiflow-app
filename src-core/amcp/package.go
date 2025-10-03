package amcp

import (
	"fmt"
	"log"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"swiflow/ability"
	"swiflow/config"
)

// PackageInfo represents information about a package to be preloaded
type PackageInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Manager string `json:"manager"` // "uvx" or "npx"
	Status  string `json:"status"`  // "checking", "installing", "completed", "failed"
}

// GetInstallCommand returns the command to install this package
func (pkg *PackageInfo) GetInstallCommand() (string, []string, error) {
	switch pkg.Manager {
	case "uvx":
		return pkg.getUvxInstallCmd()
	case "npx":
		return pkg.getNpxInstallCmd()
	default:
		return "", nil, fmt.Errorf("unsupported package manager: %s", pkg.Manager)
	}
}

// getUvxInstallCmd creates the uvx installation command
func (pkg *PackageInfo) getUvxInstallCmd() (string, []string, error) {
	// Use config.GetMcpEnv to get the correct uvx path
	command := "uv"
	if uvxPath, err := config.GetMcpEnv("uv"); err == nil {
		command = uvxPath
	}

	args := []string{"tool", "install"}
	if pkg.Version != "" {
		args = append(args, fmt.Sprintf("%s@%s", pkg.Name, pkg.Version))
	} else {
		args = append(args, pkg.Name)
	}

	return command, args, nil
}

// getNpxInstallCmd creates the npm installation command for npx packages
func (pkg *PackageInfo) getNpxInstallCmd() (string, []string, error) {
	// Use config.GetMcpEnv to get the correct npm path
	npmPath, err := config.GetMcpEnv("npm")
	if err != nil {
		// Fallback to direct npm command if config method fails
		npmPath = "npm"
	}

	args := []string{"install", "-g"}
	if pkg.Version != "" {
		args = append(args, fmt.Sprintf("%s@%s", pkg.Name, pkg.Version))
	} else {
		args = append(args, pkg.Name)
	}

	return npmPath, args, nil
}

// GetSessionName returns a unique session name for this package
func (pkg *PackageInfo) GetSessionName() string {
	return fmt.Sprintf("install-%s-%s", pkg.Manager, pkg.Name)
}

// GetProgressKey returns a unique key for tracking this package's installation progress
func (pkg *PackageInfo) GetProgressKey() string {
	return fmt.Sprintf("%s:%s", pkg.Manager, pkg.Name)
}

// IsInstalled checks if this package is already installed locally
func (pkg *PackageInfo) IsInstalled() (bool, error) {
	switch pkg.Manager {
	case "uv":
		return true, nil
	case "uvx":
		return pkg.isUvxPackageInstalled()
	case "npx":
		return pkg.isNpxPackageInstalled()
	default:
		return false, fmt.Errorf("unsupported package manager: %s", pkg.Manager)
	}
}

// isUvxPackageInstalled checks if this uvx package is installed
func (pkg *PackageInfo) isUvxPackageInstalled() (bool, error) {
	// Get the correct uvx path
	uvxPath, err := config.GetMcpEnv("uv")
	if err != nil {
		return false, fmt.Errorf("uv not available: %v", err)
	}

	command := exec.Command(uvxPath, "tool", "list")
	if output, err := command.CombinedOutput(); err != nil {
		return false, nil
	} else {
		return strings.Contains(string(output), pkg.Name), nil
	}
}

// isNpxPackageInstalled checks if this npx package is installed
func (pkg *PackageInfo) isNpxPackageInstalled() (bool, error) {
	npmPath, err := config.GetMcpEnv("npm")
	if err != nil {
		npmPath = "npm"
	}

	packageSpec := pkg.Name
	if pkg.Version != "" {
		packageSpec = fmt.Sprintf("%s@%s", pkg.Name, pkg.Version)
	}
	command := exec.Command(npmPath, "cache", "ls", packageSpec)
	if output, err := command.CombinedOutput(); err != nil {
		log.Printf("[MCP] Missing Cache %s: %v", pkg.Name, err)
		return false, fmt.Errorf("failed to check cache: %v", err)
	} else if string(output) == "" || len(string(output)) < len(pkg.Name) {
		log.Printf("[MCP] Missing Cache %s: %s", pkg.Name, string(output))
		return false, fmt.Errorf("failed to check cache: %v", err)
	}
	return true, nil
}

// Install starts async installation of this package
func (pkg *PackageInfo) Install() error {
	// Store package in tracking map
	pkg.Status = "checking"
	pkg.notifyProgress()

	// Check if already installed
	if installed, err := pkg.IsInstalled(); err != nil {
		pkg.Status = "failed"
		pkg.notifyProgress()
		return err
	} else if installed {
		pkg.Status = "completed"
		pkg.notifyProgress()
		return nil
	}

	// Start async installation
	pkg.Status = "installing"
	pkg.notifyProgress()

	// Get cmd and arguments
	cmd, args, err := pkg.GetInstallCommand()
	if err != nil {
		pkg.Status = "failed"
		pkg.notifyProgress()
		return err
	}

	// Create async command ability
	cmdAbility := &ability.DevAsyncCmdAbility{
		Name: pkg.GetSessionName(), Home: ".",
	}

	// Build full command string
	fullCmd := cmd + " " + strings.Join(args, " ")
	if err := cmdAbility.Start(fullCmd); err != nil {
		pkg.Status = "failed"
		pkg.notifyProgress()
		log.Printf("Failed to start installation for %s: %v", pkg.Name, err)
		return err
	}

	// Start goroutine to monitor installation progress
	go pkg.monitor()
	return nil
}

// monitor monitors the progress of this package's async installation
func (pkg *PackageInfo) monitor() {
	cmdAbility := &ability.DevAsyncCmdAbility{
		Name: pkg.GetSessionName(), Home: ".",
	}

	// Check status every second
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	timeout := time.After(5 * time.Minute)
	for {
		select {
		case <-ticker.C:
			// Query command status
			output, err := cmdAbility.Query()
			if err != nil {
				// Command might have finished or failed
				installed, checkErr := pkg.IsInstalled()
				if checkErr == nil && installed {
					pkg.Status = "completed"
					pkg.notifyProgress()
					log.Printf("Successfully installed package: %s via %s", pkg.Name, pkg.Manager)
					return
				} else {
					pkg.Status = "failed"
					pkg.notifyProgress()
					log.Printf("Installation failed for %s: %v", pkg.Name, err)
					return
				}
			}

			// Check if output indicates completion
			if strings.Contains(output, "successfully") || strings.Contains(output, "success") {
				// Verify installation
				installed, checkErr := pkg.IsInstalled()
				if checkErr == nil && installed {
					pkg.Status = "completed"
					pkg.notifyProgress()
					log.Printf("Successfully installed package: %s via %s", pkg.Name, pkg.Manager)
					return
				}
			}

		case <-timeout:
			// Installation timed out
			pkg.Status = "failed"
			pkg.notifyProgress()
			log.Printf("Installation timed out for %s", pkg.Name)
			_ = cmdAbility.Abort() // Try to abort the command
			return
		}
	}
}

// notifyProgress notifies all registered callbacks about progress updates
func (pkg *PackageInfo) notifyProgress() {

}

// ParseCommand parses uvx/npx commands to extract package information
func (pkg *PackageInfo) ParseCommand(cmd string, args []string) error {
	// Determine the package manager from command
	if strings.Contains(cmd, "uvx") {
		return pkg.parseUvxCommand(args)
	} else if strings.Contains(cmd, "npx") {
		return pkg.parseNpxCommand(args)
	} else if strings.HasPrefix(cmd, "uv") {
		pkg.Manager, pkg.Name = "uv", "local"
		return nil
	}
	return fmt.Errorf("unsupported command: %s", cmd)
}

// parseUvxCommand parses uvx command arguments to extract package information
// uvx usage: uvx [OPTIONS] [COMMAND]
// For MCP servers, typically only one package is executed per command
func (pkg *PackageInfo) parseUvxCommand(args []string) error {
	if len(args) == 0 {
		return nil
	}

	// Look for --from option first (explicit package specification)
	for i := 0; i < len(args)-1; i++ {
		if args[i] == "--from" {
			err := pkg.parsePackageSpec(args[i+1], "uvx")
			if err != nil {
				return err
			}
			if pkg.Name != "" {
				return nil
			}
		}
	}

	// If no --from found, first non-flag argument is the package/command
	for _, arg := range args {
		if !strings.HasPrefix(arg, "-") {
			err := pkg.parsePackageSpec(arg, "uvx")
			if err != nil {
				return err
			}
			if pkg.Name != "" {
				return nil
			}
		}
	}
	return nil
}

// parseNpxCommand parses npx command arguments to extract package information
// npx usage: npx [options] <command>[@version] [command-arg]...
// For MCP servers, typically only one package is executed per command
func (pkg *PackageInfo) parseNpxCommand(args []string) error {
	if len(args) == 0 {
		return nil
	}

	// Look for --package option first (explicit package specification)
	for i := 0; i < len(args); i++ {
		arg := args[i]

		// Handle --package=package-name format
		if strings.HasPrefix(arg, "--package=") {
			packageSpec := strings.TrimPrefix(arg, "--package=")
			err := pkg.parsePackageSpec(packageSpec, "npx")
			if err == nil || pkg.Name != "" {
				return nil
			} else if err != nil {
				return err
			}
		}

		// Handle --package package-name format
		if arg == "--package" && i+1 < len(args) {
			err := pkg.parsePackageSpec(args[i+1], "npx")
			if err == nil || pkg.Name != "" {
				return nil
			} else if err != nil {
				return err
			}
		}
	}

	// If no --package found, first non-flag argument is the package/command
	for _, arg := range args {
		// Skip common npx flags
		if arg == "-y" || arg == "--yes" || arg == "-q" || arg == "--quiet" ||
			arg == "-p" || arg == "--parseable" || arg == "-c" || arg == "--call" ||
			strings.HasPrefix(arg, "-") {
			continue
		}

		// First non-flag argument is the package/command
		err := pkg.parsePackageSpec(arg, "npx")
		if err == nil || pkg.Name != "" {
			return nil
		} else if err != nil {
			return err
		}
	}

	return nil
}

// parsePackageSpec parses a package specification (name@version or just name)
func (pkg *PackageInfo) parsePackageSpec(spec, manager string) error {
	// Special handling for scoped packages (starting with @)
	if strings.HasPrefix(spec, "@") {
		// For scoped packages like @larksuiteoapi/lark-mcp
		parts := strings.Split(spec, "@")
		if len(parts) > 2 { // Has version: @scope/name@version
			pkg.Name = "@" + parts[1]
			pkg.Version = parts[2]
			pkg.Manager = manager
		} else { // No version: @scope/name
			pkg.Name = spec
			pkg.Manager = manager
		}
		return nil
	}

	// Regular expression to match package@version format for non-scoped packages
	re := regexp.MustCompile(`^([^@]+)(?:@(.+))?$`)
	matches := re.FindStringSubmatch(spec)

	if len(matches) < 2 {
		return fmt.Errorf("unformat")
	}

	pkg.Name = matches[1]
	pkg.Manager = manager
	if len(matches) > 2 && matches[2] != "" {
		pkg.Version = matches[2]
	}

	return nil
}
