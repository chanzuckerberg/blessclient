package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/tools/cover"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

const (
	star             = "*"
	defaultFileFlags = 0644
)

var reportOutputPath = flag.String("output", "coverage.html", "the path to write the full html coverage report")
var temporaryOutputPath = flag.String("tmp", "coverage.cov", "the path to write the intermediate results")
var update = flag.Bool("update", false, "if we should write the current coverage to `COVERAGE` files")
var enforce = flag.Bool("enforce", false, "if we should enforce coverage minimums defined in `COVERAGE` files")
var include = flag.String("include", "", "the include file filter in glob form, can be a csv.")
var exclude = flag.String("exclude", "", "the exclude file filter in glob form, can be a csv.")

func main() {
	flag.Parse()

	pwd, err := os.Getwd()
	maybeFatal(err)

	fmt.Fprintln(os.Stdout, "coverage starting")
	fullCoverageData, err := removeAndOpen(*temporaryOutputPath)
	if err != nil {
		maybeFatal(err)
	}
	fmt.Fprintln(fullCoverageData, "mode: set")

	maybeFatal(filepath.Walk("./", func(currentPath string, info os.FileInfo, err error) error {
		if os.IsNotExist(err) {
			return nil
		}
		if err != nil {
			return err
		}

		if !info.IsDir() {
			return nil
		}
		if info.Name() == ".git" {
			return filepath.SkipDir
		}
		if strings.HasPrefix(info.Name(), "_") {
			return filepath.SkipDir
		}
		if info.Name() == "vendor" {
			return filepath.SkipDir
		}
		if !dirHasGlob(currentPath, "*.go") {
			return nil
		}

		if len(*include) > 0 {
			if matches := globAnyMatch(*include, currentPath); !matches {
				return nil
			}
		}

		if len(*exclude) > 0 {
			if matches := globAnyMatch(*exclude, currentPath); matches {
				return nil
			}
		}

		packageCoverReport := filepath.Join(currentPath, "profile.cov")
		err = removeIfExists(packageCoverReport)
		if err != nil {
			return err
		}

		var output []byte
		output, err = execCoverage(currentPath)
		if err != nil {
			fmt.Fprintln(os.Stderr, string(output))
			return err
		}

		coverage := extractCoverage(string(output))
		fmt.Fprintf(os.Stdout, "%s: %v%%\n", currentPath, coverage)

		if enforce != nil && *enforce {
			err = enforceCoverage(currentPath, coverage)
			if err != nil {
				return err
			}
		}

		if update != nil && *update {
			fmt.Fprintf(os.Stdout, "%s updating coverage\n", currentPath)
			err = writeCoverage(currentPath, coverage)
			if err != nil {
				return err
			}
		}

		err = mergeCoverageOutput(packageCoverReport, fullCoverageData)
		if err != nil {
			return err
		}

		err = removeIfExists(packageCoverReport)
		if err != nil {
			return err
		}

		return nil
	}))

	maybeFatal(fullCoverageData.Close())

	covered, total, err := parseFullCoverProfile(pwd, *temporaryOutputPath)
	maybeFatal(err)
	finalCoverage := (float64(covered) / float64(total)) * 100
	maybeFatal(writeCoverage(pwd, fmt.Sprintf("%.2f", finalCoverage)))
	fmt.Fprintf(os.Stdout, "final coverage: %.2f\n", finalCoverage)
	fmt.Fprintf(os.Stdout, "merging coverage output: %s\n", *reportOutputPath)
	maybeFatal(removeIfExists(*temporaryOutputPath))
	fmt.Fprintln(os.Stdout, "coverage complete")
}

// --------------------------------------------------------------------------------
// utilities
// --------------------------------------------------------------------------------

// globIncludeMatch tests if a file matches a (potentially) csv of glob filters.
func globAnyMatch(filter, file string) bool {
	parts := strings.Split(filter, ",")
	for _, part := range parts {
		if matches := glob(strings.TrimSpace(part), file); matches {
			return true
		}
	}
	return false
}

func glob(pattern, subj string) bool {
	// Empty pattern can only match empty subject
	if pattern == "" {
		return subj == pattern
	}

	// If the pattern _is_ a glob, it matches everything
	if pattern == star {
		return true
	}

	parts := strings.Split(pattern, star)

	if len(parts) == 1 {
		// No globs in pattern, so test for equality
		return subj == pattern
	}

	leadingGlob := strings.HasPrefix(pattern, star)
	trailingGlob := strings.HasSuffix(pattern, star)
	end := len(parts) - 1

	// Go over the leading parts and ensure they match.
	for i := 0; i < end; i++ {
		idx := strings.Index(subj, parts[i])

		switch i {
		case 0:
			// Check the first section. Requires special handling.
			if !leadingGlob && idx != 0 {
				return false
			}
		default:
			// Check that the middle parts match.
			if idx < 0 {
				return false
			}
		}

		// Trim evaluated text from subj as we loop over the pattern.
		subj = subj[idx+len(parts[i]):]
	}

	// Reached the last section. Requires special handling.
	return trailingGlob || strings.HasSuffix(subj, parts[end])
}

