package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

var defaultMainBranchName string = "main"
var defaultGroupByForLogDate string = "month"

const REPOSITORIES_DIRECTORY = ".repositories"

type UserContribution struct {
	Email                string
	CommitCount          int
	ContributionTimeline map[string]int // Year-Week: count
	LinesAdded           int
	LinesRemoved         int
	LinesEdited          int
	FileFilter           string
}

type BranchReport struct {
	BranchName    string
	Contributions map[string]*UserContribution
}

type ReportData struct {
	RepoName      string
	FileFilter    string
	BranchReports map[string]*BranchReport
}

type customLogWriter struct {
}

func (writer customLogWriter) Write(bytes []byte) (int, error) {
	return fmt.Print(time.Now().UTC().Format("2006-01-02 15:04:05") + " " + string(bytes))
}

func main() {
	log.SetFlags(0)
	log.SetOutput(new(customLogWriter))

	if err := isGitInstalled(); err != nil {
		log.Fatalf("Error: %s", err)
	}

	repoPath := flag.String("repository", "", "Path to the git repository (directory or URL)")
	fileFilter := flag.String("filter", "", "Filter for file types (e.g., go, py, etc.). Optional")
	optoinMainBranch := flag.String("mainbranch", defaultMainBranchName, "Name of the 'main' branch for merge-base")
	optionGroupByForLogDate := flag.String("groupby", defaultGroupByForLogDate, "Group git log date by 'week' or 'month'")
	flag.Parse()

	if *repoPath == "" {
		log.Fatal("Please provide the path to the git repository using `--repository`")
	}

	u, err := url.Parse(*repoPath)
	if err == nil && (u.Scheme == "http" || u.Scheme == "https" || u.Scheme == "git" || u.Scheme == "ssh") {
		log.Println("URL found. Cloning repository: ", *repoPath)
		newRepoPath, err := cloneRepository(*repoPath, REPOSITORIES_DIRECTORY)
		if err != nil {
			log.Fatalf("Error cloning repository: %v", err)
		}

		*repoPath = newRepoPath

		if err := checkoutRemoteBranches(*repoPath); err != nil {
			log.Fatalf("Error checking out all branched: %s", err)
		}
	}

	if _, err := os.Stat(*repoPath); os.IsNotExist(err) {
		log.Fatalf("Repository path does not exist: %s", *repoPath)
	}

	repoName := filepath.Base(*repoPath)
	log.Printf("Analyzing repository: %s", repoName)

	if *optoinMainBranch != "" {
		defaultMainBranchName = *optoinMainBranch
		log.Printf("Name of the main branch has been set to: %s", defaultMainBranchName)
	}

	if *optionGroupByForLogDate != "" {
		if (*optionGroupByForLogDate != "week") && (*optionGroupByForLogDate != "month") {
			log.Fatalf("Given option for parameter 'groupby' is not supported. Excepted 'week' or 'month'. Given: %s", *optionGroupByForLogDate)
		}

		defaultGroupByForLogDate = *optionGroupByForLogDate
		log.Printf("Default group by option has been set to: %s", defaultGroupByForLogDate)
	}

	branchReports, err := analyzeGitHistoryByBranch(*repoPath, *fileFilter)
	if err != nil {
		log.Fatalf("Error analyzing git history: %v", err)
	}

	htmlReport, err := generateHTMLReportByBranch(branchReports, repoName, *fileFilter)
	if err != nil {
		log.Fatalf("Error generating HTML report: %v", err)
	}

	filename := fmt.Sprintf("report_%s_%s.html", repoName, time.Now().Format("2006-01-02_150405"))
	err = os.WriteFile(filename, []byte(htmlReport), 0644)
	if err != nil {
		log.Fatalf("Error writing HTML report to file: %v", err)
	}

	log.Printf("HTML report generated: %s\n", filename)
}

