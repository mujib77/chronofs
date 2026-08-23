package main

import (
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/mujib77/chronofs/internal/engine"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		return
	}

	switch os.Args[1] {
	case "demo":
		runDemo()
	default:
		printUsage()
	}
}

func printUsage() {
	fmt.Println("⏱  ChronoFS")
	fmt.Println("The sub-millisecond time-scrubbing file system")
	fmt.Println("\nUsage:")
	fmt.Println("  chronofs demo")
	fmt.Println("  chronofs mount <directory>       (coming soon)")
	fmt.Println("  chronofs rewind --seconds <n>    (coming soon)")
}

func runDemo() {
	fs := engine.New()

	fmt.Println("⏱  ChronoFS — timeline demo")
	fmt.Println("\n[ t = 0s ] Creating production files...")

	fs.WriteFile("app/server.go", []byte("package app"))
    fs.WriteFile("assets/logo.svg", []byte("<svg>ChronoFS</svg>"))
    fs.WriteFile("config/prod.env", []byte("ENV=production"))

  beforeDelete := time.Now()

      printFiles(fs)
    time.Sleep(800 * time.Millisecond)
	fmt.Println("\n[ t = +0.8s ] $ rm -rf *")
	for _, path := range []string{
		"app/server.go",
		"assets/logo.svg",
		"config/prod.env",
	} {
		if err := fs.DeleteFile(path); err != nil {
			fmt.Println("Error:", err)
			return
		}
	}

	fmt.Println("✓ Folder is empty")
	printFiles(fs)

	time.Sleep(800 * time.Millisecond)

	fmt.Println("\n[ SCRUBBING ← ] Rewinding to the moment before deletion...")
	fs.Rewind(beforeDelete)

	fmt.Println("✓ Historical state restored")
	printFiles(fs)

	fmt.Println("\n⚡ ChronoFS: deletion is now a reversible point in time.")
}

func printFiles(fs *engine.Engine) {
	files := fs.ListFiles()
	paths := make([]string, 0, len(files))

	for path := range files {
		paths = append(paths, path)
	}

	sort.Strings(paths)

	if len(paths) == 0 {
		fmt.Println("   (empty)")
		return
	}

	for _, path := range paths {
		fmt.Printf("   ✓ %s\n", path)
	}
}