func enforceCoverage(path, actualCoverage string) error {
	actual, err := strconv.ParseFloat(actualCoverage, 64)
	if err != nil {
		return err
	}

	contents, err := ioutil.ReadFile(filepath.Join(path, "COVERAGE"))
	if err != nil {
		return err
	}
	expected, err := strconv.ParseFloat(strings.TrimSpace(string(contents)), 64)
	if err != nil {
		return err
	}

	if expected == 0 {
		return nil
	}

	if actual < expected {
		return fmt.Errorf(
			"%s fails coverage: %0.2f%% vs. %0.2f%%",
			path, expected, actual,
		)
	}
	return nil
}

func extractCoverage(corpus string) string {
	regex := `coverage: ([0-9,.]+)% of statements`
	expr := regexp.MustCompile(regex)

	results := expr.FindStringSubmatch(corpus)
	if len(results) > 1 {
		return results[1]
	}
	return "0"
}

func writeCoverage(path, coverage string) error {
	return ioutil.WriteFile(filepath.Join(path, "COVERAGE"), []byte(strings.TrimSpace(coverage)), defaultFileFlags)
}

func dirHasGlob(path, glob string) bool {
	files, _ := filepath.Glob(filepath.Join(path, glob))
	return len(files) > 0
}

func gobin() string {
	gobin, err := exec.LookPath("go")
	maybeFatal(err)
	return gobin
}

func execCoverage(path string) ([]byte, error) {
	cmd := exec.Command(gobin(), "test", "-timeout", "10s", "-short", "-covermode=set", "-coverprofile=profile.cov")
	cmd.Dir = path
	return cmd.CombinedOutput()
}

func mergeCoverageOutput(temp string, outFile *os.File) error {
	contents, err := ioutil.ReadFile(temp)
	if err != nil {
		return err
	}

	scanner := bufio.NewScanner(bytes.NewBuffer(contents))

	var skip int
	for scanner.Scan() {
		skip++
		if skip < 2 {
			continue
		}
		_, err = fmt.Fprintln(outFile, scanner.Text())
		if err != nil {
			return err
		}
	}
	return nil
}

func removeIfExists(path string) error {
	if _, err := os.Stat(path); err == nil {
		return os.Remove(path)
	}
	return nil
}

func maybeFatal(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "%+v\n", err)
		os.Exit(1)
	}
}

func removeAndOpen(path string) (*os.File, error) {
	if _, err := os.Stat(path); err == nil {
		if err = os.Remove(path); err != nil {
			return nil, err
		}
	}
	return os.Create(path)
}

func countFileLines(path string) (lines int, err error) {
	if filepath.Ext(path) != ".go" {
		err = fmt.Errorf("count lines path must be a .go file")
		return
	}

	var contents []byte
	contents, err = ioutil.ReadFile(path)
	if err != nil {
		return
	}
	lines = bytes.Count(contents, []byte{'\n'})
	return
}

// joinCoverPath takes a pwd, and a filename, and joins them
// overlaying parts of the suffix of the pwd, and the prefix
// of the filename that match.
// ex:
// - pwd: /foo/bar/baz, filename: bar/baz/buzz.go => /foo/bar/baz/buzz.go
func joinCoverPath(pwd, fileName string) string {
	pwdPath := lessEmpty(strings.Split(pwd, "/"))
	fileDirPath := lessEmpty(strings.Split(filepath.Dir(fileName), "/"))

	for index, dir := range pwdPath {
		if dir == first(fileDirPath) {
			pwdPath = pwdPath[:index]
			break
		}
	}

	return filepath.Join(maybePrefix(strings.Join(pwdPath, "/"), "/"), fileName)
}

// parseFullCoverProfile parses the final / merged cover output.
func parseFullCoverProfile(pwd string, path string) (covered, total int, err error) {
	files, err := cover.ParseProfiles(path)
	if err != nil {
		return
	}

	var fileTotal int
	for _, file := range files {
		fileTotal, err = countFileLines(joinCoverPath(pwd, file.FileName))
		if err != nil {
			return
		}
		for _, block := range file.Blocks {
			covered += (block.EndLine - block.StartLine) + 1
		}
		total += fileTotal
	}

	return
}
func lessEmpty(values []string) (output []string) {
	for _, value := range values {
		if len(value) > 0 {
			output = append(output, value)
		}
	}
	return
}

func first(values []string) (output string) {
	if len(values) == 0 {
		return
	}
	output = values[0]
	return
}

func maybePrefix(root, prefix string) string {
	if strings.HasPrefix(root, prefix) {
		return root
	}
	return prefix + root
}
