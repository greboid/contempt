package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/csmith/contempt"
	"github.com/csmith/contempt/pkg/materials"
	"github.com/csmith/envflag/v2"
	"golang.org/x/exp/slices"
)

var (
	templateName = flag.String("template", "Dockerfile.gotpl", "The name of the template files")
	outputName   = flag.String("output", "Dockerfile", "The name of the output files")
	filter       = flag.String("project", "", "A comma-separated list of projects to check, instead of all detected ones")
	registry     = flag.String("registry", "reg.c5h.io", "Registry to use for pulls")
	alpineMirror = flag.String("alpine-mirror", "https://dl-cdn.alpinelinux.org/alpine/", "Base URL of the Alpine mirror to use to query version and package info")
	includesDir  = flag.String("includes", "_includes", "Folder of template files to include")
	sourceLink   = flag.String("source-link", "", "Link to a browsable version of the source repo")
	tui          = flag.Bool("tui", false, "Interactively approve each new version")
	always       = flag.Bool("always", false, "Automatically approve all new versions")
	githubPR     = flag.Bool("githubpr", false, "Raise a GitHub PR for each new version")
	forgejoPR    = flag.Bool("forgejopr", false, "Raise a Forgejo PR for each new version")
	updatePRs    = flag.Bool("update-prs", false, "Update all open PRs (rebase onto latest approved versions)")
	prNumber     = flag.String("pr", "", "Specific PR number to rebase (used with -update-prs)")
)

func main() {
	envflag.Parse()

	if flag.NArg() < 1 {
		_, _ = fmt.Fprintf(os.Stderr, "Required argument missing: <input dir>\n")
		flag.Usage()
		os.Exit(2)
	}

	projectDir, err := filepath.Abs(flag.Arg(0))
	if err != nil {
		log.Fatalf("Failed to resolve project directory: %v", err)
	}

	outputDir := projectDir
	if flag.NArg() >= 2 {
		outputDir, err = filepath.Abs(flag.Arg(1))
		if err != nil {
			log.Fatalf("Failed to resolve output directory: %v", err)
		}
	}

	contempt.InitTemplates(*registry, *alpineMirror, os.DirFS(filepath.Join(projectDir, *includesDir)))

	if *updatePRs {
		cli := detectCLI()
		if *prNumber != "" {
			updateSinglePR(cli, *prNumber, projectDir, outputDir)
		} else {
			updateAllPRs(cli, projectDir, outputDir)
		}
		return
	}

	// Setup git and ensure we're on the base branch before reading approved versions
	baseBranch := setupGit()

	approved, err := materials.ReadApprovedFromDir(outputDir, *outputName)
	if err != nil {
		log.Fatalf("Failed to read approved versions: %v", err)
	}

	projects, _, err := contempt.FindProjects(projectDir, *templateName)
	if err != nil {
		log.Fatalf("Failed to find projects: %v", err)
	}

	filtered := strings.Split(*filter, ",")
	allLive := make(materials.BOM)
	projectBoms := make(map[string]materials.BOM)
	materialProjects := make(map[string][]string)

	for i := range projects {
		if *filter != "" && !slices.Contains(filtered, projects[i]) {
			continue
		}

		log.Printf("Checking project %s", projects[i])
		bom, err := contempt.CheckVersions(flag.Arg(0), filepath.Join(projects[i], *templateName))
		if err != nil {
			log.Fatalf("Failed to check project %s: %v", projects[i], err)
		}

		projectBoms[projects[i]] = bom
		maps.Copy(allLive, bom)

		for material := range bom {
			if !slices.Contains(materialProjects[material], projects[i]) {
				materialProjects[material] = append(materialProjects[material], projects[i])
			}
		}
	}

	if len(allLive) == 0 {
		fmt.Println("No materials found.")
		return
	}

	var needsApproval []string
	for material, liveVersion := range allLive {
		if materials.IsAlwaysApproved(material) {
			continue
		}
		if approved[material] != liveVersion {
			needsApproval = append(needsApproval, material)
		}
	}

	sort.Strings(needsApproval)

	if len(needsApproval) == 0 {
		fmt.Println("All materials up to date.")
		return
	}

	approveMaterials(approved, allLive, needsApproval, materialProjects, projectDir, outputDir, baseBranch)
}

