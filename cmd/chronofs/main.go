package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/mujib77/chronofs/internal/engine"
	chronofsfs "github.com/mujib77/chronofs/internal/fs"
	"golang.org/x/term"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		return
	}

	switch os.Args[1] {
	case "demo":
		runDemo()
	case "mount":
		if len(os.Args) != 3 {
			fmt.Println("Usage: chronofs mount <drive-letter>")
			return
		}

		mount(os.Args[2])

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

func mount(mountPoint string) {
	fmt.Printf("⏱  Mounting ChronoFS at %s\n", mountPoint)
	fmt.Printf("Open File Explorer and go to %s\\\n", mountPoint)
	fmt.Println("\nControls: scrub | undo | redo | rewind <seconds> | timeline | help")
	fmt.Println("Press Ctrl+C to unmount.")

	if !chronofsfs.Mount(mountPoint, handleMountCommands) {
		fmt.Println("ChronoFS could not mount. Ensure the drive letter is unused.")
	}
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
	printTimeline(fs)
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
func printTimeline(fs *engine.Engine) {
	fmt.Println("\n[ EVENT TIMELINE ]")

	for _, event := range fs.Events() {
		path := event.Path

		if event.Type == engine.EventRename {
			path = event.OldPath + " → " + event.Path
		}

		fmt.Printf(
			"  %s  %-6s  %s\n",
			event.Timestamp.Format("15:04:05.000000"),
			event.Type,
			path,
		)
	}
}

func handleMountCommands(fs *chronofsfs.FileSystem) {
	scanner := bufio.NewScanner(os.Stdin)

	for scanner.Scan() {
		parts := strings.Fields(scanner.Text())
		if len(parts) == 0 {
			continue
		}

		switch parts[0] {
		case "scrub":
			runScrubber(fs)

		case "undo":
			fileCount, restored := fs.Undo()

			if !restored {
				fmt.Println("Nothing earlier exists in the timeline.")
				continue
			}

			fmt.Printf("⏪ Undo complete — restored state has %d file(s).\n", fileCount)
			fmt.Println("Refresh File Explorer with F5 to view the restored state.")

		case "redo":
			fileCount, advanced := fs.Redo()

			if !advanced {
				fmt.Println("Already at the newest moment in the timeline.")
				continue
			}

			fmt.Printf("⏩ Redo complete — current state has %d file(s).\n", fileCount)

		case "rewind":
			if len(parts) != 2 {
				fmt.Println("Usage: rewind <seconds>")
				continue
			}

			seconds, err := strconv.Atoi(parts[1])
			if err != nil || seconds <= 0 {
				fmt.Println("Seconds must be a positive whole number.")
				continue
			}

			fileCount := fs.Rewind(seconds)
			fmt.Printf("⏪ Rewound %d seconds — restored state has %d file(s).\n", seconds, fileCount)
			fmt.Println("Refresh File Explorer with F5 to view the restored state.")

		case "timeline":
			fmt.Println("\n[ EVENT TIMELINE ]")
			for _, event := range fs.Events() {
				fmt.Printf(
					"  %s  %-6s  %s\n",
					event.Timestamp.Format("15:04:05.000000"),
					event.Type,
					event.Path,
				)
			}

		case "help":
			fmt.Println("Commands: scrub | undo | redo | rewind <seconds> | timeline | help")

		default:
			fmt.Println("Unknown command. Type help.")
		}
	}
	if err := scanner.Err(); err != nil {
		fmt.Println("Mount console input error:", err)
	}
}

func runScrubber(fs *chronofsfs.FileSystem) {
	fd := int(os.Stdin.Fd())

	if !term.IsTerminal(fd) {
		fmt.Println("Scrubber requires an interactive terminal.")
		return
	}

	oldState, err := term.MakeRaw(fd)
	if err != nil {
		fmt.Println("Could not activate scrubber:", err)
		return
	}
	defer term.Restore(fd, oldState)

	renderScrubber(fs)

	for {
		key := make([]byte, 1)

		if _, err := os.Stdin.Read(key); err != nil {
			return
		}

		switch key[0] {
		case 'q', 'Q':
			fmt.Print("\r\n")
			return

		case 'a', 'A':
			fs.Undo()
			renderScrubber(fs)

		case 'd', 'D':
			fs.Redo()
			renderScrubber(fs)

		case 27:
			sequence := make([]byte, 2)

			if _, err := io.ReadFull(os.Stdin, sequence); err != nil {
				return
			}

			if sequence[0] != '[' {
				continue
			}

			switch sequence[1] {
			case 'D':
				fs.Undo()
				renderScrubber(fs)

			case 'C':
				fs.Redo()
				renderScrubber(fs)
			}
		}
	}
}

func renderScrubber(fs *chronofsfs.FileSystem) {
	cursor, total := fs.TimelinePosition()

	const width = 40
	marker := 0

	if total > 1 {
		marker = cursor * (width - 1) / (total - 1)
	}

	var bar strings.Builder

	for index := 0; index < width; index++ {
		if index == marker {
			bar.WriteString("●")
		} else {
			bar.WriteString("─")
		}
	}

	fmt.Print("\033[2J\033[H")
	fmt.Println("⏱  CHRONOFS — LIVE TIME SCRUBBER")
	fmt.Println()
	fmt.Printf("  Past  [%s]  Present\n", bar.String())
	fmt.Printf("  Snapshot %d of %d\n", cursor+1, total)
	fmt.Println()
	fmt.Println("  ← / A  travel backward")
	fmt.Println("  → / D  travel forward")
	fmt.Println("  Q      exit scrubber")
	fmt.Println()
	fmt.Println("  Watch File Explorer reconstruct itself in real time.")
}
