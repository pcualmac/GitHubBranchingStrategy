package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// Constants for branch names to avoid typos
const (
	DevelopmentBranch = "Development"
	NightlyBranch     = "Nightly"
	ReleaseBranch     = "Release"
	MasterBranch      = "Master"
)

// GitCommandExecutor defines an interface for executing Git commands
type GitCommandExecutor interface {
	RunGitCommand(args ...string) error
	GitStatusPorcelain() (bool, error)
}

// RealGitExecutor implements GitCommandExecutor for actual Git operations
type RealGitExecutor struct{}

func (r *RealGitExecutor) RunGitCommand(args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	fmt.Printf("Executing: git %s\n", strings.Join(args, " "))
	return cmd.Run()
}

func (r *RealGitExecutor) GitStatusPorcelain() (bool, error) {
	cmd := exec.Command("git", "status", "--porcelain")
	output, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("failed to get git status: %w", err)
	}
	return len(strings.TrimSpace(string(output))) == 0, nil
}

// WorkflowManager handles the Git workflow operations
type WorkflowManager struct {
	executor GitCommandExecutor
}

func NewWorkflowManager(executor GitCommandExecutor) *WorkflowManager {
	return &WorkflowManager{executor: executor}
}

// checkoutBranch performs a git checkout operation
func (wm *WorkflowManager) checkoutBranch(branch string) error {
	fmt.Printf("Switching to branch: %s\n", branch)
	return wm.executor.RunGitCommand("checkout", branch)
}

// pullOrigin pulls the latest changes from the remote origin for the current branch
func (wm *WorkflowManager) pullOrigin(branch string) error {
	fmt.Printf("Pulling latest changes for %s from origin/%s...\n", branch, branch)
	return wm.executor.RunGitCommand("pull", "origin", branch)
}

// fetchOrigin fetches all remote branches and tags
func (wm *WorkflowManager) fetchOrigin() error {
	fmt.Println("Fetching latest changes from all remotes...")
	return wm.executor.RunGitCommand("fetch", "origin")
}

// mergeBranch merges the sourceBranch into the current branch
func (wm *WorkflowManager) mergeBranch(sourceBranch string, noFF bool, message string) error {
	args := []string{"merge", sourceBranch}
	if noFF {
		args = append(args, "--no-ff")
	}
	if message != "" {
		args = append(args, "-m", message)
	}
	fmt.Printf("Merging %s into current branch...\n", sourceBranch)
	return wm.executor.RunGitCommand(args...)
}

// pushBranch pushes the current branch to the remote origin
func (wm *WorkflowManager) pushBranch(branch string, force bool, tags bool) error {
	args := []string{"push", "origin"}
	if force {
		args = append(args, "--force")
	}
	if tags {
		args = append(args, "--tags")
	} else {
		args = append(args, branch)
	}
	fmt.Printf("Pushing %s to origin...\n", branch)
	return wm.executor.RunGitCommand(args...)
}

// updateFeatureBranch updates a feature branch by merging origin/development into it
func (wm *WorkflowManager) updateFeatureBranch(featureBranch string) error {
	fmt.Println("--- U_B: Updating Feature Branch ---")
	if err := wm.checkoutBranch(featureBranch); err != nil {
		return fmt.Errorf("failed to checkout feature branch %s: %w", featureBranch, err)
	}
	if err := wm.fetchOrigin(); err != nil {
		return fmt.Errorf("failed to fetch origin: %w", err)
	}
	if err := wm.mergeBranch("origin/development", false, ""); err != nil {
		return fmt.Errorf("merge conflict detected or merge failed. Please resolve manually and re-run if needed: %w", err)
	}
	if err := wm.pushBranch(featureBranch, false, false); err != nil {
		return fmt.Errorf("failed to push updated feature branch: %w", err)
	}
	fmt.Println("--- U_B: Feature Branch Updated Successfully ---")
	return nil
}

