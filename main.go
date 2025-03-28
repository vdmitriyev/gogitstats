package main

import (
	"bytes"
	"flag"
	"fmt"
	"html/template"
	"log"
	"os"
	"os/exec"
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
}

type BranchReport struct {
	BranchName    string
	Contributions map[string]*UserContribution
}

func main() {
	repoPath := flag.String("repo", "", "Path to the git repository")
	flag.Parse()

	if *repoPath == "" {
		log.Fatal("Please provide the path to the git repository using --repo")
	}

	branchReports, err := analyzeGitHistoryByBranch(*repoPath)
	if err != nil {
		log.Fatalf("Error analyzing git history: %v", err)
	}

	htmlReport, err := generateHTMLReportByBranch(branchReports)
	if err != nil {
		log.Fatalf("Error generating HTML report: %v", err)
	}

	filename := fmt.Sprintf("reports_%s.html", time.Now().Format("20060102150405"))
	err = os.WriteFile(filename, []byte(htmlReport), 0644)
	if err != nil {
		log.Fatalf("Error writing HTML report to file: %v", err)
	}

	fmt.Printf("HTML report generated: %s\n", filename)
}

func analyzeGitHistoryByBranch(repoPath string) (map[string]*BranchReport, error) {
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
<html>
<head>
<title>Git Contribution Report by Branch</title>
<style>
table {
        border-collapse: collapse;
        width: 100%;
}
th, td {
        border: 1px solid #dddddd;
        text-align: left;
        padding: 8px;
}
tr:nth-child(even) {
        background-color: #f2f2f2;
}
</style>
</head>
<body>

{{range $branchName, $branchReport := .}}
<h2>Contribution Report for Branch: {{$branchName}}</h2>

<table>
<tr>
        <th>Email</th>
        <th>Commit Count</th>
        <th>Contribution Timeline</th>
        <th>Lines Added</th>
        <th>Lines Removed</th>
        <th>Lines Edited</th>
</tr>
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
</tr>
{{end}}
</table>
{{end}}

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