var backtickRE = regexp.MustCompile("`([^`]+)`")

func parseMaterialVersion(body string) (string, string, bool) {
	matches := backtickRE.FindAllStringSubmatch(body, -1)
	if len(matches) < 2 {
		return "", "", false
	}
	return matches[0][1], matches[len(matches)-1][1], true
}

func setupGit() string {
	baseBranch, err := runOutput("git", "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		log.Fatalf("Failed to determine current branch: %v", err)
	}

	_ = runSilent("git", "config", "user.name", "contempt")
	_ = runSilent("git", "config", "user.email", "contempt[bot]@users.noreply.github.com")

	return baseBranch
}

func detectCLI() string {
	if err := runSilent("gh", "auth", "status"); err == nil {
		return "gh"
	}
	if err := runSilent("tea", "auth", "status"); err == nil {
		return "tea"
	}
	log.Fatalf("Neither gh nor tea is authenticated. For GitHub Actions, set the GH_TOKEN environment variable.")
	return ""
}

func updateAllPRs(cli string, projectDir, outputDir string) {
	projects, _, err := contempt.FindProjects(projectDir, *templateName)
	if err != nil {
		log.Fatalf("Failed to find projects: %v", err)
	}

	materialProjects := make(map[string][]string)
	for i := range projects {
		bom, err := contempt.CheckVersions(flag.Arg(0), filepath.Join(projects[i], *templateName))
		if err != nil {
			log.Fatalf("Failed to check project %s: %v", projects[i], err)
		}
		for material := range bom {
			if !slices.Contains(materialProjects[material], projects[i]) {
				materialProjects[material] = append(materialProjects[material], projects[i])
			}
		}
	}

	baseBranch := setupGit()

	prList, err := runOutput(cli, "pr", "list", "--state", "open", "--json", "number", "-q", ".[].number")
	if err != nil {
		log.Fatalf("Failed to list open PRs: %v", err)
	}

	for _, prNum := range strings.Split(prList, "\n") {
		prNum = strings.TrimSpace(prNum)
		if prNum == "" {
			continue
		}
		updatePR(cli, prNum, projectDir, outputDir, baseBranch, materialProjects)
	}
}

func updateSinglePR(cli string, prRef string, projectDir, outputDir string) {
	projects, _, err := contempt.FindProjects(projectDir, *templateName)
	if err != nil {
		log.Fatalf("Failed to find projects: %v", err)
	}

	materialProjects := make(map[string][]string)
	for i := range projects {
		bom, err := contempt.CheckVersions(flag.Arg(0), filepath.Join(projects[i], *templateName))
		if err != nil {
			log.Fatalf("Failed to check project %s: %v", projects[i], err)
		}
		for material := range bom {
			if !slices.Contains(materialProjects[material], projects[i]) {
				materialProjects[material] = append(materialProjects[material], projects[i])
			}
		}
	}

	baseBranch := setupGit()
	updatePR(cli, prRef, projectDir, outputDir, baseBranch, materialProjects)
}

