## About

Minimalistic CLI utility to analyze the Git history a specified repository and generate an HTML report detailing user contributions in each branch. 
The report includes:

-   Commit count
-   Contribution timeline (grouped by year and week)
-   Total lines added
-   Total lines removed
-   Total lines edited

The utility processes each branch in the repository and provides a separate table in the report for each branch. 
The path to the Git repository is provided as a command-line argument. 
The output is saved as an HTML file named `report_REPO-NAME_DATE_TIME.html`.

CLI parameters:

* `--repo` - Path to the git repository
* `--filter` - Filter for file types (e.g., go, py, etc.). Optional
* `--mainbranch` - Name of the 'main' branch for merge-base (default "main")
* `--groupby` -  Group git log date by 'week' or 'month' (default "month")

**NOTE:** In order to fetch history of remote git branches, they must be pulled into local repository.

## Run with Go

1. Install Go:
    - If you don't have Go installed, download and install it from [golang.org](https://golang.org/dl/)
1. Clone the repository (or copy the code):
1. Run the executable:
    ```bash
    go run . --help
    ```

## Usage Example

To generate the HTML report, run the executable with the `--repo` flag specifying the path to your Git repository:

```bash
go run . --repo /path/to/your/git/repository
```

## Report Example

Run options:
```
go run . --repo ../sourcecodesnippets --mainbranch master --filter *.yml
```

Screenshot:

![alt text](docs/report-example-ui.png)

## License

[MIT](LICENSE)