// isGitInstalled checks if Git is installed and accessible in the system's PATH.
//
// It uses exec.LookPath to search for the "git" executable.
//
// Returns:
//   - nil if Git is found.
//   - An error if Git is not installed or not found in the PATH.
func isGitInstalled() error {
	_, err := exec.LookPath("git")
	if err != nil {
		return errors.New("git is not installed or not found in PATH")
	}
	return nil
}

// cloneRepository clones a Git repository from the given URL to the specified destination directory.
//
// It first checks if the destination directory exists. If not, it creates it.
// Then, it derives the repository name from the URL and constructs the local repository path.
// If the local repository does not exist, it executes the "git clone" command.
// If the local repository already exists, it skips the cloning process.
//
// Parameters:
//   - repoURL: The URL of the Git repository to clone.
//   - destDir: The destination directory where the repository should be cloned.
//
// Returns:
//   - The local path to the cloned repository.
//   - An error, if any, occurred during the cloning process.
func cloneRepository(repoURL, destDir string) (string, error) {

	if _, err := os.Stat(destDir); os.IsNotExist(err) {
		if err := os.MkdirAll(destDir, 0755); err != nil {
			return "", fmt.Errorf("failed to create directory %s: %w", destDir, err)
		}
	}

	repoName := filepath.Base(repoURL)
	localRepoPath := filepath.Join(destDir, repoName)

	if _, err := os.Stat(localRepoPath); os.IsNotExist(err) {
		cmd := exec.Command("git", "clone", repoURL, localRepoPath)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return "", fmt.Errorf("failed to clone repository: %s, output: %s", err, output)
		}
		log.Printf("Repository cloned to: %s", localRepoPath)
	} else {
		log.Printf("Repository already exists at: %s", localRepoPath)
	}

	return localRepoPath, nil
}

// checkoutRemoteBranches checks out all remote branches of a Git repository located at repoPath.
//
// It executes the following steps:
//  1. Retrieves the list of remote branches using `git branch -r`.
//  2. Iterates through each remote branch, skipping empty branches and symbolic HEAD references.
//  3. If a branch starts with "origin/", it extracts the branch name and attempts to check it out locally
//     using `git checkout -b <local_branch_name> <remote_branch_name>`.
//  4. If the checkout fails and the error message does not indicate that the branch already exists,
//     it returns an error.
//
// Parameters:
//   - repoPath: The path to the Git repository.
//
// Returns:
//   - nil if all remote branches are successfully checked out or already exist.
//   - An error if any other error occurs during the process.
func checkoutRemoteBranches(repoPath string) error {

	log.Printf("Checking remote branches")

	cmd := exec.Command("git", "branch", "-r")
	cmd.Dir = repoPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to get remote branches: %w, output: %s", err, output)
	}

	branches := strings.Split(string(output), "\n")

	for _, branch := range branches {
		branch = strings.TrimSpace(branch)
		if branch == "" || strings.Contains(branch, "HEAD ->") { // Skip HEAD -> branches
			continue
		}

		if strings.HasPrefix(branch, "origin/") {
			branchName := strings.TrimPrefix(branch, "origin/")
			branchName = strings.TrimSpace(branchName)

			checkoutCmd := exec.Command("git", "checkout", "-b", branchName, branch)
			checkoutCmd.Dir = repoPath

			var stderr bytes.Buffer
			checkoutCmd.Stderr = &stderr

			err := checkoutCmd.Run()
			if err != nil && !strings.Contains(stderr.String(), "already exists") {
				return fmt.Errorf("failed to checkout branch %s: %w, stderr: %s", branchName, err, stderr.String())
			}
		}
	}

	return nil
}