func updatePR(cli string, prRef string, projectDir, outputDir, baseBranch string, materialProjects map[string][]string) {

	body, err := runOutput(cli, "pr", "view", prRef, "--json", "body", "-q", ".body")
	if err != nil {
		log.Fatalf("Failed to read PR %s: %v", prRef, err)
	}

	material, version, ok := parseMaterialVersion(body)
	if !ok {
		log.Fatalf("Could not parse material and version from PR body: %s", body)
	}

	approved, err := materials.ReadApprovedFromDir(outputDir, *outputName)
	if err != nil {
		log.Fatalf("Failed to read approved versions: %v", err)
	}

	if approved[material] == version {
		fmt.Printf("Material %s is already approved at version %s, closing PR.\n", material, version)
		_ = runSilent(cli, "pr", "close", prRef, "--comment", "Version already approved.")
		return
	}

	branchName, err := runOutput(cli, "pr", "view", prRef, "--json", "headRefName", "-q", ".headRefName")
	if err != nil {
		log.Fatalf("Failed to get branch name for PR %s: %v", prRef, err)
	}

	if err := runSilent("git", "fetch", "origin", branchName); err != nil {
		log.Fatalf("Failed to fetch branch %s: %v", branchName, err)
	}

	if err := runSilent("git", "checkout", branchName); err != nil {
		log.Fatalf("Failed to checkout branch %s: %v", branchName, err)
	}

	if err := runSilent("git", "reset", "--hard", baseBranch); err != nil {
		log.Fatalf("Failed to reset branch %s to %s: %v", branchName, baseBranch, err)
	}

	if err := regenerateProjects(materialProjects[material], projectDir, outputDir, nil); err != nil {
		log.Fatalf("Failed to regenerate projects: %v", err)
	}

	if err := runSilent("git", "add", outputDir); err != nil {
		log.Fatalf("Failed to stage changes: %v", err)
	}

	commitMsg := formatPRTitle(approved, material, version)
	if err := runSilent("git", "commit", "-m", commitMsg); err != nil {
		log.Printf("No changes to commit for %s (PR is up to date)", material)
		_ = runSilent("git", "checkout", baseBranch)
		return
	}

	if err := runSilent("git", "push", "--force-with-lease", "origin", branchName); err != nil {
		log.Fatalf("Failed to push branch %s: %v", branchName, err)
	}

	fmt.Printf("Updated PR for %s\n", material)
	_ = runSilent("git", "checkout", baseBranch)
}

func approveMaterials(approved materials.BOM, allLive materials.BOM, needsApproval []string, materialProjects map[string][]string, projectDir, outputDir, baseBranch string) {
	if *githubPR {
		approvePullRequests(approved, allLive, needsApproval, materialProjects, "gh", projectDir, outputDir, baseBranch)
		return
	}

	if *forgejoPR {
		approvePullRequests(approved, allLive, needsApproval, materialProjects, "tea", projectDir, outputDir, baseBranch)
		return
	}

	if *always {
		for _, material := range needsApproval {
			approved[material] = allLive[material]
		}
		if err := regenerateAffectedProjects(needsApproval, materialProjects, projectDir, outputDir); err != nil {
			log.Fatalf("Failed to regenerate projects: %v", err)
		}
		fmt.Println("Approved versions updated.")
		return
	}

	if *tui {
		approvedMaterials := approveTui(approved, allLive, needsApproval)
		if len(approvedMaterials) > 0 {
			if err := regenerateAffectedProjects(approvedMaterials, materialProjects, projectDir, outputDir); err != nil {
				log.Fatalf("Failed to regenerate projects: %v", err)
			}
			fmt.Println("Approved versions updated.")
		}
		return
	}

	printSummary(approved, allLive, needsApproval)
}

type existingPR struct {
	number  string
	version string
}

func listOpenPRs(cli string) map[string]existingPR {
	type prEntry struct {
		Number int
		Body   string
	}

	raw, err := runOutput(cli, "pr", "list", "--state", "open", "--json", "number,body")
	if err != nil {
		log.Printf("Failed to list open PRs: %v", err)
		return nil
	}

	var entries []prEntry
	if err := json.Unmarshal([]byte(raw), &entries); err != nil {
		log.Printf("Failed to parse PR list: %v", err)
		return nil
	}

	result := make(map[string]existingPR)
	for _, entry := range entries {
		material, version, ok := parseMaterialVersion(entry.Body)
		if !ok {
			continue
		}
		result[material] = existingPR{number: fmt.Sprintf("%d", entry.Number), version: version}
	}
	return result
}

