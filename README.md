## About

Minimalistic CLI utility to analyze  Git history of a specified repository and generate HTML report detailing user contributions in each branch. 
The report includes:

-   Commit count
-   Contribution timeline (grouped by `week` or `month`)
-   Total lines added
-   Total lines removed
-   Total lines edited

The utility processes each branch in the repository and provides a summary report for each git branch.
The path to the Git repository is provided as a command-line argument. 
The output is saved as HTML file named `report_REPO-NAME_DATE_TIME.html`.

### CLI Parameters

Here are essential CLI parameters of the utility:

* `--repository` - Path to the git repository (directory or URL)
* `--filter` - Filter for file types (e.g., go, py, etc.). Optional
* `--mainbranch` - Name of the 'main' branch for merge-base (default "main")
* `--groupby` -  Group git log date by 'week' or 'month' (default "month")
* `--help` - Show help message 

**NOTE:** In order to fetch history of remote git branches, they must be pulled into local repository. This should be done automatically by the utility, if URL is used.

## Run with Go

1. Install Go:
    - If you don't have Go installed, download and install it from [golang.org](https://golang.org/dl/)
1. Clone the repository (or copy the code):
1. Run the executable:
    ```bash
    go run . --help
    ```

## Usage Example

To generate the HTML report, run the executable with the `--repository` flag specifying the path to your Git repository:

```bash
go run . --repository /path/to/your/git/repository
```

## Report Example

Run options:
```
go run . --repositorysitory ../sourcecodesnippets --mainbranch master --filter *.yml
```

Screenshot:

![alt text](docs/report-example-ui.png)

## License

[MIT](LICENSE)