// updateDevelopment pulls latest changes into the Development branch
func (wm *WorkflowManager) updateDevelopment() error {
	fmt.Println("--- U_D: Updating Development Branch ---")
	if err := wm.checkoutBranch(DevelopmentBranch); err != nil {
		return fmt.Errorf("failed to checkout Development branch: %w", err)
	}
	if err := wm.pullOrigin(DevelopmentBranch); err != nil {
		printMergeConflictInstructions()
		return fmt.Errorf("failed to pull origin/Development: %w", err)
	}
	fmt.Println("--- U_D: Development Branch Updated Successfully ---")
	return nil
}

func printMergeConflictInstructions() {
	fmt.Println("\nMerge conflict resolution instructions:")
	fmt.Println("1. Run `git status` to view conflicting files")
	fmt.Println("2. Manually edit files (remove conflict markers)")
	fmt.Println("3. Run `git add .` to mark conflicts as resolved")
	fmt.Println("4. Run `git commit -m \"fix: Resolve merge conflicts\"`")
	fmt.Println("5. Run `git push origin Development`")
}

// promoteDevToNightlyBasic promotes Development to Nightly (basic version)
func (wm *WorkflowManager) promoteDevToNightlyBasic() error {
	fmt.Println("--- F_M_D: Promoting Development to Nightly (Basic) ---")

	if err := wm.checkoutBranch(DevelopmentBranch); err != nil {
		return fmt.Errorf("failed to checkout Development branch: %w", err)
	}
	if err := wm.pullOrigin(DevelopmentBranch); err != nil {
		return fmt.Errorf("failed to pull Development before promoting: %w", err)
	}

	if err := wm.checkoutBranch(NightlyBranch); err != nil {
		return fmt.Errorf("failed to checkout Nightly branch: %w", err)
	}
	if err := wm.pullOrigin(NightlyBranch); err != nil {
		return fmt.Errorf("failed to pull Nightly before merge: %w", err)
	}
	if err := wm.fetchOrigin(); err != nil {
		return fmt.Errorf("failed to fetch origin: %w", err)
	}
	if err := wm.mergeBranch("origin/Development", false, ""); err != nil {
		return fmt.Errorf("merge conflict detected during Development to Nightly. Please resolve manually: %w", err)
	}
	if err := wm.pushBranch(NightlyBranch, false, false); err != nil {
		return fmt.Errorf("failed to push updated Nightly branch: %w", err)
	}
	fmt.Println("--- F_M_D: Development to Nightly (Basic) Completed Successfully ---")
	return nil
}

// createFeatureBranch creates a new feature branch from Development
func (wm *WorkflowManager) createFeatureBranch(featureName string) error {
	fmt.Println("--- F_D: Creating Feature Branch ---")
	if err := wm.checkoutBranch(DevelopmentBranch); err != nil {
		return fmt.Errorf("failed to checkout Development: %w", err)
	}
	newBranch := fmt.Sprintf("feature/%s", featureName)
	fmt.Printf("Creating new branch: %s from Development...\n", newBranch)
	if err := wm.executor.RunGitCommand("checkout", "-b", newBranch); err != nil {
		return fmt.Errorf("failed to create new feature branch: %w", err)
	}
	fmt.Println("--- F_D: Feature Branch Created Successfully ---")
	return nil
}

// consumeFeature merges a feature branch into Development
func (wm *WorkflowManager) consumeFeature(featureBranch string) error {
	fmt.Println("--- C_F: Consuming Feature Branch into Development ---")

	if err := wm.checkoutBranch(featureBranch); err != nil {
		return fmt.Errorf("failed to checkout feature branch %s: %w", featureBranch, err)
	}
	if err := wm.pullOrigin(featureBranch); err != nil {
		return fmt.Errorf("failed to pull feature branch %s: %w", featureBranch, err)
	}

	if err := wm.checkoutBranch(DevelopmentBranch); err != nil {
		return fmt.Errorf("failed to checkout Development branch: %w", err)
	}
	if err := wm.pullOrigin(DevelopmentBranch); err != nil {
		return fmt.Errorf("failed to pull Development before merging feature: %w", err)
	}

	if err := wm.checkoutBranch(featureBranch); err != nil {
		return fmt.Errorf("failed to checkout feature branch %s: %w", featureBranch, err)
	}
	if err := wm.mergeBranch(DevelopmentBranch, false, ""); err != nil {
		return fmt.Errorf("merge conflict detected when merging Development into feature. Please resolve manually: %w", err)
	}

	if err := wm.checkoutBranch(DevelopmentBranch); err != nil {
		return fmt.Errorf("failed to checkout Development branch for final merge: %w", err)
	}
	if err := wm.mergeBranch(featureBranch, false, ""); err != nil {
		return fmt.Errorf("merge conflict detected when merging feature into Development. Please resolve manually: %w", err)
	}

	if err := wm.pushBranch(DevelopmentBranch, false, false); err != nil {
		return fmt.Errorf("failed to push updated Development branch: %w", err)
	}

	fmt.Println("--- C_F: Feature Consumed into Development Successfully ---")
	return nil
}