func isPRRejectedWithNo(cli string, material string) bool {
	// Search for closed PRs related to this material
	searchQuery := fmt.Sprintf(`"%s" in:body is:closed`, material)
	raw, err := runOutput(cli, "pr", "list", "--search", searchQuery, "--state", "closed", "--json", "number,body,comments")
	if err != nil {
		log.Printf("Failed to search closed PRs for %s: %v", material, err)
		return false
	}

	var prEntries []struct {
		Number   int
		Body     string
		Comments []struct {
			Body string
		}
	}
	if err := json.Unmarshal([]byte(raw), &prEntries); err != nil {
		log.Printf("Failed to parse closed PR search for %s: %v", material, err)
		return false
	}

	// Check each closed PR for a "no" comment
	for _, pr := range prEntries {
		prMaterial, _, ok := parseMaterialVersion(pr.Body)
		if !ok || prMaterial != material {
			continue
		}

		// Check all comments for "no"
		for _, comment := range pr.Comments {
			commentBody := strings.ToLower(strings.TrimSpace(comment.Body))
			if commentBody == "no" {
				return true
			}
		}
	}

	return false
}

func approvePullRequests(approved materials.BOM, allLive materials.BOM, needsApproval []string, materialProjects map[string][]string, cli string, projectDir, outputDir, baseBranch string) {
	if err := runSilent(cli, "auth", "status"); err != nil {
		log.Fatalf("%s is not authenticated. For GitHub Actions, set the GH_TOKEN environment variable.", cli)
	}

	// Git is already configured in main()

	openPRs := listOpenPRs(cli)

	for _, material := range needsApproval {
		liveVersion := allLive[material]

		// Check if there's a closed PR with a "no" comment
		if isPRRejectedWithNo(cli, material) {
			fmt.Printf("Skipping %s: closed PR exists with 'no' comment.\n", material)
			continue
		}

		if existing, ok := openPRs[material]; ok {
			if existing.version == liveVersion {
				fmt.Printf("PR already exists for %s at version %s, skipping.\n", material, liveVersion)
				continue
			}
			fmt.Printf("Closing existing PR #%s for %s (version %s), creating new PR for version %s.\n", existing.number, material, existing.version, liveVersion)
			_ = runSilent(cli, "pr", "close", existing.number, "--comment", fmt.Sprintf("Superseded by newer version %s.", liveVersion))
		}

		branchName := fmt.Sprintf("update/%s-%s", sanitizeBranchName(material), liveVersion)

		// Delete the branch if it already exists (e.g., from a closed PR)
		if exec.Command("git", "rev-parse", "--verify", branchName).Run() == nil {
			_ = runSilent("git", "branch", "-D", branchName)
		}

		if err := runSilent("git", "checkout", "-b", branchName); err != nil {
			log.Printf("Failed to create branch for %s: %v", material, err)
			continue
		}

		overrides := materials.BOM{material: liveVersion}
		if err := regenerateProjects(materialProjects[material], projectDir, outputDir, overrides); err != nil {
			log.Printf("Failed to regenerate projects for %s: %v", material, err)
			_ = runSilent("git", "checkout", baseBranch)
			continue
		}

		if err := runSilent("git", "add", outputDir); err != nil {
			log.Printf("Failed to stage changes for %s: %v", material, err)
			_ = runSilent("git", "checkout", baseBranch)
			continue
		}

		// Check if there are any staged changes before attempting to commit
		if err := exec.Command("git", "diff", "--cached", "--quiet").Run(); err == nil {
			// No staged changes (exit code 0 means no differences)
			fmt.Printf("No changes for %s, skipping PR creation.\n", material)
			_ = runSilent("git", "checkout", baseBranch)
			continue
		}

		commitMsg := formatPRTitle(approved, material, liveVersion)
		if err := runSilent("git", "commit", "-m", commitMsg); err != nil {
			log.Printf("Failed to commit for %s: %v", material, err)
			_ = runSilent("git", "checkout", baseBranch)
			continue
		}

		if err := runSilent("git", "push", "origin", branchName); err != nil {
			log.Printf("Failed to push branch for %s: %v", material, err)
			_ = runSilent("git", "checkout", baseBranch)
			continue
		}

		prTitle := commitMsg
		prBody := formatPRBody(approved, material, liveVersion)
		if err := runSilent(cli, "pr", "create", "--head", branchName, "--base", baseBranch, "--title", prTitle, "--body", prBody); err != nil {
			log.Printf("Failed to create PR for %s: %v", material, err)
			_ = runSilent("git", "push", "origin", "--delete", branchName)
			_ = runSilent("git", "checkout", baseBranch)
			continue
		}

		fmt.Printf("Created PR for %s\n", material)
		_ = runSilent("git", "checkout", baseBranch)
	}
}

