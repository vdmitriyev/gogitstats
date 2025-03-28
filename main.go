package main

import (
	"bytes"
	"flag"
	"fmt"
	"html/template"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

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

type customLogWriter struct {
}

func (writer customLogWriter) Write(bytes []byte) (int, error) {
	return fmt.Print(time.Now().UTC().Format("2006-01-02 15:04:05") + " " + string(bytes))
}

func main() {
	log.SetFlags(0)
	log.SetOutput(new(customLogWriter))

	repoPath := flag.String("repo", "", "Path to the git repository")
	fileFilter := flag.String("filefilter", "", "Optional file type (as extension) to filter by (e.g., go, py, etc.)")
	flag.Parse()

	if *repoPath == "" {
		log.Fatal("Please provide the path to the git repository using --repo")
	}

	if _, err := os.Stat(*repoPath); os.IsNotExist(err) {
		log.Fatalf("Repository path does not exist: %s", *repoPath)
	}

	repoName := filepath.Base(*repoPath)

	log.Printf("Analyzing repository: %s", repoName)

	branchReports, err := analyzeGitHistoryByBranch(*repoPath, *fileFilter)
	if err != nil {
		log.Fatalf("Error analyzing git history: %v", err)
	}

	htmlReport, err := generateHTMLReportByBranch(branchReports)
	if err != nil {
		log.Fatalf("Error generating HTML report: %v", err)
	}

	filename := fmt.Sprintf("report_%s_%s.html", repoName, time.Now().Format("2006-01-02_1504"))
	err = os.WriteFile(filename, []byte(htmlReport), 0644)
	if err != nil {
		log.Fatalf("Error writing HTML report to file: %v", err)
	}

	log.Printf("HTML report generated: %s\n", filename)
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

		branchReports[branchName] = &BranchReport{
			BranchName:    branchName,
			Contributions: make(map[string]*UserContribution),
		}

		cmdLog := exec.Command("git", "log", "--pretty=format:%ae,%ad,%H", "--date=short", "--numstat", branchName)
		if fileFilter != "" {
			log.Printf("Applying for branch '%s' filter: %s", branchName, fileFilter)
			cmdLog = exec.Command("git", "log", "--pretty=format:%ae,%ad,%H", "--date=short", "--numstat", branchName, "--", fileFilter)
		}

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
						_, week := dateParsed.ISOWeek()
						yearWeek := fmt.Sprintf("%d-%02d", dateParsed.Year(), week)
						branchReports[branchName].Contributions[currentEmail].ContributionTimeline[yearWeek]++
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
	return branchReports, nil
}

func generateHTMLReportByBranch(branchReports map[string]*BranchReport) (string, error) {
	tmpl := `
<!DOCTYPE html>
<html lang="en" data-bs-theme="dark">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Git Contribution Report by Branch</title>
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
<div class="d-flex justify-content-end mb-3">
	<button id="themeToggle" class="btn btn-outline-light">Light Theme</button>
</div>

{{range $branchName, $branchReport := .}}
<h3> Branch: <span class="badge text-bg-warning">{{$branchName}}</span></h3>

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

	var buf bytes.Buffer
	err = t.Execute(&buf, branchReports)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}