// promoteDevelopmentToNightly promotes the Development branch to Nightly with validation
func (wm *WorkflowManager) promoteDevelopmentToNightly() error {
	fmt.Println("--- Promote: Promoting Development to Nightly with validation ---")

	if err := wm.checkoutBranch(DevelopmentBranch); err != nil {
		return fmt.Errorf("failed to checkout Development: %w", err)
	}
	clean, err := wm.executor.GitStatusPorcelain()
	if err != nil {
		return fmt.Errorf("failed to check git status: %w", err)
	}
	if !clean {
		return fmt.Errorf("error: Development has uncommitted changes")
	}

	if err := wm.fetchOrigin(); err != nil {
		return fmt.Errorf("failed to fetch origin: %w", err)
	}
	if err := wm.pullOrigin(DevelopmentBranch); err != nil {
		return fmt.Errorf("failed to pull origin/Development: %w", err)
	}

	if err := wm.checkoutBranch(NightlyBranch); err != nil {
		return fmt.Errorf("failed to checkout Nightly: %w", err)
	}
	if err := wm.mergeBranch("origin/Development", true, ""); err != nil {
		printMergeConflictInstructions()
		return fmt.Errorf("merge conflict during Development to Nightly promotion: %w", err)
	}

	if err := wm.pushBranch(NightlyBranch, false, false); err != nil {
		return fmt.Errorf("failed to push Nightly: %w", err)
	}

	fmt.Println("--- Promote: Development Promoted to Nightly Successfully ---")
	return nil
}

// developmentToRelease promotes the Development branch to Release
func (wm *WorkflowManager) developmentToRelease(versionTag string) error {
	fmt.Println("--- D_R: Promoting Development to Release ---")

	if err := wm.checkoutBranch(DevelopmentBranch); err != nil {
		return fmt.Errorf("failed to checkout Development: %w", err)
	}
	if err := wm.pullOrigin(DevelopmentBranch); err != nil {
		return fmt.Errorf("failed to pull Development: %w", err)
	}

	// Check if Release branch exists
	cmd := exec.Command("git", "show-ref", "--verify", "refs/heads/"+ReleaseBranch)
	if err := cmd.Run(); err != nil { // Branch does not exist, create it
		if err := wm.executor.RunGitCommand("checkout", "-b", ReleaseBranch); err != nil {
			return fmt.Errorf("failed to create Release branch: %w", err)
		}
	} else { // Branch exists, checkout and pull
		if err := wm.checkoutBranch(ReleaseBranch); err != nil {
			return fmt.Errorf("failed to checkout Release branch: %w", err)
		}
		if err := wm.pullOrigin(ReleaseBranch); err != nil {
			return fmt.Errorf("failed to pull Release branch: %w", err)
		}
	}

	mergeMsg := fmt.Sprintf("chore: Promote Development to Release [%s]", time.Now().Format("2006-01-02"))
	if err := wm.mergeBranch(DevelopmentBranch, true, mergeMsg); err != nil {
		printMergeConflictInstructions()
		return fmt.Errorf("merge conflict during Development to Release promotion: %w", err)
	}

	if err := wm.pushBranch(ReleaseBranch, false, false); err != nil {
		return fmt.Errorf("failed to push Release branch: %w", err)
	}

	if versionTag != "" {
		fmt.Printf("Tagging release as %s...\n", versionTag)
		if err := wm.executor.RunGitCommand("tag", "-a", versionTag, "-m", fmt.Sprintf("Release candidate %s", versionTag)); err != nil {
			return fmt.Errorf("failed to create tag: %w", err)
		}
		if err := wm.pushBranch("", false, true); err != nil {
			return fmt.Errorf("failed to push tag: %w", err)
		}
	}

	fmt.Println("--- D_R: Development Promoted to Release Successfully ---")
	return nil
}

