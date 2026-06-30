package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

// parseChoice reads a 1-based integer from r, validated in [1, max].
func parseChoice(r io.Reader, max int) (int, error) {
	scanner := bufio.NewScanner(r)
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return 0, fmt.Errorf("reading input: %w", err)
		}
		return 0, fmt.Errorf("no input")
	}
	text := strings.TrimSpace(scanner.Text())
	if text == "" {
		return 0, fmt.Errorf("enter a number between 1 and %d", max)
	}
	n, err := strconv.Atoi(text)
	if err != nil || n < 1 || n > max {
		return 0, fmt.Errorf("enter a number between 1 and %d", max)
	}
	return n, nil
}

func pickProduct(products []Product) (Product, error) {
	if len(products) == 0 {
		return Product{}, fmt.Errorf("no products available")
	}
	fmt.Fprintln(os.Stderr, "\nSelect a product:")
	for i, p := range products {
		fmt.Fprintf(os.Stderr, "  %2d. %s\n", i+1, p.Name)
	}
	fmt.Fprint(os.Stderr, "\nChoice: ")
	n, err := parseChoice(os.Stdin, len(products))
	if err != nil {
		return Product{}, err
	}
	return products[n-1], nil
}

func pickLanguage(langs []Language) (Language, error) {
	if len(langs) == 0 {
		return Language{}, fmt.Errorf("no languages available")
	}
	fmt.Fprintln(os.Stderr, "\nSelect a language:")
	for i, l := range langs {
		fmt.Fprintf(os.Stderr, "  %2d. %s\n", i+1, l.Language)
	}
	fmt.Fprint(os.Stderr, "\nChoice: ")
	n, err := parseChoice(os.Stdin, len(langs))
	if err != nil {
		return Language{}, err
	}
	return langs[n-1], nil
}

func pickEvalProduct(products []EvalProduct) (EvalProduct, error) {
	if len(products) == 0 {
		return EvalProduct{}, fmt.Errorf("no eval products available")
	}
	fmt.Fprintln(os.Stderr, "\nSelect an evaluation product:")
	for i, p := range products {
		fmt.Fprintf(os.Stderr, "  %2d. %s\n", i+1, p.Name)
	}
	fmt.Fprint(os.Stderr, "\nChoice: ")
	n, err := parseChoice(os.Stdin, len(products))
	if err != nil {
		return EvalProduct{}, err
	}
	return products[n-1], nil
}

func isTerminal() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

func pickCombined() (isEval bool, p Product, ep EvalProduct, err error) {
	total := len(consumerProducts) + len(evalProducts)
	fmt.Fprintln(os.Stderr, "\nSelect a product:")
	for i, prod := range consumerProducts {
		fmt.Fprintf(os.Stderr, "  %2d. %s\n", i+1, prod.Name)
	}
	fmt.Fprintln(os.Stderr, "\n      ── Evaluation / Enterprise ──")
	offset := len(consumerProducts)
	for i, e := range evalProducts {
		fmt.Fprintf(os.Stderr, "  %2d. %s\n", offset+i+1, e.Name)
	}
	fmt.Fprint(os.Stderr, "\nChoice: ")
	n, err := parseChoice(os.Stdin, total)
	if err != nil {
		return false, Product{}, EvalProduct{}, err
	}
	if n <= offset {
		return false, consumerProducts[n-1], EvalProduct{}, nil
	}
	return true, Product{}, evalProducts[n-offset-1], nil
}

func pickArchitecture(links []DownloadLink) (DownloadLink, error) {
	fmt.Fprintln(os.Stderr, "Select architecture:")
	for i, l := range links {
		label := l.Architecture
		if label == "" {
			label = fmt.Sprintf("link %d", i+1)
		}
		fmt.Fprintf(os.Stderr, "  %2d. %s\n", i+1, label)
	}
	fmt.Fprint(os.Stderr, "\nChoice: ")
	n, err := parseChoice(os.Stdin, len(links))
	if err != nil {
		return DownloadLink{}, err
	}
	return links[n-1], nil
}

func openInBrowser(rawURL string) error {
	switch runtime.GOOS {
	case "windows":
		// Write URL to a temp .url file (Windows Internet Shortcut format).
		// The URL never touches cmd.exe — only the temp filename does, which
		// contains no special characters.
		f, err := os.CreateTemp("", "msdl-*.url")
		if err != nil {
			return err
		}
		name := f.Name()
		defer os.Remove(name)
		if _, err := fmt.Fprintf(f, "[InternetShortcut]\r\nURL=%s\r\n", rawURL); err != nil {
			f.Close()
			return err
		}
		f.Close()
		cmdExe := os.Getenv("COMSPEC")
		if cmdExe == "" {
			cmdExe = `C:\Windows\System32\cmd.exe`
		}
		return exec.Command(cmdExe, "/c", "start", "", name).Run()
	case "darwin":
		return exec.Command("open", rawURL).Run()
	default:
		return exec.Command("xdg-open", rawURL).Run()
	}
}

func copyToClipboard(text string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		clipExe := `C:\Windows\System32\clip.exe`
		if sr := os.Getenv("SystemRoot"); sr != "" {
			clipExe = sr + `\System32\clip.exe`
		}
		cmd = exec.Command(clipExe)
	case "darwin":
		cmd = exec.Command("pbcopy")
	default:
		if _, err := exec.LookPath("xclip"); err == nil {
			cmd = exec.Command("xclip", "-selection", "clipboard")
		} else {
			cmd = exec.Command("xsel", "--clipboard", "--input")
		}
	}
	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
}

func postFetchMenu(uri string) {
	fmt.Fprint(os.Stderr, "\n  [O] Open in browser   [C] Copy to clipboard   [Enter] Exit\n  > ")
	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		return
	}
	switch strings.ToLower(strings.TrimSpace(scanner.Text())) {
	case "o":
		if err := openInBrowser(uri); err != nil {
			fmt.Fprintln(os.Stderr, "  Could not open browser:", err)
		}
	case "c":
		if err := copyToClipboard(uri); err != nil {
			fmt.Fprintln(os.Stderr, "  Could not copy to clipboard:", err)
		} else {
			fmt.Fprintln(os.Stderr, "  ✓ Copied to clipboard")
		}
	}
}