func approveTui(approved materials.BOM, allLive materials.BOM, needsApproval []string) []string {
	var approvedMaterials []string
	reader := bufio.NewReader(os.Stdin)

	for _, material := range needsApproval {
		liveVersion := allLive[material]
		if currentVersion, ok := approved[material]; ok {
			fmt.Printf("%s: %s -> %s [y/N]: ", material, currentVersion, liveVersion)
		} else {
			fmt.Printf("%s: (new) -> %s [y/N]: ", material, liveVersion)
		}

		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(strings.ToLower(response))
		if response == "y" || response == "yes" {
			approvedMaterials = append(approvedMaterials, material)
		}
	}

	return approvedMaterials
}

func printSummary(approved materials.BOM, allLive materials.BOM, needsApproval []string) {
	var newMaterials []string
	var updatedMaterials []string

	for _, material := range needsApproval {
		if _, ok := approved[material]; !ok {
			newMaterials = append(newMaterials, material)
		} else {
			updatedMaterials = append(updatedMaterials, material)
		}
	}

	if len(newMaterials) > 0 {
		fmt.Println("New materials needing approval:")
		for _, material := range newMaterials {
			fmt.Printf("  %s: %s\n", material, allLive[material])
		}
	}

	if len(updatedMaterials) > 0 {
		fmt.Println("Updated materials:")
		for _, material := range updatedMaterials {
			fmt.Printf("  %s: %s -> %s\n", material, approved[material], allLive[material])
		}
	}
}

func regenerateProjects(projects []string, projectDir, outputDir string, overrides materials.BOM) error {
	for _, project := range projects {
		outPath := filepath.Join(outputDir, project, *outputName)
		var err error
		if overrides != nil {
			_, err = contempt.GenerateWithOverrides(*sourceLink, projectDir, filepath.Join(project, *templateName), outPath, overrides)
		} else {
			_, err = contempt.Generate(*sourceLink, projectDir, filepath.Join(project, *templateName), outPath)
		}
		if err != nil {
			return fmt.Errorf("failed to regenerate project %s: %w", project, err)
		}
	}
	return nil
}

func regenerateAffectedProjects(materials []string, materialProjects map[string][]string, projectDir, outputDir string) error {
	seen := make(map[string]bool)
	for _, material := range materials {
		for _, project := range materialProjects[material] {
			if !seen[project] {
				seen[project] = true
			}
		}
	}

	var projects []string
	for project := range seen {
		projects = append(projects, project)
	}
	sort.Strings(projects)
	return regenerateProjects(projects, projectDir, outputDir, nil)
}

func sanitizeBranchName(material string) string {
	return strings.NewReplacer(":", "-", "/", "-", "_", "-").Replace(material)
}

func formatPRTitle(approved materials.BOM, material, version string) string {
	if currentVersion, ok := approved[material]; ok {
		return fmt.Sprintf("Update %s %s -> %s", material, currentVersion, version)
	}
	return fmt.Sprintf("Approve %s %s", material, version)
}

func formatPRBody(approved materials.BOM, material, version string) string {
	if currentVersion, ok := approved[material]; ok {
		return fmt.Sprintf("Update `%s` from `%s` to `%s`.", material, currentVersion, version)
	}
	return fmt.Sprintf("Approve new dependency `%s` at version `%s`.", material, version)
}

func runSilent(args ...string) error {
	var stderr bytes.Buffer
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Command %s failed: %s\n", strings.Join(args, " "), stderr.String())
	}
	return err
}

func runOutput(args ...string) (string, error) {
	cmd := exec.Command(args[0], args[1:]...)
	out, err := cmd.Output()
	return strings.TrimSpace(string(out)), err
}