// syncDevWithMaster backs up Development and resets Development to Master
func (wm *WorkflowManager) syncDevWithMaster() error {
	fmt.Println("--- M: Backing up Development and resetting to Master ---")

	if err := wm.checkoutBranch(DevelopmentBranch); err != nil {
		return fmt.Errorf("failed to checkout Development for backup: %w", err)
	}
	backupTag := fmt.Sprintf("backup/development-%s", time.Now().Format("20060102"))
	if err := wm.executor.RunGitCommand("tag", backupTag); err != nil {
		return fmt.Errorf("failed to create development backup tag: %w", err)
	}
	if err := wm.pushBranch("", false, true); err != nil {
		return fmt.Errorf("failed to push backup tag: %w", err)
	}

	if err := wm.checkoutBranch(DevelopmentBranch); err != nil {
		return fmt.Errorf("failed to checkout Development for reset: %w", err)
	}
	if err := wm.fetchOrigin(); err != nil {
		return fmt.Errorf("failed to fetch origin before resetting Development: %w", err)
	}
	fmt.Println("Hard resetting Development to origin/master... (DANGER: This discards local Development changes)")
	if err := wm.executor.RunGitCommand("reset", "--hard", "origin/master"); err != nil {
		return fmt.Errorf("failed to hard reset Development to origin/master: %w", err)
	}
	fmt.Println("Force pushing Development to remote... (DANGER: This overwrites remote Development)")
	if err := wm.pushBranch(DevelopmentBranch, true, false); err != nil {
		return fmt.Errorf("failed to force push Development: %w", err)
	}
	fmt.Println("--- M: Development Backup and Sync with Master Completed ---")
	return nil
}

// createHotfix creates a new hotfix branch from Master
func (wm *WorkflowManager) createHotfix(hotfixName string) error {
	fmt.Println("--- C_H: Creating Hotfix Branch ---")

	if err := wm.checkoutBranch(MasterBranch); err != nil {
		return fmt.Errorf("failed to checkout Master branch: %w", err)
	}
	if err := wm.pullOrigin(MasterBranch); err != nil {
		return fmt.Errorf("failed to pull Master branch: %w", err)
	}

	newHotfixBranch := fmt.Sprintf("hotfix/%s", hotfixName)
	fmt.Printf("Creating new hotfix branch: %s from Master...\n", newHotfixBranch)
	if err := wm.executor.RunGitCommand("checkout", "-b", newHotfixBranch); err != nil {
		return fmt.Errorf("failed to create new hotfix branch: %w", err)
	}

	if err := wm.pushBranch(newHotfixBranch, false, false); err != nil {
		return fmt.Errorf("failed to push hotfix branch to remote: %w", err)
	}
	fmt.Println("--- C_H: Hotfix Branch Created and Pushed Successfully ---")
	return nil
}

