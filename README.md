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
The output is saved as an HTML file named `reports_DATETIME.html`.

## Installation with Go

1.  **Install Go:** If you don't have Go installed, download and install it from [golang.org](https://golang.org/dl/).
2.  **Clone the repository (or copy the code):**

    ```bash
    git clone <your-repository-url> # If you have it in a repo
    # or just create a main.go file and paste the code.
    ```

3.  **Navigate to the project directory:**

    ```bash
    cd <your-project-directory>
    ```

4.  **Build the executable:**

    ```bash
    go run .
    ```

## Usage Example

To generate the HTML report, run the executable with the `--repo` flag specifying the path to your Git repository:

```bash
go run . --repo /path/to/your/git/repository
```

## License

MIT