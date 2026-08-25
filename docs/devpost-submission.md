# ChronoFS — The Sub-Millisecond Time-Scrubbing File System

## Tagline

Delete the project. Scrub time backward. Watch it rebuild itself in File Explorer.

## Inspiration

I wanted to build something around one painful developer moment: deleting the wrong files and instantly regretting it.

Usually the recovery story is stressful. You look for backups, Git history, recycle bins, or recovery tools. But what if deletion was not the end of the story? What if the filesystem itself remembered the moments before the mistake?

That question became ChronoFS.

## What it does

ChronoFS is a Windows user-space filesystem that mounts as a normal drive in File Explorer.

You can create files, edit them, create folders, rename files, and delete an entire project. ChronoFS records every filesystem state in an in-memory timeline.

Then you can open its terminal scrubber and move through time:

- Left Arrow moves backward through filesystem history.
- Right Arrow moves forward again.
- Deleted files and folders reappear directly inside Windows File Explorer.
- Moving forward makes them disappear again.
- `undo`, `redo`, and `rewind` commands are also available.

The main demo is simple: open the mounted drive, delete a full project, launch the scrubber, and watch the project reconstruct itself live.

## How I built it

ChronoFS is written in Go.

I used WinFsp and cgofuse to mount ChronoFS as a native Windows drive. The filesystem handlers capture file writes, truncates, renames, deletes, and directory operations.

The state lives in an in-memory snapshot engine. Every change creates a timestamped historical state. A timeline cursor moves backward or forward through those states.

When the cursor changes, ChronoFS sends Windows file-change notifications through WinFsp. That is what makes File Explorer update automatically when a deleted file or folder is restored.

## Challenges I ran into

The hardest part was not building an in-memory filesystem. It was making the filesystem feel real in Windows.

I had to handle file creation, writing, truncation, deletion, directory creation, directory removal, and Explorer refresh behavior. I also had to make sure that moving backward and forward through history kept the filesystem and Explorer view synchronized.

Another challenge was the timeline itself. A one-way undo system was not enough for a real scrubber. I added a bidirectional cursor so the filesystem can move backward and forward through snapshots.

## Accomplishments that I'm proud of

- Mounting ChronoFS as a real Windows drive.
- Creating and deleting files through normal Windows tools.
- Restoring deleted files directly in File Explorer.
- Restoring deleted folders directly in File Explorer.
- Making Explorer update automatically without pressing F5.
- Building an interactive Arrow-key time scrubber.
- Turning a scary `Remove-Item -Recurse -Force` moment into a visual recovery demo.

## What I learned

I learned how user-space filesystems work, how WinFsp connects a Go filesystem to Windows, and how much detail exists behind normal file operations.

I also learned that a strong demo is not only about the feature. It is about making the feature visible. Watching files reconstruct themselves in Explorer makes the idea instantly understandable.

## What's next for ChronoFS

This prototype keeps its history in memory during a mount session.

The next version would add:

- Persistent disk-backed timelines.
- Memory limits and snapshot retention policies.
- Searchable timeline events.
- A graphical timeline window.
- Restore points and named checkpoints.
- Support for larger projects.
- Optional encrypted history.
- Safer recovery workflows for real development folders.

## Built with

- Go
- WinFsp
- cgofuse
- Windows File Explorer
- PowerShell