func analyzeGitHistoryByBranch(repoPath string, fileFilter string) (map[string]*BranchReport, error) {
	cmdBranches := exec.Command("git", "branch", "--format=%(refname:short)")
	cmdBranches.Dir = repoPath
	outputBranches, err := cmdBranches.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("git branch failed: %v, output: %s", err, outputBranches)
	}

	branchNames := strings.Split(string(outputBranches), "\n")
	branchReports := make(map[string]*BranchReport)

	for _, branchName := range branchNames {

		branchName = strings.TrimSpace(branchName)
		if branchName == "" {
			continue
		}

		logRange := branchName

		// Get merge base to get stats from the branch only
		if branchName != defaultMainBranchName {
			cmdMergeBase := exec.Command("git", "merge-base", defaultMainBranchName, branchName)
			cmdMergeBase.Dir = repoPath
			outputMergeBase, err := cmdMergeBase.CombinedOutput()
			if err != nil {
				log.Printf("command 'git merge-base' for branch '%s' failed: %v; message: %s", branchName, err, outputMergeBase)
				log.Printf("using default 'git log' range: %s", logRange)
			} else {
				mergeBase := strings.TrimSpace(string(outputMergeBase))
				logRange = fmt.Sprintf("%s..%s", mergeBase, branchName)
			}
		}

		branchReports[branchName] = &BranchReport{
			BranchName:    branchName,
			Contributions: make(map[string]*UserContribution),
		}

		cmdLog := exec.Command("git", "log", "--pretty=format:%ae,%ad,%H", "--date=short", "--numstat", branchName)
		if fileFilter != "" {
			log.Printf("Applying for branch '%s' filter: %s", branchName, fileFilter)
			cmdLog = exec.Command("git", "log", "--pretty=format:%ae,%ad,%H", "--date=short", "--numstat", logRange, "--", fileFilter)
		}

		//log.Printf("git cmd: %s", cmdLog)

		cmdLog.Dir = repoPath
		outputLog, err := cmdLog.CombinedOutput()
		if err != nil {
			log.Printf("git log for branch %s failed: %v, output: %s", branchName, err, outputLog)
			continue
		}

		linesLog := strings.Split(string(outputLog), "\n")
		var currentCommit string
		var currentDate string
		var currentEmail string

		for _, line := range linesLog {
			if strings.Contains(line, "@") && strings.Contains(line, ",") {
				parts := strings.Split(line, ",")
				if len(parts) >= 3 {
					currentEmail = parts[0]
					currentDate = parts[1]
					currentCommit = parts[2]
					if _, ok := branchReports[branchName].Contributions[currentEmail]; !ok {
						branchReports[branchName].Contributions[currentEmail] = &UserContribution{
							Email:                currentEmail,
							ContributionTimeline: make(map[string]int),
							FileFilter:           fileFilter,
						}
					}
					branchReports[branchName].Contributions[currentEmail].CommitCount++

					dateParsed, err := time.Parse("2006-01-02", currentDate)
					if err == nil {
						if defaultGroupByForLogDate == "month" {
							//yearMonth := fmt.Sprintf("%d-%s", dateParsed.Year(), dateParsed.Month().String())
							//branchReports[branchName].Contributions[currentEmail].ContributionTimeline[yearMonth]++
							yearMonth := fmt.Sprintf("%d-%s", dateParsed.Year(), dateParsed.Month().String()[:3])
							branchReports[branchName].Contributions[currentEmail].ContributionTimeline[strings.ToUpper(yearMonth)]++
						} else {
							_, week := dateParsed.ISOWeek()
							yearWeek := fmt.Sprintf("%d-%02d", dateParsed.Year(), week)
							branchReports[branchName].Contributions[currentEmail].ContributionTimeline[yearWeek]++
						}
					}
				}
			} else if strings.Contains(line, "\t") && currentCommit != "" {
				parts := strings.Split(line, "\t")
				if len(parts) == 3 && parts[0] != "-" && parts[1] != "-" {
					added, _ := strconv.Atoi(parts[0])
					removed, _ := strconv.Atoi(parts[1])
					branchReports[branchName].Contributions[currentEmail].LinesAdded += added
					branchReports[branchName].Contributions[currentEmail].LinesRemoved += removed
					branchReports[branchName].Contributions[currentEmail].LinesEdited += added + removed
				}
			}
		}
	}

	// Remove empty branch reports
	for branchName, report := range branchReports {
		if len(report.Contributions) == 0 {
			delete(branchReports, branchName)
		}
	}

	return branchReports, nil
}