// updateMaster merges a hotfix into Master and forward-ports to Development
func (wm *WorkflowManager) updateMaster(hotfixBranch string) error {
	fmt.Println("--- U_M: Updating Master with Hotfix and Forward-Porting ---")

	if err := wm.checkoutBranch(MasterBranch); err != nil {
		return fmt.Errorf("failed to checkout Master branch: %w", err)
	}
	if err := wm.mergeBranch(hotfixBranch, true, ""); err != nil {
		return fmt.Errorf("merge conflict detected during hotfix merge to Master. Please resolve manually: %w", err)
	}

	if err := wm.pushBranch(MasterBranch, false, false); err != nil {
		return fmt.Errorf("failed to push Master after hotfix merge: %w", err)
	}

	if err := wm.checkoutBranch(DevelopmentBranch); err != nil {
		return fmt.Errorf("failed to checkout Development for forward-port: %w", err)
	}
	forwardPortMsg := fmt.Sprintf("chore: Forward-port %s to Development", hotfixBranch)
	if err := wm.mergeBranch(hotfixBranch, true, forwardPortMsg); err != nil {
		return fmt.Errorf("merge conflict detected during hotfix forward-port to Development. Please resolve manually: %w", err)
	}

	if err := wm.pushBranch(DevelopmentBranch, false, false); err != nil {
		return fmt.Errorf("failed to push Development after forward-port: %w", err)
	}

	fmt.Printf("Cleaning up hotfix branch: %s...\n", hotfixBranch)
	if err := wm.executor.RunGitCommand("branch", "-d", hotfixBranch); err != nil {
		fmt.Printf("Warning: Failed to delete local hotfix branch %s: %v\n", hotfixBranch, err)
	}
	if err := wm.executor.RunGitCommand("push", "origin", "--delete", hotfixBranch); err != nil {
		fmt.Printf("Warning: Failed to delete remote hotfix branch %s: %v\n", hotfixBranch, err)
	}
	fmt.Println("--- U_M: Master Updated and Hotfix Forward-Ported Successfully ---")
	return nil
}

func printUsage() {
	fmt.Println("Git Workflow Tool - Manage your Git branching workflow")
	fmt.Println("Usage: go run main.go <command> [arguments]")
	fmt.Println("\nCommands:")
	fmt.Println("  ub <feature-branch>     Update feature branch with latest development changes")
	fmt.Println("  ud                      Update Development branch with latest changes")
	fmt.Println("  fmd                     Promote Development to Nightly (basic version)")
	fmt.Println("  cfb <feature-name>      Create new feature branch from Development")
	fmt.Println("  cf <feature-branch>     Consume feature branch into Development")
	fmt.Println("  promote                 Promote Development to Nightly with validation")
	fmt.Println("  dr [version-tag]        Promote Development to Release (optional version tag)")
	fmt.Println("  m                       Sync Development with Master (backup and reset)")
	fmt.Println("  ch <hotfix-name>        Create hotfix branch from Master")
	fmt.Println("  um <hotfix-branch>      Update Master with hotfix and forward-port")
	fmt.Println("  help                    Display this help message")
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	executor := &RealGitExecutor{}
	wm := NewWorkflowManager(executor)

	command := os.Args[1]
	var err error

	switch command {
	case "ub":
		if len(os.Args) < 3 {
			fmt.Println("Error: Feature branch name required")
			printUsage()
			os.Exit(1)
		}
		err = wm.updateFeatureBranch(os.Args[2])
	case "ud":
		err = wm.updateDevelopment()
	case "fmd":
		err = wm.promoteDevToNightlyBasic()
	case "cfb":
		if len(os.Args) < 3 {
			fmt.Println("Error: Feature name required")
			printUsage()
			os.Exit(1)
		}
		err = wm.createFeatureBranch(os.Args[2])
	case "cf":
		if len(os.Args) < 3 {
			fmt.Println("Error: Feature branch name required")
			printUsage()
			os.Exit(1)
		}
		err = wm.consumeFeature(os.Args[2])
	case "promote":
		err = wm.promoteDevelopmentToNightly()
	case "dr":
		versionTag := ""
		if len(os.Args) >= 3 {
			versionTag = os.Args[2]
		}
		err = wm.developmentToRelease(versionTag)
	case "m":
		err = wm.syncDevWithMaster()
	case "ch":
		if len(os.Args) < 3 {
			fmt.Println("Error: Hotfix name required")
			printUsage()
			os.Exit(1)
		}
		err = wm.createHotfix(os.Args[2])
	case "um":
		if len(os.Args) < 3 {
			fmt.Println("Error: Hotfix branch name required")
			printUsage()
			os.Exit(1)
		}
		err = wm.updateMaster(os.Args[2])
	case "help":
		printUsage()
		os.Exit(0)
	default:
		fmt.Printf("Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}

	if err != nil {
		fmt.Printf("\nError: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("\nOperation completed successfully!")
}