func generateHTMLReportByBranch(branchReports map[string]*BranchReport, repoName string, fileFilter string) (string, error) {
	tmpl := `
<!DOCTYPE html>
<html lang="en" data-bs-theme="dark">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Git Contribution Report: {{.RepoName}}</title>
<link href="https://cdn.jsdelivr.net/npm/bootstrap@5.3.2/dist/css/bootstrap.min.css" rel="stylesheet">
<script src="https://cdn.jsdelivr.net/npm/bootstrap@5.3.2/dist/js/bootstrap.bundle.min.js"></script>
<style>
	.fixed-width {
		width: 150px;
	}
</style>
</head>
<body>

<div class="container mt-4">

<h4> Repository name: <span class="badge text-bg-success">{{.RepoName}}</span></h4>
<h4> Applied file filter: <span class="badge text-bg-info">{{.FileFilter}}</span></h4>

<div class="d-flex justify-content-end mb-3">
	<button id="themeToggle" class="btn btn-outline-light">Light Theme</button>
</div>

{{range $branchName, $branchReport := .BranchReports}}
<h4> Branch: <span class="badge text-bg-warning">{{$branchName}}</span></h4>

<table class="table table-dark table-striped">
	<thead>
		<tr>
			<th class="fixed-width">Email</th>
			<th class="fixed-width">Commit Count</th>
			<th class="fixed-width">Contribution Timeline</th>
			<th>Lines Added</th>
			<th>Lines Removed</th>
			<th>Lines Edited</th>
			<th>File Filter</th>
		</tr>
	</thead>
	<tbody>
		{{range sortContributions .Contributions}}
		<tr>
			<td>{{.Email}}</td>
			<td>{{.CommitCount}}</td>
			<td>
				{{range $yearWeek, $count := .ContributionTimeline}}
					{{$yearWeek}}: {{$count}}<br>
				{{end}}
			</td>
			<td>{{.LinesAdded}}</td>
			<td>{{.LinesRemoved}}</td>
			<td>{{.LinesEdited}}</td>
			<td>{{.FileFilter}}</td>
		</tr>
		{{end}}
	</tbody>
</table>
{{end}}
</div>

<script>
const themeToggle = document.getElementById('themeToggle');
let currentTheme = 'dark';

themeToggle.addEventListener('click', () => {
	if (currentTheme === 'dark') {
		document.documentElement.setAttribute('data-bs-theme', 'light');
		document.querySelectorAll('table').forEach(table => {
			table.classList.remove('table-dark');
		});
		themeToggle.textContent = 'Dark Theme';
		currentTheme = 'light';
	} else {
		document.documentElement.setAttribute('data-bs-theme', 'dark');
		document.querySelectorAll('table').forEach(table => {
			table.classList.add('table-dark');
		});
		themeToggle.textContent = 'Light Theme';
		currentTheme = 'dark';
	}
});

</script>
</body>
</html>
`
	t, err := template.New("report").Funcs(template.FuncMap{
		"sortContributions": func(contributions map[string]*UserContribution) []*UserContribution {
			sorted := make([]*UserContribution, 0, len(contributions))
			for _, c := range contributions {
				sorted = append(sorted, c)
			}
			sort.Slice(sorted, func(i, j int) bool {
				return sorted[i].LinesAdded > sorted[j].LinesAdded // Sort by LinesAdded descending
			})
			return sorted
		},
	}).Parse(tmpl)

	if err != nil {
		return "", err
	}

	data := ReportData{
		RepoName:      repoName,
		FileFilter:    fileFilter,
		BranchReports: branchReports,
	}

	var buf bytes.Buffer
	err = t.Execute(&buf, data)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}
